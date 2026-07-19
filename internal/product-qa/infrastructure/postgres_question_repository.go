package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/product-qa/domain"
)

type PostgresQuestionRepository struct {
	pool *pgxpool.Pool
}

var _ domain.QuestionRepository = (*PostgresQuestionRepository)(nil)

func NewPostgresQuestionRepository(pool *pgxpool.Pool) *PostgresQuestionRepository {
	return &PostgresQuestionRepository{pool: pool}
}

const questionColumns = `id, product_id, asker_name, asker_email, question_text, answer_text,
	status, is_published, answered_at, source_ip, user_agent, created_at, updated_at, deleted_at`

func (r *PostgresQuestionRepository) Create(ctx context.Context, q *domain.Question) (*domain.Question, error) {
	query := `
		INSERT INTO product_questions (product_id, asker_name, asker_email, question_text, source_ip, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING ` + questionColumns

	row := r.pool.QueryRow(ctx, query,
		q.ProductID(), q.AskerName(), q.AskerEmail(), q.QuestionText(), q.SourceIP(), q.UserAgent(),
	)
	created, err := scanQuestion(row)
	if err != nil {
		return nil, fmt.Errorf("question repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresQuestionRepository) GetByID(ctx context.Context, id string) (*domain.Question, error) {
	query := "SELECT " + questionColumns + " FROM product_questions WHERE id = $1 AND deleted_at IS NULL"

	row := r.pool.QueryRow(ctx, query, id)
	q, err := scanQuestion(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("question repository getById: %w", err)
	}
	return q, nil
}

// questionFilterConditions builds WHERE clauses shared by GetAll/Count so the
// two queries can never drift out of sync with each other — same pattern as
// inquiry's inquiryFilterConditions.
func questionFilterConditions(filter domain.QuestionFilter) ([]string, []any) {
	conditions := []string{"deleted_at IS NULL"}
	args := []any{}
	argN := 1
	next := func() int {
		n := argN
		argN++
		return n
	}

	if filter.ProductID != nil {
		conditions = append(conditions, fmt.Sprintf("product_id = $%d", next()))
		args = append(args, *filter.ProductID)
	}
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", next()))
		args = append(args, string(*filter.Status))
	}
	if filter.PublishedOnly {
		conditions = append(conditions, "is_published = true")
	}

	return conditions, args
}

func (r *PostgresQuestionRepository) GetAll(ctx context.Context, filter domain.QuestionFilter) ([]*domain.Question, error) {
	query := "SELECT " + questionColumns + " FROM product_questions"
	conditions, args := questionFilterConditions(filter)
	query += " WHERE " + strings.Join(conditions, " AND ")
	query += " ORDER BY created_at DESC"

	argN := len(args) + 1
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argN)
		args = append(args, filter.Limit)
		argN++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argN)
		args = append(args, filter.Offset)
	}

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("question repository getAll: %w", err)
	}
	defer rows.Close()

	var questions []*domain.Question
	for rows.Next() {
		q, err := scanQuestion(rows)
		if err != nil {
			return nil, fmt.Errorf("question repository getAll scan: %w", err)
		}
		questions = append(questions, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("question repository getAll rows: %w", err)
	}
	return questions, nil
}

func (r *PostgresQuestionRepository) Count(ctx context.Context, filter domain.QuestionFilter) (int, error) {
	query := "SELECT COUNT(*) FROM product_questions"
	conditions, args := questionFilterConditions(filter)
	query += " WHERE " + strings.Join(conditions, " AND ")

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("question repository count: %w", err)
	}
	return count, nil
}

func (r *PostgresQuestionRepository) Update(ctx context.Context, q *domain.Question) (*domain.Question, error) {
	query := `
		UPDATE product_questions
		SET answer_text = $1, status = $2, is_published = $3, answered_at = $4, updated_at = now()
		WHERE id = $5
		RETURNING ` + questionColumns

	row := r.pool.QueryRow(ctx, query, q.AnswerText(), string(q.Status()), q.IsPublished(), q.AnsweredAt(), q.ID())
	updated, err := scanQuestion(row)
	if err != nil {
		return nil, fmt.Errorf("question repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresQuestionRepository) SoftDelete(ctx context.Context, id string) error {
	if _, err := r.pool.Exec(ctx, "UPDATE product_questions SET deleted_at = $1 WHERE id = $2", time.Now(), id); err != nil {
		return fmt.Errorf("question repository softDelete: %w", err)
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanQuestion(row rowScanner) (*domain.Question, error) {
	var (
		id, productID, askerName, questionText, status string
		askerEmail, answerText                         *string
		isPublished                                    bool
		answeredAt                                     *time.Time
		sourceIP, userAgent                            *string
		createdAt, updatedAt                           time.Time
		deletedAt                                      *time.Time
	)

	if err := row.Scan(
		&id, &productID, &askerName, &askerEmail, &questionText, &answerText,
		&status, &isPublished, &answeredAt, &sourceIP, &userAgent, &createdAt, &updatedAt, &deletedAt,
	); err != nil {
		return nil, err
	}

	return domain.RehydrateQuestion(
		id, productID, askerName, askerEmail, questionText, answerText,
		domain.QuestionStatus(status), isPublished, answeredAt, sourceIP, userAgent,
		createdAt, updatedAt, deletedAt,
	), nil
}
