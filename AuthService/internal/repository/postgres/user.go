package postgres

import (
	"context"
	"time"

	repo "github.com/kingxl111/mosprom/AuthService/internal/repository"

	sq "github.com/Masterminds/squirrel"
)

const (
	usersTable         = "users"
	refreshTokensTable = "refresh_tokens"
	userSessionsTable  = "user_sessions"

	idColumn           = "id"
	emailColumn        = "email"
	passwordHashColumn = "password_hash"
	roleColumn         = "role"
	isActiveColumn     = "is_active"
	createdAtColumn    = "created_at"
	updatedAtColumn    = "updated_at"
	lastLoginColumn    = "last_login"

	userIDColumn    = "user_id"
	tokenColumn     = "token"
	expiresAtColumn = "expires_at"
	revokedColumn   = "revoked"

	ipAddressColumn    = "ip_address"
	userAgentColumn    = "user_agent"
	lastActivityColumn = "last_activity"
)

// repository структура, оборачивающая DB
type repository struct {
	db *DB
}

func NewRepository(db *DB) *repository {
	return &repository{db: db}
}

func (r *repository) CreateUser(ctx context.Context, user *User) error {
	builder := sq.Insert(usersTable).
		Columns(emailColumn, passwordHashColumn, roleColumn, isActiveColumn, createdAtColumn, updatedAtColumn).
		Values(user.Email, user.PasswordHash, user.Role, user.IsActive, time.Now(), time.Now()).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return repo.ErrorBuildInsertQuery
	}

	err = r.db.pool.QueryRow(ctx, query, args...).Scan(&user.ID)
	if err != nil {
		return repo.ErrorInsertUser
	}
	return nil
}

func (r *repository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	builder := sq.Select(
		idColumn,
		emailColumn,
		passwordHashColumn,
		roleColumn,
		isActiveColumn,
		createdAtColumn,
		updatedAtColumn,
		lastLoginColumn,
	).
		From(usersTable).
		Where(sq.Eq{emailColumn: email}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, repo.ErrorBuildSelectQuery
	}

	row := r.db.pool.QueryRow(ctx, query, args...)
	var u User
	err = row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.LastLogin)
	if err != nil {
		return nil, repo.ErrorUserNotFound
	}

	return &u, nil
}

func (r *repository) UpdateLastLogin(ctx context.Context, userID int) error {
	builder := sq.Update(usersTable).
		Set(lastLoginColumn, time.Now()).
		Set(updatedAtColumn, time.Now()).
		Where(sq.Eq{idColumn: userID}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return repo.ErrorBuildUpdateQuery
	}

	_, err = r.db.pool.Exec(ctx, query, args...)
	if err != nil {
		return repo.ErrorUpdateUser
	}
	return nil
}

func (r *repository) SaveRefreshToken(ctx context.Context, token *RefreshToken) error {
	builder := sq.Insert(refreshTokensTable).
		Columns(userIDColumn, tokenColumn, expiresAtColumn, revokedColumn, createdAtColumn).
		Values(token.UserID, token.Token, token.ExpiresAt, false, time.Now()).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return repo.ErrorBuildInsertQuery
	}

	err = r.db.pool.QueryRow(ctx, query, args...).Scan(&token.ID)
	if err != nil {
		return repo.ErrorInsertToken
	}
	return nil
}

func (r *repository) RevokeRefreshToken(ctx context.Context, token string) error {
	builder := sq.Update(refreshTokensTable).
		Set(revokedColumn, true).
		Where(sq.Eq{tokenColumn: token}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return repo.ErrorBuildUpdateQuery
	}

	_, err = r.db.pool.Exec(ctx, query, args...)
	if err != nil {
		return repo.ErrorUpdateToken
	}
	return nil
}

func (r *repository) CreateSession(ctx context.Context, s *UserSession) error {
	builder := sq.Insert(userSessionsTable).
		Columns(userIDColumn, ipAddressColumn, userAgentColumn, createdAtColumn, lastActivityColumn).
		Values(s.UserID, s.IPAddress, s.UserAgent, time.Now(), time.Now()).
		Suffix("RETURNING id").
		PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return repo.ErrorBuildInsertQuery
	}

	err = r.db.pool.QueryRow(ctx, query, args...).Scan(&s.ID)
	if err != nil {
		return repo.ErrorInsertSession
	}
	return nil
}

func (r *repository) UpdateSessionActivity(ctx context.Context, sessionID int) error {
	builder := sq.Update(userSessionsTable).
		Set(lastActivityColumn, time.Now()).
		Where(sq.Eq{idColumn: sessionID}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return repo.ErrorBuildUpdateQuery
	}

	_, err = r.db.pool.Exec(ctx, query, args...)
	if err != nil {
		return repo.ErrorUpdateSession
	}
	return nil
}

func (r *repository) GetRefreshToken(ctx context.Context, token string) (*RefreshToken, error) {
	builder := sq.Select(
		idColumn,
		userIDColumn,
		tokenColumn,
		expiresAtColumn,
		revokedColumn,
		createdAtColumn,
	).
		From(refreshTokensTable).
		Where(sq.Eq{tokenColumn: token}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, repo.ErrorBuildSelectQuery
	}

	row := r.db.pool.QueryRow(ctx, query, args...)
	var rt RefreshToken
	err = row.Scan(&rt.ID, &rt.UserID, &rt.Token, &rt.ExpiresAt, &rt.Revoked, &rt.CreatedAt)
	if err != nil {
		return nil, repo.ErrorNotFound
	}
	return &rt, nil
}

func (r *repository) GetUserByID(ctx context.Context, id int) (*User, error) {
	builder := sq.Select(
		idColumn,
		emailColumn,
		passwordHashColumn,
		roleColumn,
		isActiveColumn,
		createdAtColumn,
		updatedAtColumn,
		lastLoginColumn,
	).
		From(usersTable).
		Where(sq.Eq{idColumn: id}).
		PlaceholderFormat(sq.Dollar)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, repo.ErrorBuildSelectQuery
	}

	row := r.db.pool.QueryRow(ctx, query, args...)
	var u User
	err = row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt, &u.LastLogin)
	if err != nil {
		return nil, repo.ErrorNotFound
	}
	return &u, nil
}
