package db

import "testing"

func TestSelectBuilder(t *testing.T) {
	query, args, err := Select("id", "name").
		From("users").
		Where("active = ?", true).
		OrderBy("id DESC").
		Limit(10).
		Offset(20).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if query != "SELECT id, name FROM users WHERE active = ? ORDER BY id DESC LIMIT 10 OFFSET 20" {
		t.Fatalf("unexpected query: %s", query)
	}
	if len(args) != 1 || args[0] != true {
		t.Fatalf("unexpected args")
	}
}

func TestSelectBuilderDialect(t *testing.T) {
	query, args, err := Select("*").
		From("users").
		Where("id = ?", 10).
		Dialect(DialectDollar).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if query != "SELECT * FROM users WHERE id = $1" {
		t.Fatalf("unexpected query: %s", query)
	}
	if len(args) != 1 || args[0] != 10 {
		t.Fatalf("unexpected args")
	}
}

func TestInsertBuilder(t *testing.T) {
	query, args, err := Insert("users").
		Columns("name", "email").
		Values("a", "b").
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if query != "INSERT INTO users (name, email) VALUES (?, ?)" {
		t.Fatalf("unexpected query: %s", query)
	}
	if len(args) != 2 || args[0] != "a" || args[1] != "b" {
		t.Fatalf("unexpected args")
	}
}

func TestUpdateBuilder(t *testing.T) {
	query, args, err := Update("users").
		Set("name", "a").
		Set("email", "b").
		Where("id = ?", 5).
		Build()
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if query != "UPDATE users SET name = ?, email = ? WHERE id = ?" {
		t.Fatalf("unexpected query: %s", query)
	}
	if len(args) != 3 || args[0] != "a" || args[1] != "b" || args[2] != 5 {
		t.Fatalf("unexpected args")
	}
}

func TestInsertBuilderErrors(t *testing.T) {
	_, _, err := Insert("").Columns("name").Values("a").Build()
	if err != ErrMissingTable {
		t.Fatalf("expected missing table error")
	}
	_, _, err = Insert("users").Build()
	if err != ErrMissingColumns {
		t.Fatalf("expected missing columns error")
	}
	_, _, err = Insert("users").Columns("name").Build()
	if err != ErrMissingValues {
		t.Fatalf("expected missing values error")
	}
	_, _, err = Insert("users").Columns("name").Values("a", "b").Build()
	if err != ErrMismatchedValues {
		t.Fatalf("expected mismatched values error")
	}
}
