package postgrescontainer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Query any

type sqlizer interface {
	ToSql() (sql string, args []any, err error)
}

var errInvalidQueryType = errors.New("invalid query type, expected string or sqlizer types")

func execQuery(ctx context.Context, db *sql.DB, query Query) error {
	switch query := query.(type) {
	case sqlizer:
		return execSqlizer(ctx, db, query)
	case string:
		return execString(ctx, db, query)
	default:
		return errInvalidQueryType
	}
}

func execSqlizer(ctx context.Context, db *sql.DB, query sqlizer) error {
	sql, args, err := query.ToSql()
	if err != nil {
		return fmt.Errorf("exec sqlizer query, failed convert ToSql, %w", err)
	}

	_, err = db.ExecContext(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("exec %s query, %w", sql, err)
	}

	return nil
}

func execString(ctx context.Context, db *sql.DB, query string) error {
	_, err := db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("exec %s query, %w", query, err)
	}

	return nil
}
