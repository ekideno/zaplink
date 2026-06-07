package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/ekideno/zaplink/internal/model"
	"github.com/jackc/pgx/v5"
)

func (r *LinkPostgresRepository) CreateLink(ctx context.Context, link *model.Link) error {
	const query = `
		INSERT INTO links (short_code, original_url, is_active)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	return r.db.Pool.QueryRow(ctx, query, link.ShortCode, link.OriginalURL, link.IsActive).
		Scan(&link.ID, &link.CreatedAt)
}

func (r *LinkPostgresRepository) GetLinkByID(ctx context.Context, id int64) (*model.Link, error) {
	const query = `
		SELECT id, short_code, original_url, is_active, created_at
		FROM links
		WHERE id = $1
	`

	link := new(model.Link)
	err := r.db.Pool.QueryRow(ctx, query, id).
		Scan(&link.ID, &link.ShortCode, &link.OriginalURL, &link.IsActive, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get link by id: %w", err)
	}

	return link, nil
}

func (r *LinkPostgresRepository) GetLinkByShortCode(ctx context.Context, shortCode string) (*model.Link, error) {
	const query = `
		SELECT id, short_code, original_url, is_active, created_at
		FROM links
		WHERE short_code = $1
	`

	link := new(model.Link)
	err := r.db.Pool.QueryRow(ctx, query, shortCode).
		Scan(&link.ID, &link.ShortCode, &link.OriginalURL, &link.IsActive, &link.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("get link by short code: %w", err)
	}

	return link, nil
}

func (r *LinkPostgresRepository) UpdateLink(ctx context.Context, link *model.Link) error {
	const query = `
		UPDATE links
		SET short_code = $1,
		    original_url = $2,
		    is_active = $3
		WHERE id = $4
	`

	ct, err := r.db.Pool.Exec(ctx, query, link.ShortCode, link.OriginalURL, link.IsActive, link.ID)
	if err != nil {
		return fmt.Errorf("update link: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return nil
	}

	return nil
}
