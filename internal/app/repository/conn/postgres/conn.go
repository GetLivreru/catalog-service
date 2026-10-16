package rcpostgres

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
	"github.com/uptrace/bun/migrate"

	"github.com/GetLivreru/catalog-service/internal/app/config/section"
	"github.com/GetLivreru/catalog-service/migration"
)

type (
	Client struct {
		_bunDB
		rawBunDB *bun.DB
		cfg      section.RepositoryPostgres
	}

	_bunDB = bun.IDB
)

func (c *Client) GetRawBunDB() *bun.DB {
	return c.rawBunDB
}

func NewClient(ctx context.Context, cfg section.RepositoryPostgres) (*Client, error) {
	u := url.URL{
		Scheme: "postgres",
		Host:   cfg.Address,
		User:   url.UserPassword(cfg.Username, cfg.Password),
		Path:   cfg.Name,
	}
	args := make(url.Values)
	args.Set("sslmode", "disable")
	u.RawQuery = args.Encode()
	log.Printf("PostgreSQL timeouts read=%s write=%s", cfg.ReadTimeout, cfg.WriteTimeout)

	sqlDB := sql.OpenDB(pgdriver.NewConnector(
		pgdriver.WithDSN(u.String()),
		pgdriver.WithReadTimeout(cfg.ReadTimeout),
		pgdriver.WithWriteTimeout(cfg.WriteTimeout),
	))
	sqlDB.SetMaxOpenConns(10)
	db := bun.NewDB(sqlDB, pgdialect.New(), bun.WithDiscardUnknownColumns())

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return &Client{_bunDB: db, rawBunDB: db, cfg: cfg}, nil
}

func (c *Client) Migrate(ctx context.Context) (oldVer, newVer int64, err error) {
	migrations := migrate.NewMigrations()
	if err := migrations.Discover(migration.Postgres); err != nil {
		return 0, 0, fmt.Errorf("failed to discover migrations: %w", err)
	}
	m := migrate.NewMigrator(c.rawBunDB, migrations,
		migrate.WithTableName(c.cfg.MigrationTable),
		migrate.WithLocksTableName(c.cfg.MigrationTable+"_lock"),
		migrate.WithMarkAppliedOnSuccess(true),
	)
	if err := m.Init(ctx); err != nil {
		return 0, 0, fmt.Errorf("failed to initialize migrations: %w", err)
	}
	applied, err := m.AppliedMigrations(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	oldVer, err = migrationVersion(applied)
	if err != nil {
		return 0, 0, err
	}
	group, err := m.Migrate(ctx)
	if err != nil {
		return oldVer, oldVer, fmt.Errorf("failed to apply migrations: %w", err)
	}
	newVer, err = migrationVersion(group.Migrations)
	if err != nil {
		return oldVer, oldVer, err
	}
	return oldVer, max(oldVer, newVer), nil
}

// migrationVersion returns the highest numeric version regardless of query order.
func migrationVersion(migrations migrate.MigrationSlice) (int64, error) {
	var version int64
	for _, m := range migrations {
		v, err := strconv.ParseInt(m.Name, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse migration version %q: %w", m.Name, err)
		}
		version = max(version, v)
	}
	return version, nil
}
