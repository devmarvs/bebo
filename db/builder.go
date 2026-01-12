package db

import (
	"errors"
	"strconv"
	"strings"
)

// Dialect configures SQL placeholder style.
type Dialect int

const (
	// DialectQuestion uses ? placeholders.
	DialectQuestion Dialect = iota
	// DialectDollar uses $1 style placeholders.
	DialectDollar
)

var (
	// ErrMissingTable indicates a missing table name.
	ErrMissingTable = errors.New("table required")
	// ErrMissingColumns indicates a missing column list.
	ErrMissingColumns = errors.New("columns required")
	// ErrMissingValues indicates missing values for insert or update.
	ErrMissingValues = errors.New("values required")
	// ErrMismatchedValues indicates values count mismatch.
	ErrMismatchedValues = errors.New("values count does not match columns")
)

// SelectBuilder builds SELECT queries.
type SelectBuilder struct {
	dialect Dialect
	columns []string
	table   string
	where   []string
	args    []any
	orderBy string
	limit   *int
	offset  *int
}

// Select starts a SELECT query.
func Select(columns ...string) SelectBuilder {
	return SelectBuilder{columns: columns, dialect: DialectQuestion}
}

// Dialect sets the SQL dialect.
func (b SelectBuilder) Dialect(dialect Dialect) SelectBuilder {
	b.dialect = dialect
	return b
}

// From sets the source table.
func (b SelectBuilder) From(table string) SelectBuilder {
	b.table = table
	return b
}

// Where adds a WHERE clause.
func (b SelectBuilder) Where(condition string, args ...any) SelectBuilder {
	if condition != "" {
		b.where = append(b.where, condition)
		b.args = append(b.args, args...)
	}
	return b
}

// OrderBy sets the ORDER BY clause.
func (b SelectBuilder) OrderBy(order string) SelectBuilder {
	b.orderBy = order
	return b
}

// Limit sets the LIMIT clause.
func (b SelectBuilder) Limit(limit int) SelectBuilder {
	if limit >= 0 {
		b.limit = &limit
	}
	return b
}

// Offset sets the OFFSET clause.
func (b SelectBuilder) Offset(offset int) SelectBuilder {
	if offset >= 0 {
		b.offset = &offset
	}
	return b
}

// Build returns the SQL string and args.
func (b SelectBuilder) Build() (string, []any, error) {
	if b.table == "" {
		return "", nil, ErrMissingTable
	}

	columns := b.columns
	if len(columns) == 0 {
		columns = []string{"*"}
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(columns, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(b.table)
	if len(b.where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(b.where, " AND "))
	}
	if b.orderBy != "" {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(b.orderBy)
	}
	if b.limit != nil {
		sb.WriteString(" LIMIT ")
		sb.WriteString(strconv.Itoa(*b.limit))
	}
	if b.offset != nil {
		sb.WriteString(" OFFSET ")
		sb.WriteString(strconv.Itoa(*b.offset))
	}

	query := normalizePlaceholders(sb.String(), b.dialect)
	return query, b.args, nil
}

// InsertBuilder builds INSERT queries.
type InsertBuilder struct {
	dialect Dialect
	table   string
	columns []string
	values  []any
}

// Insert starts an INSERT query.
func Insert(table string) InsertBuilder {
	return InsertBuilder{table: table, dialect: DialectQuestion}
}

// Dialect sets the SQL dialect.
func (b InsertBuilder) Dialect(dialect Dialect) InsertBuilder {
	b.dialect = dialect
	return b
}

// Columns sets the column list.
func (b InsertBuilder) Columns(columns ...string) InsertBuilder {
	b.columns = append([]string{}, columns...)
	return b
}

// Values sets the insert values.
func (b InsertBuilder) Values(values ...any) InsertBuilder {
	b.values = append([]any{}, values...)
	return b
}

// Build returns the SQL string and args.
func (b InsertBuilder) Build() (string, []any, error) {
	if b.table == "" {
		return "", nil, ErrMissingTable
	}
	if len(b.columns) == 0 {
		return "", nil, ErrMissingColumns
	}
	if len(b.values) == 0 {
		return "", nil, ErrMissingValues
	}
	if len(b.values) != len(b.columns) {
		return "", nil, ErrMismatchedValues
	}

	placeholders := make([]string, len(b.columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := "INSERT INTO " + b.table + " (" + strings.Join(b.columns, ", ") + ") VALUES (" + strings.Join(placeholders, ", ") + ")"
	query = normalizePlaceholders(query, b.dialect)
	return query, append([]any{}, b.values...), nil
}

// UpdateBuilder builds UPDATE queries.
type UpdateBuilder struct {
	dialect   Dialect
	table     string
	sets      []string
	setArgs   []any
	where     []string
	whereArgs []any
}

// Update starts an UPDATE query.
func Update(table string) UpdateBuilder {
	return UpdateBuilder{table: table, dialect: DialectQuestion}
}

// Dialect sets the SQL dialect.
func (b UpdateBuilder) Dialect(dialect Dialect) UpdateBuilder {
	b.dialect = dialect
	return b
}

// Set adds a column assignment.
func (b UpdateBuilder) Set(column string, value any) UpdateBuilder {
	if column != "" {
		b.sets = append(b.sets, column+" = ?")
		b.setArgs = append(b.setArgs, value)
	}
	return b
}

// Where adds a WHERE clause.
func (b UpdateBuilder) Where(condition string, args ...any) UpdateBuilder {
	if condition != "" {
		b.where = append(b.where, condition)
		b.whereArgs = append(b.whereArgs, args...)
	}
	return b
}

// Build returns the SQL string and args.
func (b UpdateBuilder) Build() (string, []any, error) {
	if b.table == "" {
		return "", nil, ErrMissingTable
	}
	if len(b.sets) == 0 {
		return "", nil, ErrMissingValues
	}

	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(b.table)
	sb.WriteString(" SET ")
	sb.WriteString(strings.Join(b.sets, ", "))
	if len(b.where) > 0 {
		sb.WriteString(" WHERE ")
		sb.WriteString(strings.Join(b.where, " AND "))
	}

	query := normalizePlaceholders(sb.String(), b.dialect)
	args := make([]any, 0, len(b.setArgs)+len(b.whereArgs))
	args = append(args, b.setArgs...)
	args = append(args, b.whereArgs...)
	return query, args, nil
}

func normalizePlaceholders(query string, dialect Dialect) string {
	if dialect != DialectDollar {
		return query
	}
	var sb strings.Builder
	sb.Grow(len(query))
	idx := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			sb.WriteString("$")
			sb.WriteString(strconv.Itoa(idx))
			idx++
			continue
		}
		sb.WriteByte(query[i])
	}
	return sb.String()
}
