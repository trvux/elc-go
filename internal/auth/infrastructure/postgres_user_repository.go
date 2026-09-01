package infrastructure

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/trvux/elc-go/internal/auth/domain"
	"github.com/trvux/elc-go/internal/platform/apperr"
)

type PostgresUserRepository struct {
	pool *pgxpool.Pool
}

var _ domain.UserRepository = (*PostgresUserRepository)(nil)

func NewPostgresUserRepository(pool *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{pool: pool}
}

const userColumns = "id, username, email, password_hash, name, phone, avatar_url, role, status, last_login_at, created_at, updated_at"

func (r *PostgresUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	query := `
		INSERT INTO users (username, email, password_hash, name, phone, avatar_url, role, status)
		VALUES ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, $8)
		RETURNING ` + userColumns

	row := r.pool.QueryRow(ctx, query,
		user.Username(), user.Email(), user.PasswordHash(), user.Name(), user.Phone(), user.AvatarURL(),
		string(user.Role()), string(user.Status()),
	)
	created, err := scanUser(row)
	if err != nil {
		// A rare concurrent AcceptInvite for the same username/email (both
		// requests pass the application layer's ExistsByUsernameOrEmail
		// check before either commits) hits this table's unique constraint —
		// map it to a clean 409 instead of leaking a raw 500, same "map a
		// known DB condition to an apperr here" convention as page's Update
		// TOCTOU fix.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, apperr.NewConflictError("username or email already in use")
		}
		return nil, fmt.Errorf("user repository create: %w", err)
	}
	return created, nil
}

func (r *PostgresUserRepository) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	query := `
		UPDATE users
		SET username = $1, email = $2, password_hash = $3, name = NULLIF($4, ''), phone = NULLIF($5, ''),
		    avatar_url = NULLIF($6, ''), role = $7, status = $8, last_login_at = $9, updated_at = now()
		WHERE id = $10
		RETURNING ` + userColumns

	row := r.pool.QueryRow(ctx, query,
		user.Username(), user.Email(), user.PasswordHash(), user.Name(), user.Phone(), user.AvatarURL(),
		string(user.Role()), string(user.Status()), user.LastLoginAt(), user.ID(),
	)
	updated, err := scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("user repository update: %w", err)
	}
	return updated, nil
}

func (r *PostgresUserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE id = $1", id)
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("user repository getById: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row := r.pool.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE email = lower($1)", email)
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("user repository getByEmail: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) GetByIdentifier(ctx context.Context, identifier string) (*domain.User, error) {
	query := "SELECT " + userColumns + " FROM users WHERE username = lower($1) OR email = lower($1)"
	row := r.pool.QueryRow(ctx, query, identifier)
	user, err := scanUser(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("user repository getByIdentifier: %w", err)
	}
	return user, nil
}

func (r *PostgresUserRepository) ExistsByUsernameOrEmail(ctx context.Context, username, email string) (bool, error) {
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM users WHERE username = lower($1) OR email = lower($2))"
	if err := r.pool.QueryRow(ctx, query, username, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("user repository existsByUsernameOrEmail: %w", err)
	}
	return exists, nil
}

func (r *PostgresUserRepository) GetAll(ctx context.Context) ([]*domain.User, error) {
	rows, err := r.pool.Query(ctx, "SELECT "+userColumns+" FROM users ORDER BY created_at ASC")
	if err != nil {
		return nil, fmt.Errorf("user repository getAll: %w", err)
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("user repository getAll scan: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("user repository getAll rows: %w", err)
	}

	return users, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner) (*domain.User, error) {
	var (
		id, username, email, passwordHash, role, status string
		name, phone, avatarURL                          *string
		lastLoginAt                                     *time.Time
		createdAt, updatedAt                            time.Time
	)

	if err := row.Scan(&id, &username, &email, &passwordHash, &name, &phone, &avatarURL, &role, &status, &lastLoginAt, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	return domain.RehydrateUser(id, username, email, passwordHash, deref(name), deref(phone), deref(avatarURL), domain.Role(role), domain.UserStatus(status), lastLoginAt, createdAt, updatedAt), nil
}
