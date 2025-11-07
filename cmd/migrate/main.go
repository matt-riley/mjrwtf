package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/matt-riley/mjrwtf/internal/migrations"
	"github.com/pressly/goose/v3"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	var (
		flags    = flag.NewFlagSet("migrate", flag.ExitOnError)
		dir      = flags.String("dir", "", "directory with migration files")
		dbDriver = flags.String("driver", "", "database driver (sqlite3 or postgres)")
		dbURL    = flags.String("url", "", "database connection string")
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

	// Get database URL from env if not provided
	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
		if *dbURL == "" {
			log.Fatal("DATABASE_URL environment variable or -url flag is required")
		}
	}

	// Auto-detect driver from URL if not provided
	if *dbDriver == "" {
		*dbDriver = detectDriver(*dbURL)
	}

	// Validate driver
	if *dbDriver != "sqlite3" && *dbDriver != "postgres" {
		log.Fatalf("Unsupported database driver: %s (supported: sqlite3, postgres)", *dbDriver)
	}

	// Set up migrations based on driver
	var migrationsFS embed.FS
	if *dir == "" {
		if *dbDriver == "sqlite3" {
			*dir = migrations.SQLiteDir
			migrationsFS = migrations.SQLiteMigrations
		} else {
			*dir = migrations.PostgresDir
			migrationsFS = migrations.PostgresMigrations
		}
	}

	// Open database connection
	db, err := sql.Open(*dbDriver, *dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Set up goose
	goose.SetBaseFS(migrationsFS)
	if err := goose.SetDialect(*dbDriver); err != nil {
		log.Fatalf("Failed to set dialect: %v", err)
	}

	// Run command
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

		// Determine which directory to use for creation
		createDir := "internal/migrations/sqlite"
		if *dbDriver == "postgres" {
			createDir = "internal/migrations/postgres"
		}

		// Use local filesystem for creation (not embedded)
		goose.SetBaseFS(nil)
		if err := goose.Create(db, createDir, name, migrationType); err != nil {
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

// detectDriver attempts to detect the database driver from the connection string
func detectDriver(url string) string {
	switch {
	case strings.HasPrefix(url, "postgres://"), strings.HasPrefix(url, "postgresql://"):
		return "postgres"
	case strings.HasSuffix(url, ".db") || strings.Contains(url, "sqlite"):
		return "sqlite3"
	default:
		return ""
	}
}

func usage() {
	fmt.Print(`migrate - Database migration tool

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
    -driver string
        Database driver (sqlite3 or postgres). Auto-detected if not provided.
    -url string
        Database connection string. Uses DATABASE_URL env var if not provided.
    -dir string
        Directory with migration files. Auto-detected based on driver if not provided.

Examples:
    # Apply migrations (auto-detects from DATABASE_URL)
    migrate up

    # Apply migrations with explicit driver and URL
    migrate -driver sqlite3 -url ./database.db up

    # Rollback last migration
    migrate down

    # Show migration status
    migrate status

    # Create new migration
    migrate create add_users_table

    # Create new migration for specific driver
    migrate -driver postgres create add_users_table

Environment Variables:
    DATABASE_URL    Database connection string (used if -url not provided)
`)
}
