package store_test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // migrate's pgx driver
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

// testPool is shared by all tests in this package. TestMain starts ONE
// throwaway Postgres container, migrates it, runs every test, then kills it.
var testPool *pgxpool.Pool

func TestMain(m *testing.M) {
	// TestMain runs before the testing framework parses flags — parse them
	// ourselves so testing.Short() is usable here.
	flag.Parse()

	// `go test -short ./...` skips the Docker-dependent setup entirely.
	if testing.Short() {
		os.Exit(m.Run())
	}

	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("mb_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	if err != nil {
		log.Fatalf("starting postgres container: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("connection string: %v", err)
	}

	if err := runMigrations(dsn); err != nil {
		log.Fatalf("migrating test db: %v", err)
	}

	testPool, err = pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("connecting pool: %v", err)
	}

	code := m.Run()

	testPool.Close()
	timeout := 10 * time.Second
	_ = container.Stop(ctx, &timeout)
	os.Exit(code)
}

// runMigrations applies the real migration files — the test schema is BY
// CONSTRUCTION identical to dev and prod.
func runMigrations(dsn string) error {
	src, err := iofs.New(os.DirFS("../../migrations"), ".")
	if err != nil {
		return fmt.Errorf("reading migrations: %w", err)
	}
	// migrate's pgx driver registers itself under the scheme "pgx5"
	mig, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+trimScheme(dsn))
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer func() { _, _ = mig.Close() }() // best-effort cleanup in tests
	if err := mig.Up(); err != nil {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}

func trimScheme(dsn string) string {
	for _, prefix := range []string{"postgres://", "postgresql://"} {
		if len(dsn) > len(prefix) && dsn[:len(prefix)] == prefix {
			return dsn[len(prefix):]
		}
	}
	return dsn
}

// resetDB wipes all data between tests (schema stays). RESTART IDENTITY
// makes generated ids predictable again.
func resetDB(t *testing.T) {
	t.Helper()
	// promo_codes is listed explicitly: every other E7 table hangs off
	// users, orders or promo_codes and is caught by CASCADE, but the codes
	// table itself references nothing here and would leak between tests.
	_, err := testPool.Exec(context.Background(), `
		TRUNCATE cart_items, order_items, orders, sessions, users,
		         product_variants, products, categories, promo_codes
		RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("resetting db: %v", err)
	}
}
