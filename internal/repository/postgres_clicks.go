package repository

import (
	"context"
	"database/sql"

	"github.com/ekideno/zaplink/internal/db"
	"github.com/ekideno/zaplink/internal/model"
)

type ClickPostgresRepository struct {
	db *db.DB
}

func NewClickPostgres(database *db.DB) *ClickPostgresRepository {
	return &ClickPostgresRepository{db: database}
}

func (r *ClickPostgresRepository) CreateClick(ctx context.Context, click *model.Click) error {
	const query = `
		INSERT INTO clicks (link_id, user_agent, ip_address)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	var userAgent sql.NullString
	var ipAddress sql.NullString

	if click.UserAgent != nil {
		userAgent = sql.NullString{String: *click.UserAgent, Valid: true}
	}
	if click.IPAddress != nil {
		ipAddress = sql.NullString{String: *click.IPAddress, Valid: true}
	}

	if err := r.db.Pool.QueryRow(ctx, query, click.LinkID, userAgent, ipAddress).
		Scan(&click.ID, &click.CreatedAt); err != nil {
		return err
	}

	return nil
}
