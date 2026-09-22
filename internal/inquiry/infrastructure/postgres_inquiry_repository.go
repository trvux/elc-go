package infrastructure

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/inquiry/domain"
)

// PostgresInquiryRepository implements domain.InquiryRepository against the
// inquiries table.
type PostgresInquiryRepository struct {
	pool *pgxpool.Pool
}

var _ domain.InquiryRepository = (*PostgresInquiryRepository)(nil)

func NewPostgresInquiryRepository(pool *pgxpool.Pool) *PostgresInquiryRepository {
	return &PostgresInquiryRepository{pool: pool}
}

const inquiryColumns = `id, name, phone, email, message, product_id, project_id, service_id,
	lead_type, sub_type, qualify_data, attachments,
	status, internal_note, source_ip, user_agent, created_at, updated_at`

func (r *PostgresInquiryRepository) Create(ctx context.Context, inquiry *domain.Inquiry) (*domain.Inquiry, error) {
	query := `
		INSERT INTO inquiries (
			name, phone, email, message, product_id, project_id, service_id,
			lead_type, sub_type, qualify_data, attachments,
			status, source_ip, user_agent
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING ` + inquiryColumns

	// attachments is a domain []string — marshaled to JSON here so pgx sends
	// it as a jsonb value rather than encoding a Go slice as a postgres
	// array; qualifyData is already json.RawMessage (raw bytes), same as
	// how internal/page's Content field is inserted as-is. A nil slice
	// marshals to JSON `null`, not `[]` — normalize so the stored value
	// matches the column's `[]` default instead of a JSON null.
	attachments := inquiry.Attachments()
	if attachments == nil {
		attachments = []string{}
	}
	attachmentsBytes, err := json.Marshal(attachments)
	if err != nil {
		return nil, fmt.Errorf("inquiry repository create: marshal attachments: %w", err)
	}
	// Cast to json.RawMessage, not plain []byte — pgx's default type map
	// sends plain []byte through the bytea codec (wrong wire format for a
	// jsonb column: "invalid input syntax for type json"), but recognizes
	// json.RawMessage specifically and sends it as JSON text, same as
	// qualifyData below.
	attachmentsJSON := json.RawMessage(attachmentsBytes)

	row := r.pool.QueryRow(ctx, query,
		inquiry.Name(), inquiry.Phone(), inquiry.Email(), inquiry.Message(),
		inquiry.ProductID(), inquiry.ProjectID(), inquiry.ServiceID(),
		string(inquiry.LeadType()), inquiry.SubType(), inquiry.QualifyData(), attachmentsJSON,
		string(inquiry.Status()), inquiry.SourceIP(), inquiry.UserAgent(),
	)
	created, err := scanInquiry(row)
	if err != nil {
		return nil, fmt.Errorf("inquiry repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresInquiryRepository) GetAll(ctx context.Context, filter domain.InquiryFilter) ([]*domain.Inquiry, error) {
	query := "SELECT " + inquiryColumns + " FROM inquiries"
	conditions, args := inquiryFilterConditions(filter)
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}
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
		return nil, fmt.Errorf("inquiry repository getAll: %w", err)
	}
	defer rows.Close()

	var inquiries []*domain.Inquiry
	for rows.Next() {
		inquiry, err := scanInquiry(rows)
		if err != nil {
			return nil, fmt.Errorf("inquiry repository getAll scan: %w", err)
		}
		inquiries = append(inquiries, inquiry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inquiry repository getAll rows: %w", err)
	}

	return inquiries, nil
}

func (r *PostgresInquiryRepository) Count(ctx context.Context, filter domain.InquiryFilter) (int, error) {
	query := "SELECT COUNT(*) FROM inquiries"
	conditions, args := inquiryFilterConditions(filter)
	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	if err := r.pool.QueryRow(ctx, query, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("inquiry repository count: %w", err)
	}
	return count, nil
}

func (r *PostgresInquiryRepository) GetByID(ctx context.Context, id string) (*domain.Inquiry, error) {
	query := "SELECT " + inquiryColumns + " FROM inquiries WHERE id = $1"

	row := r.pool.QueryRow(ctx, query, id)
	inquiry, err := scanInquiry(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("inquiry repository getById: %w", err)
	}
	return inquiry, nil
}

func (r *PostgresInquiryRepository) Update(ctx context.Context, inquiry *domain.Inquiry) (*domain.Inquiry, error) {
	query := `
		UPDATE inquiries
		SET status = $1, internal_note = $2, updated_at = now()
		WHERE id = $3
		RETURNING ` + inquiryColumns

	row := r.pool.QueryRow(ctx, query, string(inquiry.Status()), inquiry.InternalNote(), inquiry.ID())
	updated, err := scanInquiry(row)
	if err != nil {
		return nil, fmt.Errorf("inquiry repository update: %w", err)
	}
	return updated, nil
}

// inquiryFilterConditions builds WHERE clauses shared by GetAll/Count so the
// two queries can never drift out of sync with each other.
func inquiryFilterConditions(filter domain.InquiryFilter) ([]string, []any) {
	conditions := []string{}
	args := []any{}
	argN := 1

	if filter.Status != "" && filter.Status != "all" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argN))
		args = append(args, filter.Status)
		argN++
	}
	if filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR phone ILIKE $%d OR email ILIKE $%d)", argN, argN, argN))
		args = append(args, "%"+filter.Search+"%")
		argN++
	}

	return conditions, args
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanInquiry(row rowScanner) (*domain.Inquiry, error) {
	var (
		id, name, phone, status, leadType string
		email, message                    *string
		productID                         *string
		projectID                         *string
		serviceID                         *string
		subType                           *string
		qualifyData                       json.RawMessage
		attachmentsRaw                    json.RawMessage
		internalNote                      *string
		sourceIP                          *string
		userAgent                         *string
		createdAt, updatedAt              time.Time
	)

	if err := row.Scan(
		&id, &name, &phone, &email, &message,
		&productID, &projectID, &serviceID,
		&leadType, &subType, &qualifyData, &attachmentsRaw,
		&status, &internalNote, &sourceIP, &userAgent,
		&createdAt, &updatedAt,
	); err != nil {
		return nil, err
	}

	var attachments []string
	if len(attachmentsRaw) > 0 {
		if err := json.Unmarshal(attachmentsRaw, &attachments); err != nil {
			return nil, fmt.Errorf("inquiry repository scan: unmarshal attachments: %w", err)
		}
	}

	return domain.RehydrateInquiry(
		id, name, phone, email, message,
		productID, projectID, serviceID,
		domain.LeadType(leadType), subType, qualifyData, attachments,
		domain.InquiryStatus(status), internalNote, sourceIP, userAgent,
		createdAt, updatedAt,
	), nil
}
