package domain

// SortOrder controls ordering direction for ordered query results.
type SortOrder string

const (
	SortAscending  SortOrder = "asc"
	SortDescending SortOrder = "desc"
)
