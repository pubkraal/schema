package schema

import (
	"database/sql"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPostgresLockSQL(t *testing.T) {
	name := `"schema_migrations"`

	sql := Postgres.LockSQL(name)
	if !strings.Contains(strings.ToUpper(sql), "IN EXCLUSIVE MODE") {
		t.Errorf("Expected LOCK TABLE statement to be in Exclusive Mode:\n%s", sql)
	}
	if !strings.HasPrefix(strings.ToUpper(sql), "LOCK TABLE") {
		t.Errorf("Expected LOCK TABLE statement:\n%s", sql)
	}
	if !strings.Contains(sql, name) {
		t.Errorf("Expected table name in quoted form:\n%s", sql)
	}
}

func TestPostgresSimultaneousCreateTable(t *testing.T) {
	tableNames := []string{
		"firstMigrations",
		"otherMigrations",
		"seeds",
		"otherSeeds",
	}
	migrators := make([]Migrator, len(tableNames))
	for i, table := range tableNames {
		migrators[i] = NewMigrator(WithDialect(Postgres), WithTableName(table))
	}
	wg := &sync.WaitGroup{}
	for _, migrator := range migrators {
		wg.Add(3)
		go runCreateMigrationsTable(t, wg, migrator, postgres11DB)
		go runCreateMigrationsTable(t, wg, migrator, postgres11DB)
		go runCreateMigrationsTable(t, wg, migrator, postgres11DB)
	}
	wg.Wait()
}

func runCreateMigrationsTable(t *testing.T, wg *sync.WaitGroup, migrator Migrator, db *sql.DB) {
	err := migrator.createMigrationsTable(db)
	if err != nil {
		t.Error(err)
	}
	wg.Done()
}

func TestPostgres11CreateMigrationsTable(t *testing.T) {
	migrator := NewMigrator(WithDialect(Postgres))
	err := migrator.createMigrationsTable(postgres11DB)
	if err != nil {
		t.Errorf("Error occurred when creating migrations table: %s", err)
	}

	// Test that we can re-run it safely
	err = migrator.createMigrationsTable(postgres11DB)
	if err != nil {
		t.Errorf("Calling createMigrationsTable a second time failed: %s", err)
	}
}

func TestPostgres11MultiStatementMigrations(t *testing.T) {
	tableName := time.Now().Format(time.RFC3339Nano)
	// tableName := "postgres_migrations"
	migrator := NewMigrator(WithDialect(Postgres), WithTableName(tableName))

	migrationSet1 := []*Migration{
		{
			ID: "2019-09-23 Create Artists and Albums",
			Script: `
		CREATE TABLE artists (
			id SERIAL PRIMARY KEY,
			name CHARACTER VARYING (255) NOT NULL DEFAULT ''
		);
		CREATE UNIQUE INDEX idx_artists_name ON artists (name);
		CREATE TABLE albums (
			id SERIAL PRIMARY KEY,
			title CHARACTER VARYING (255) NOT NULL DEFAULT '',
			artist_id INTEGER NOT NULL REFERENCES artists(id)
		);
		`,
		},
	}
	err := migrator.Apply(postgres11DB, migrationSet1)
	if err != nil {
		t.Error(err)
	}

	secondMigratorWithPublicSchema := NewMigrator(WithDialect(Postgres), WithTableName("public", tableName))
	migrationSet2 := []*Migration{
		{
			ID: "2019-09-24 Create Tracks",
			Script: `
		CREATE TABLE tracks (
			id SERIAL PRIMARY KEY,
			name CHARACTER VARYING (255) NOT NULL DEFAULT '',
			artist_id INTEGER NOT NULL REFERENCES artists(id),
			album_id INTEGER NOT NULL REFERENCES albums(id)
		);`,
		},
	}
	err = secondMigratorWithPublicSchema.Apply(postgres11DB, migrationSet2)
	if err != nil {
		t.Error(err)
	}
}
