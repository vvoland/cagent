package page

// ID represents a unique identifier for pages
type ID string

// ChangeMsg is used to change the current page
type ChangeMsg struct {
	ID ID
}
