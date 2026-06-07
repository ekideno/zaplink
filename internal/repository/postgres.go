package repository

import "github.com/ekideno/zaplink/internal/db"

type LinkPostgresRepository struct {
	db *db.DB
}

type ClickPostgresRepository struct {
	db *db.DB
}

func NewLinkPostgres(database *db.DB) *LinkPostgresRepository {
	return &LinkPostgresRepository{db: database}
}

func NewClickPostgres(database *db.DB) *ClickPostgresRepository {
	return &ClickPostgresRepository{db: database}
}
