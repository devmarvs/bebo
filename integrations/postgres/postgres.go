package postgres

import (
	"database/sql"

	"github.com/devmarvs/bebo/db"
	"github.com/devmarvs/bebo/session"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const Driver = "pgx"

func Open(dsn string, options db.Options) (*sql.DB, error) {
	return db.Open(Driver, dsn, options)
}

type SessionStore = session.PostgresStore
type SessionOptions = session.PostgresOptions

func NewSessionStore(options SessionOptions) (*SessionStore, error) {
	return session.NewPostgresStore(options)
}
