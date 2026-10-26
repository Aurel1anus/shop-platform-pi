package mysqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	mysql "github.com/go-sql-driver/mysql"

	"github.com/LYP-leo/xzxg-shop/commerce/migrations"
)

type MySQLStore struct{ db *sql.DB }

func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

func OpenMySQL(ctx context.Context, dsn string) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}
	return NewMySQLStore(db), nil
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func isDuplicateKeyError(err error) bool {
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func (s *MySQLStore) Migrate(ctx context.Context) error { return s.execScript(ctx, migrations.Schema) }
func (s *MySQLStore) Seed(ctx context.Context) error    { return s.execScript(ctx, migrations.Seed) }
func (s *MySQLStore) execScript(ctx context.Context, script string) error {
	for _, stmt := range splitSQLStatements(script) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("exec schema/seed: %w", err)
		}
	}
	return nil
}
