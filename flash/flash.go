package flash

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/devmarvs/bebo/session"
)

// ErrStoreMissing indicates the flash store is not configured.
var ErrStoreMissing = errors.New("flash store missing")

// Message represents a flash message.
type Message struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Store persists flash messages in a session store.
type Store struct {
	Store session.Store
	Key   string
}

// New creates a flash store with default key.
func New(store session.Store) Store {
	return Store{Store: store, Key: "bebo_flash"}
}

// Add appends a flash message and saves the session.
func (s Store) Add(w http.ResponseWriter, r *http.Request, msg Message) error {
	sess, err := s.getSession(r)
	if err != nil {
		return err
	}
	messages, _ := decodeMessages(sess.Get(s.Key))
	messages = append(messages, msg)

	encoded, err := encodeMessages(messages)
	if err != nil {
		return err
	}
	sess.Set(s.Key, encoded)
	return sess.Save(w)
}

// Peek returns flash messages without clearing them.
func (s Store) Peek(r *http.Request) ([]Message, error) {
	sess, err := s.getSession(r)
	if err != nil {
		return nil, err
	}
	return decodeMessages(sess.Get(s.Key))
}

// Pop returns flash messages and clears them from the session.
func (s Store) Pop(w http.ResponseWriter, r *http.Request) ([]Message, error) {
	sess, err := s.getSession(r)
	if err != nil {
		return nil, err
	}
	messages, decodeErr := decodeMessages(sess.Get(s.Key))
	sess.Delete(s.Key)
	if saveErr := sess.Save(w); saveErr != nil {
		return nil, saveErr
	}
	if decodeErr != nil {
		return nil, decodeErr
	}
	return messages, nil
}

func (s Store) getSession(r *http.Request) (*session.Session, error) {
	if s.Store == nil {
		return nil, ErrStoreMissing
	}
	if s.Key == "" {
		s.Key = "bebo_flash"
	}
	sess, err := s.Store.Get(r)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func encodeMessages(messages []Message) (string, error) {
	payload, err := json.Marshal(messages)
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func decodeMessages(value string) ([]Message, error) {
	if value == "" {
		return nil, nil
	}
	var messages []Message
	if err := json.Unmarshal([]byte(value), &messages); err != nil {
		return nil, err
	}
	return messages, nil
}
