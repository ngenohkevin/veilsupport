package testhelpers

import (
	"context"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// SetupTestDB creates a test database using testcontainers
func SetupTestDB(t *testing.T) *pgxpool.Pool {
	ctx := context.Background()

	// Start PostgreSQL container
	postgresContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	require.NoError(t, err)

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Connect to database
	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Test connection
	err = pool.Ping(ctx)
	require.NoError(t, err)

	// Run database migrations
	runMigrations(t, connStr)

	// Cleanup function
	t.Cleanup(func() {
		pool.Close()
		postgresContainer.Terminate(ctx)
	})

	return pool
}

// runMigrations runs database migrations for tests
func runMigrations(t *testing.T, connStr string) {
	m, err := migrate.New(
		"file://../../db/migrations",
		connStr,
	)
	require.NoError(t, err)

	err = m.Up()
	require.NoError(t, err)

	// Close migrate instance
	sourceErr, dbErr := m.Close()
	require.NoError(t, sourceErr)
	require.NoError(t, dbErr)
}

// CleanupTestDB performs cleanup operations on test database
func CleanupTestDB(t *testing.T, pool *pgxpool.Pool) {
	// This will be called automatically via t.Cleanup in SetupTestDB
	// Additional cleanup logic can be added here if needed
}