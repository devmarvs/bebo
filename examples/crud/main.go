package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/devmarvs/bebo/config"
	"github.com/devmarvs/bebo/db"
	"github.com/devmarvs/bebo/migrate"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	shouldMigrate := flag.Bool("migrate", false, "run migrations on startup")
	flag.Parse()

	cfg := loadConfig()

	dbConn, err := openDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer dbConn.Close()

	if *shouldMigrate || cfg.AutoMigrate {
		if err := runMigrations(dbConn, migrationsDir()); err != nil {
			log.Fatalf("migrate: %v", err)
		}
	}

	app := NewApp(dbConn, cfg)
	if err := app.RunWithSignals(); err != nil {
		log.Fatalf("run: %v", err)
	}
}

func loadConfig() AppConfig {
	appCfg := config.LoadFromEnv("BEBO_", config.Default())
	databaseURL := envString("BEBO_DATABASE_URL", "")
	if databaseURL == "" {
		databaseURL = envString("DATABASE_URL", "")
	}
	if databaseURL == "" {
		databaseURL = "postgres://postgres:postgres@localhost:5432/bebo_crud?sslmode=disable"
	}

	sessionKey := envString("BEBO_SESSION_KEY", "")
	if sessionKey == "" {
		sessionKey = "dev-session-key-change-me"
		log.Print("warning: using default session key")
	}

	secureCookies := envBool("BEBO_SECURE_COOKIES", false)
	autoMigrate := envBool("BEBO_AUTO_MIGRATE", false)

	return AppConfig{
		App:           appCfg,
		DatabaseURL:   databaseURL,
		SessionKey:    []byte(sessionKey),
		SecureCookies: secureCookies,
		AutoMigrate:   autoMigrate,
	}
}

func openDB(dsn string) (*sql.DB, error) {
	return db.Open("pgx", dsn, db.Options{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: time.Hour,
		PingTimeout:     5 * time.Second,
	})
}

func runMigrations(dbConn *sql.DB, dir string) error {
	runner := migrate.New(dbConn, dir)
	runner.Locker = migrate.AdvisoryLocker{ID: 42, Timeout: 5 * time.Second}
	_, err := runner.Up(context.Background())
	return err
}

func migrationsDir() string {
	if _, err := os.Stat("migrations"); err == nil {
		return "migrations"
	}
	return filepath.Join("examples", "crud", "migrations")
}

func envString(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func envBool(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
