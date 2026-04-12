//go:build !postgres

package cmds

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rm-hull/gps-routes-api/db"
)

func initializeDB(ctx context.Context, config db.DBConfig) (*pgxpool.Pool, *sql.DB, error) {
	sqlDB, err := db.NewSQLiteDB(config)
	if err != nil {
		return nil, nil, err
	}
	return nil, sqlDB, nil
}
