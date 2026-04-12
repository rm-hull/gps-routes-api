//go:build postgres

package cmds

import (
	"context"
	"database/sql"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rm-hull/gps-routes-api/db"
)

func initializeDB(ctx context.Context, config db.DBConfig) (*pgxpool.Pool, *sql.DB, error) {
	pool, err := db.NewDBPool(ctx, config)
	if err != nil {
		return nil, nil, err
	}

	sqlDB := stdlib.OpenDB(*pool.Config().ConnConfig)
	return pool, sqlDB, nil
}
