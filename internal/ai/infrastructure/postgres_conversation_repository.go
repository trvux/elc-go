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
//
// The whole check-then-act runs inside one transaction, serialized by a
// Postgres advisory lock keyed by visitor_id (pg_advisory_xact_lock) —
// there's no existing row to SELECT ... FOR UPDATE when this is a brand-new
// visitor's first message (nothing exists yet to lock), so a row-level lock
// can't close this race the way it does for the resurrect-soft-deleted-slug
// pattern elsewhere in this codebase (category/project/service-group).
// Without this, two concurrent calls for the same visitor (a double-click
// send, or two tabs open at once) would both see no recent conversation and
// both INSERT, splitting the visitor's history across two rows. The lock is
// transaction-scoped, so it releases automatically on commit or rollback —
// no separate unlock call needed. See
// docs/rfc/2026-09-02-backend-code-review-round2.md §3.2.
func (r *PostgresConversationRepository) GetOrCreateByVisitor(ctx context.Context, visitorID string, userID *string) (*domain.Conversation, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (begin tx): %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtext($1))", visitorID); err != nil {
		return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (advisory lock): %w", err)
	}

	row := tx.QueryRow(ctx, `
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
			if _, updErr := tx.Exec(ctx, `UPDATE ai_conversations SET user_id = $1 WHERE id = $2`, *userID, conv.ID); updErr != nil {
				return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (attach user): %w", updErr)
			}
			conv.UserID = userID
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (commit tx): %w", err)
		}
		return conv, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (query): %w", err)
	}

	row = tx.QueryRow(ctx, `
		INSERT INTO ai_conversations (visitor_id, user_id)
		VALUES ($1, $2)
		RETURNING id, visitor_id, user_id, created_at, updated_at`,
		visitorID, userID,
	)
	conv, err = scanConversation(row)
	if err != nil {
		return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (insert): %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("ai conversation repository getOrCreateByVisitor (commit tx): %w", err)
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
		INSERT INTO ai_messages (conversation_id, role, content, blocked_reason, incomplete, provider_id, model_id, input_tokens, output_tokens, cache_hit_tokens, cost_usd, products_shown)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at`,
		msg.ConversationID, string(msg.Role), msg.Content, msg.BlockedReason, msg.Incomplete, msg.ProviderID, msg.ModelID,
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
	// incomplete = false excludes replies a mid-stream error cut short —
	// feeding a truncated answer back to the model as if it were the full
	// reply would confuse later turns, see the Phase 2 RFC §2.4.
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, role, content, created_at
		FROM ai_messages
		WHERE conversation_id = $1 AND role IN ('user', 'assistant') AND blocked_reason IS NULL AND NOT incomplete
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

// ListConversations is the admin-facing paginated list (Phase 3) — see
// ConversationRepository's doc comment on how this differs from
// ListRecentMessages.
func (r *PostgresConversationRepository) ListConversations(ctx context.Context, filter domain.ConversationFilter) ([]*domain.ConversationSummary, int, error) {
	total, err := r.countConversations(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return nil, 0, nil
	}

	rows, err := r.pool.Query(ctx, `
		SELECT c.id, c.visitor_id, c.user_id, c.created_at, c.updated_at,
		       COUNT(m.id) AS message_count,
		       COALESCE(SUM(m.cost_usd), 0) AS total_cost_usd
		FROM ai_conversations c
		LEFT JOIN ai_messages m ON m.conversation_id = c.id
		WHERE ($1::timestamptz IS NULL OR c.created_at >= $1)
		  AND ($2::timestamptz IS NULL OR c.created_at <= $2)
		GROUP BY c.id
		ORDER BY c.updated_at DESC
		LIMIT $3 OFFSET $4`,
		filter.From, filter.To, filter.Limit, filter.Offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("ai conversation repository listConversations: %w", err)
	}
	defer rows.Close()

	var summaries []*domain.ConversationSummary
	for rows.Next() {
		var s domain.ConversationSummary
		if err := rows.Scan(&s.ID, &s.VisitorID, &s.UserID, &s.CreatedAt, &s.UpdatedAt, &s.MessageCount, &s.TotalCostUSD); err != nil {
			return nil, 0, fmt.Errorf("ai conversation repository listConversations (scan): %w", err)
		}
		summaries = append(summaries, &s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("ai conversation repository listConversations (rows): %w", err)
	}
	return summaries, total, nil
}

func (r *PostgresConversationRepository) countConversations(ctx context.Context, filter domain.ConversationFilter) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM ai_conversations
		WHERE ($1::timestamptz IS NULL OR created_at >= $1)
		  AND ($2::timestamptz IS NULL OR created_at <= $2)`,
		filter.From, filter.To,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("ai conversation repository countConversations: %w", err)
	}
	return count, nil
}

// GetMessages returns every message of one conversation — see
// ConversationRepository's doc comment on how this differs from
// ListRecentMessages.
func (r *PostgresConversationRepository) GetMessages(ctx context.Context, conversationID string) ([]*domain.ConversationMessage, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, conversation_id, role, content, blocked_reason, incomplete,
		       provider_id, model_id, input_tokens, output_tokens, cache_hit_tokens,
		       cost_usd, products_shown, created_at
		FROM ai_messages
		WHERE conversation_id = $1
		ORDER BY created_at ASC`,
		conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("ai conversation repository getMessages: %w", err)
	}
	defer rows.Close()

	var messages []*domain.ConversationMessage
	for rows.Next() {
		m, err := scanFullConversationMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("ai conversation repository getMessages (scan): %w", err)
		}
		messages = append(messages, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ai conversation repository getMessages (rows): %w", err)
	}
	return messages, nil
}

