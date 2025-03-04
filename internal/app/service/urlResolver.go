package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

type URLResolver struct {
	storage           Storage
	numCharsShortLink int
	elements          string
}

func NewURLResolver(numChars int, storage Storage) (*URLResolver, error) {
	return &URLResolver{
		storage:           storage,
		numCharsShortLink: numChars,
		elements:          "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
	}, nil
}

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

func (u *URLResolver) base16ToBase62(hexString string) string {
	var value uint64
	for _, char := range hexString {
		if char >= '0' && char <= '9' {
			value = value*16 + uint64(char-'0')
		} else if char >= 'a' && char <= 'f' {
			value = value*16 + uint64(char-'a'+10)
		}
	}

	// Convert to base62
	var sb []rune
	for value > 0 {
		sb = append([]rune{rune(u.elements[value%62])}, sb...)
		value /= 62
	}

	return string(sb)
}

func (u *URLResolver) LongToShort(url string) string {
	return u.hashToShort(url)
}

func (u *URLResolver) ShortToLong(ctx context.Context, short string) (string, error) {
	r, err := u.storage.FindByShort(ctx, short)

	return r.Original, err
}
