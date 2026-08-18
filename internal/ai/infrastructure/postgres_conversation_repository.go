package infrastructure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/ai/domain"
)

type PostgresConversationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresConversationRepository(pool *pgxpool.Pool) *PostgresConversationRepository {
	return &PostgresConversationRepository{pool: pool}
}

var _ domain.ConversationRepository = (*PostgresConversationRepository)(nil)

func scanConversation(row pgx.Row) (*domain.Conversation, error) {
	var (
		id, visitorID        string
		userID               *string
		createdAt, updatedAt time.Time
	)
	if err := row.Scan(&id, &visitorID, &userID, &createdAt, &updatedAt); err != nil {
		return nil, err
	}
	return &domain.Conversation{ID: id, VisitorID: visitorID, UserID: userID, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

// GetOrCreateByVisitor: a message within the last 24h of the visitor's most
// recent conversation counts as the same session and appends there;
// anything older starts a fresh conversation — otherwise one long-idle
// return visitor would accumulate every question ever asked into one
// endless thread.
func (r *PostgresConversationRepository) GetOrCreateByVisitor(ctx context.Context, visitorID string, userID *string) (*domain.Conversation, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, visitor_id, user_id, created_at, updated_at
		FROM ai_conversations
		WHERE visitor_id = $1 AND updated_at > now() - interval '24 hours'
		ORDER BY updated_at DESC
		LIMIT 1`,
		visitorID,
	)
	conv, err := scanConversation(row)
	if err == nil {
		// Attach user_id opportunistically if it wasn't captured before
		// (the visitor logged in mid-conversation) — never clear one
		// that's already set.
		if userID != nil && conv.UserID == nil {
			if _, updErr := r.pool.Exec(ctx, `UPDATE ai_conversations SET user_id = $1 WHERE id = $2`, *userID, conv.ID); updErr != nil {
				return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (attach user): %w", updErr)
			}
			conv.UserID = userID
		}
		return conv, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (query): %w", err)
	}

	row = r.pool.QueryRow(ctx, `
		INSERT INTO ai_conversations (visitor_id, user_id)
		VALUES ($1, $2)
		RETURNING id, visitor_id, user_id, created_at, updated_at`,
		visitorID, userID,
	)
	conv, err = scanConversation(row)
	if err != nil {
		return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (insert): %w", err)
	}
	return conv, nil
}

func (r *PostgresConversationRepository) AppendMessage(ctx context.Context, msg *domain.ConversationMessage) (*domain.ConversationMessage, error) {
	var productsJSON []byte
	if msg.ProductsShown != nil {
		var err error
		productsJSON, err = json.Marshal(msg.ProductsShown)
		if err != nil {
			return nil, fmt.Errorf("ai conversation repository appendMessage (marshal products): %w", err)
		}
	}

	var inputTokens, outputTokens, cacheHitTokens *int
	if msg.Usage != nil {
		inputTokens = &msg.Usage.InputTokens
		outputTokens = &msg.Usage.OutputTokens
		cacheHitTokens = &msg.Usage.CacheHitTokens
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("ai conversation repository appendMessage (begin tx): %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row := tx.QueryRow(ctx, `
		INSERT INTO ai_messages (conversation_id, role, content, blocked_reason, provider_id, model_id, input_tokens, output_tokens, cache_hit_tokens, cost_usd, products_shown)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at`,
		msg.ConversationID, string(msg.Role), msg.Content, msg.BlockedReason, msg.ProviderID, msg.ModelID,
		inputTokens, outputTokens, cacheHitTokens, msg.CostUSD, productsJSON,
	)
	var id string
	var createdAt time.Time
	if err := row.Scan(&id, &createdAt); err != nil {
		return nil, fmt.Errorf("ai conversation repository appendMessage (insert): %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE ai_conversations SET updated_at = now() WHERE id = $1`, msg.ConversationID); err != nil {
		return nil, fmt.Errorf("ai conversation repository appendMessage (touch conversation): %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("ai conversation repository appendMessage (commit): %w", err)
	}

	out := *msg
	out.ID = id
	out.CreatedAt = createdAt
	return &out, nil
}

func (r *PostgresConversationRepository) ListRecentMessages(ctx context.Context, conversationID string, limit int) ([]*domain.ConversationMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, role, content, created_at
		FROM ai_messages
		WHERE conversation_id = $1 AND role IN ('user', 'assistant') AND blocked_reason IS NULL
		ORDER BY created_at DESC
		LIMIT $2`,
		conversationID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("ai conversation repository listRecentMessages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.ConversationMessage
	for rows.Next() {
		var m domain.ConversationMessage
		var role string
		if err := rows.Scan(&m.ID, &m.ConversationID, &role, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("ai conversation repository listRecentMessages (scan): %w", err)
		}
		m.Role = domain.Role(role)
		messages = append(messages, &m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ai conversation repository listRecentMessages (rows): %w", err)
	}

	// Rows come back newest-first (for LIMIT to keep the most recent N);
	// reverse to chronological order before handing to the model.
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}
