package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/database"
	"github.com/matt-riley/mjrwtf/internal/migrations"
	"github.com/pressly/goose/v3"

	_ "github.com/mattn/go-sqlite3"
)

const driverName = "sqlite3"

func main() {
	var (
		flags = flag.NewFlagSet("migrate", flag.ExitOnError)
		dir   = flags.String("dir", "", "directory with migration files")
		dbURL = flags.String("url", "", "database connection string")
	)

	flags.Usage = usage
	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}

	args := flags.Args()
	if len(args) < 1 {
		flags.Usage()
		os.Exit(1)
	}

	command := args[0]

	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
		if *dbURL == "" {
			log.Fatal("DATABASE_URL environment variable or -url flag is required")
		}
	}

	var migrationsFS embed.FS
	useEmbeddedFS := false
	if *dir == "" {
		useEmbeddedFS = true
		*dir = migrations.SQLiteDir
		migrationsFS = migrations.SQLiteMigrations
	}

	dsn := database.NormalizeSQLiteDSN(*dbURL)

	db, err := sql.Open(driverName, dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if useEmbeddedFS {
		goose.SetBaseFS(migrationsFS)
	}
	if err := goose.SetDialect(driverName); err != nil {
		log.Fatalf("Failed to set dialect: %v", err)
	}

	switch command {
	case "up":
		if err := goose.Up(db, *dir); err != nil {
			log.Fatalf("Migration up failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "down":
		if err := goose.Down(db, *dir); err != nil {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("Migration rollback completed successfully")

	case "status":
		if err := goose.Status(db, *dir); err != nil {
			log.Fatalf("Migration status failed: %v", err)
		}

	case "version":
		version, err := goose.GetDBVersion(db)
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Current version: %d\n", version)

	case "create":
		if len(args) < 2 {
			log.Fatal("create command requires migration name")
		}
		name := args[1]
		migrationType := "sql"
		if len(args) > 2 {
			migrationType = args[2]
		}

		createDir := "internal/migrations/sqlite"

		if useEmbeddedFS {
			originalFS := migrationsFS
			goose.SetBaseFS(nil)
			defer goose.SetBaseFS(originalFS)
		}

		err := goose.Create(db, createDir, name, migrationType)
		if err != nil {
			log.Fatalf("Failed to create migration: %v", err)
		}
		fmt.Printf("Created migration in %s\n", createDir)

	case "reset":
		if err := goose.Reset(db, *dir); err != nil {
			log.Fatalf("Migration reset failed: %v", err)
		}
		fmt.Println("Database reset completed successfully")

	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

func usage() {
	fmt.Print(`migrate - Database migration tool (SQLite only)

Usage:
    migrate [flags] <command> [args]

Commands:
    up          Apply all pending migrations
    down        Rollback the most recent migration
    status      Show migration status
    version     Show current migration version
    create NAME [TYPE]  Create a new migration file (TYPE: sql or go, default: sql)
    reset       Rollback all migrations

Flags:
    -url string
        Database connection string. Uses DATABASE_URL env var if not provided.
    -dir string
        Directory with migration files. Defaults to embedded SQLite migrations if not provided.

Examples:
    # Apply migrations (uses DATABASE_URL)
    migrate up

    # Apply migrations with explicit URL
    migrate -url ./database.db up

    # Create new migration
    migrate create add_users_table

Environment Variables:
    DATABASE_URL    Database connection string (used if -url not provided)
`)
}
