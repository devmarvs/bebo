package main

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/devmarvs/bebo/db"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrConflict           = errors.New("conflict")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type Store struct {
	db     *sql.DB
	helper db.Helper
}

func NewStore(dbConn *sql.DB) *Store {
	return &Store{db: dbConn, helper: db.Helper{Timeout: 2 * time.Second}}
}

func (s *Store) CreateUser(ctx context.Context, email, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	row, cancel := s.helper.QueryRow(ctx, s.db,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 RETURNING id, email, password_hash, created_at`,
		email,
		string(hash),
	)
	defer cancel()

	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrConflict
		}
		return nil, err
	}
	return &user, nil
}

func (s *Store) UserByID(ctx context.Context, id int64) (*User, error) {
	row, cancel := s.helper.QueryRow(ctx, s.db,
		`SELECT id, email, password_hash, created_at FROM users WHERE id = $1`,
		id,
	)
	defer cancel()

	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *Store) userByEmail(ctx context.Context, email string) (*User, error) {
	row, cancel := s.helper.QueryRow(ctx, s.db,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`,
		email,
	)
	defer cancel()

	var user User
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (s *Store) Authenticate(ctx context.Context, email, password string) (*User, error) {
	user, err := s.userByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

func (s *Store) ListNotes(ctx context.Context, userID int64) ([]Note, error) {
	rows, err := s.helper.Query(ctx, s.db,
		`SELECT id, user_id, title, body, created_at, updated_at
		 FROM notes WHERE user_id = $1 ORDER BY created_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var note Note
		if err := rows.Scan(&note.ID, &note.UserID, &note.Title, &note.Body, &note.CreatedAt, &note.UpdatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, note)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return notes, nil
}

func (s *Store) NoteByID(ctx context.Context, userID, noteID int64) (*Note, error) {
	row, cancel := s.helper.QueryRow(ctx, s.db,
		`SELECT id, user_id, title, body, created_at, updated_at
		 FROM notes WHERE id = $1 AND user_id = $2`,
		noteID,
		userID,
	)
	defer cancel()

	var note Note
	if err := row.Scan(&note.ID, &note.UserID, &note.Title, &note.Body, &note.CreatedAt, &note.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &note, nil
}

func (s *Store) CreateNote(ctx context.Context, userID int64, title, body string) (*Note, error) {
	row, cancel := s.helper.QueryRow(ctx, s.db,
		`INSERT INTO notes (user_id, title, body) VALUES ($1, $2, $3)
		 RETURNING id, user_id, title, body, created_at, updated_at`,
		userID,
		title,
		body,
	)
	defer cancel()

	var note Note
	if err := row.Scan(&note.ID, &note.UserID, &note.Title, &note.Body, &note.CreatedAt, &note.UpdatedAt); err != nil {
		return nil, err
	}
	return &note, nil
}

func (s *Store) UpdateNote(ctx context.Context, userID, noteID int64, title, body string) (*Note, error) {
	row, cancel := s.helper.QueryRow(ctx, s.db,
		`UPDATE notes SET title = $1, body = $2, updated_at = NOW()
		 WHERE id = $3 AND user_id = $4
		 RETURNING id, user_id, title, body, created_at, updated_at`,
		title,
		body,
		noteID,
		userID,
	)
	defer cancel()

	var note Note
	if err := row.Scan(&note.ID, &note.UserID, &note.Title, &note.Body, &note.CreatedAt, &note.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &note, nil
}

func (s *Store) DeleteNote(ctx context.Context, userID, noteID int64) error {
	result, err := s.helper.Exec(ctx, s.db,
		`DELETE FROM notes WHERE id = $1 AND user_id = $2`,
		noteID,
		userID,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate key") || strings.Contains(message, "unique constraint")
}
