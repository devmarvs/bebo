package db

// DefaultPageSize is the default page size when none is provided.
const DefaultPageSize = 20

// MaxPageSize is the default maximum page size.
const MaxPageSize = 100

// Pagination describes paging configuration.
type Pagination struct {
	Page    int
	Size    int
	MaxSize int
}

// Normalize applies defaults and bounds.
func (p Pagination) Normalize() Pagination {
	page := p.Page
	if page <= 0 {
		page = 1
	}
	size := p.Size
	if size <= 0 {
		size = DefaultPageSize
	}
	maxSize := p.MaxSize
	if maxSize <= 0 {
		maxSize = MaxPageSize
	}
	if size > maxSize {
		size = maxSize
	}
	return Pagination{Page: page, Size: size, MaxSize: maxSize}
}

// LimitOffset returns SQL limit/offset values.
func (p Pagination) LimitOffset() (int, int) {
	normalized := p.Normalize()
	return normalized.Size, (normalized.Page - 1) * normalized.Size
}