func scanFullConversationMessage(row pgx.Row) (*domain.ConversationMessage, error) {
	var (
		m                                         domain.ConversationMessage
		role                                      string
		inputTokens, outputTokens, cacheHitTokens *int
		productsJSON                              []byte
	)
	if err := row.Scan(
		&m.ID, &m.ConversationID, &role, &m.Content, &m.BlockedReason, &m.Incomplete,
		&m.ProviderID, &m.ModelID, &inputTokens, &outputTokens, &cacheHitTokens,
		&m.CostUSD, &productsJSON, &m.CreatedAt,
	); err != nil {
		return nil, err
	}
	m.Role = domain.Role(role)
	if inputTokens != nil || outputTokens != nil || cacheHitTokens != nil {
		usage := domain.TokenUsage{}
		if inputTokens != nil {
			usage.InputTokens = *inputTokens
		}
		if outputTokens != nil {
			usage.OutputTokens = *outputTokens
		}
		if cacheHitTokens != nil {
			usage.CacheHitTokens = *cacheHitTokens
		}
		m.Usage = &usage
	}
	if len(productsJSON) > 0 {
		_ = json.Unmarshal(productsJSON, &m.ProductsShown)
	}
	return &m, nil
}

// usageReportQueries maps each UsageGroupBy to its pre-written SQL — never
// built dynamically from the groupBy param, so an invalid value can't reach
// SQL (application.GetUsageReport rejects it before this is ever called).
var usageReportQueries = map[domain.UsageGroupBy]string{
	domain.UsageGroupByDay: `
		SELECT to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS key,
		       COUNT(*), COUNT(*) FILTER (WHERE blocked_reason IS NOT NULL),
		       COALESCE(SUM(input_tokens), 0), COALESCE(SUM(output_tokens), 0), COALESCE(SUM(cost_usd), 0)
		FROM ai_messages
		WHERE role = 'assistant' AND created_at BETWEEN $1 AND $2
		GROUP BY key ORDER BY key`,
	domain.UsageGroupByProvider: `
		SELECT COALESCE(p.name, 'unknown') AS key,
		       COUNT(*), COUNT(*) FILTER (WHERE m.blocked_reason IS NOT NULL),
		       COALESCE(SUM(m.input_tokens), 0), COALESCE(SUM(m.output_tokens), 0), COALESCE(SUM(m.cost_usd), 0)
		FROM ai_messages m
		LEFT JOIN ai_providers p ON p.id = m.provider_id
		WHERE m.role = 'assistant' AND m.created_at BETWEEN $1 AND $2
		GROUP BY key ORDER BY key`,
	domain.UsageGroupByModel: `
		SELECT COALESCE(p.name || '/' || mo.model_name, 'unknown') AS key,
		       COUNT(*), COUNT(*) FILTER (WHERE m.blocked_reason IS NOT NULL),
		       COALESCE(SUM(m.input_tokens), 0), COALESCE(SUM(m.output_tokens), 0), COALESCE(SUM(m.cost_usd), 0)
		FROM ai_messages m
		LEFT JOIN ai_models mo ON mo.id = m.model_id
		LEFT JOIN ai_providers p ON p.id = mo.provider_id
		WHERE m.role = 'assistant' AND m.created_at BETWEEN $1 AND $2
		GROUP BY key ORDER BY key`,
}

func (r *PostgresConversationRepository) GetUsageReport(ctx context.Context, from, to time.Time, groupBy domain.UsageGroupBy) ([]domain.UsageReportRow, error) {
	query, ok := usageReportQueries[groupBy]
	if !ok {
		return nil, fmt.Errorf("ai conversation repository getUsageReport: unknown groupBy %q", groupBy)
	}

	rows, err := r.pool.Query(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("ai conversation repository getUsageReport: %w", err)
	}
	defer rows.Close()

	var report []domain.UsageReportRow
	for rows.Next() {
		var row domain.UsageReportRow
		if err := rows.Scan(&row.Key, &row.MessageCount, &row.BlockedCount, &row.InputTokens, &row.OutputTokens, &row.CostUSD); err != nil {
			return nil, fmt.Errorf("ai conversation repository getUsageReport (scan): %w", err)
		}
		report = append(report, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ai conversation repository getUsageReport (rows): %w", err)
	}
	return report, nil
}
