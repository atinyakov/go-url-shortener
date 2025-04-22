// Package storage provides data models and functions for managing URL records.
// It contains the URLRecord struct, which represents a record for a shortened URL,
// including the original URL, shortened URL, associated user ID, and a deletion flag.
package storage

// URLRecord represents a record for a shortened URL in the storage system.
// It contains the original URL, the shortened URL, the user ID who created the record,
// and a flag indicating whether the record is marked as deleted.
type URLRecord struct {
	ID        string `json:"uuid"`         // The unique identifier for the URL record
	Original  string `json:"original_url"` // The original URL before shortening
	Short     string `json:"short_url"`    // The shortened URL
	UserID    string `json:"user_id"`      // The ID of the user who created the shortened URL
	IsDeleted bool   `json:"is_deleted"`   // A flag indicating if the URL record is deleted
}
