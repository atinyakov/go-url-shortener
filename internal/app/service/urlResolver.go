// Package service provides functionality for URL shortening and resolution.
package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

// URLResolver is a service that handles URL shortening and resolution.
// It generates short links using a hashing mechanism and stores them in a storage backend.
type URLResolver struct {
	storage           Storage // Storage backend for URL resolution.
	numCharsShortLink int     // The desired length of the shortened URL.
	elements          string  // Base62 encoding elements (0-9, a-z, A-Z).
}

// NewURLResolver creates a new URLResolver instance with the given number of characters for the short link.
// It initializes the resolver with the provided storage backend.
func NewURLResolver(numChars int, storage Storage) (*URLResolver, error) {
	return &URLResolver{
		storage:           storage,
		numCharsShortLink: numChars,
		elements:          "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
	}, nil
}

// hashToShort generates a short URL by hashing the original URL with SHA-256 and then encoding it in Base62.
// It truncates the result to the specified length of the short URL.
func (u *URLResolver) hashToShort(url string) string {
	// Hash the URL using SHA-256
	hash := sha256.Sum256([]byte(url))
	hexHash := hex.EncodeToString(hash[:])

	// Base62 encode the hash
	base62 := u.base16ToBase62(hexHash)

	// Truncate to the desired length
	shortURL := base62[:u.numCharsShortLink]
	return shortURL
}

// base16ToBase62 converts a hexadecimal string to a Base62 encoded string.
func (u *URLResolver) base16ToBase62(hexString string) string {
	var value uint64
	for _, char := range hexString {
		if char >= '0' && char <= '9' {
			value = value*16 + uint64(char-'0')
		} else if char >= 'a' && char <= 'f' {
			value = value*16 + uint64(char-'a'+10)
		}
	}

	// Convert to Base62
	var sb []rune
	for value > 0 {
		sb = append([]rune{rune(u.elements[value%62])}, sb...)
		value /= 62
	}

	return string(sb)
}

// LongToShort converts a long URL to a shortened URL by hashing it and encoding the hash in Base62.
func (u *URLResolver) LongToShort(url string) string {
	return u.hashToShort(url)
}

// ShortToLong resolves a shortened URL to its original form by querying the storage backend.
func (u *URLResolver) ShortToLong(ctx context.Context, short string) (string, error) {
	r, err := u.storage.FindByShort(ctx, short)

	return r.Original, err
}
