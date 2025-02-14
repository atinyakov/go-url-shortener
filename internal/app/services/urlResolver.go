package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/atinyakov/go-url-shortener/internal/storage"
)

type URLResolver struct {
	storage           storage.StorageI
	numCharsShortLink int
	elements          string
}

func NewURLResolver(numChars int, storage storage.StorageI) (*URLResolver, error) {
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
	// r, err := u.storage.FindByOriginal(url)
	// if err != nil {
	// 	fmt.Println("HERE !")
	// 	return "", false, err
	// }

	// if r.Short != "" {

	// 	return r.Short, true, nil
	// }

	// short := u.hashToShort(url)

	// collisionCount := 0

	// r, err := u.storage.FindByShort(short)
	// if err != nil {
	// 	return "", false, err
	// }

	// exists := r.Original != ""

	// if exists {
	// 	collisionCount++
	// 	modifiedInput := fmt.Sprintf("%s%d", url, collisionCount)
	// 	short = u.hashToShort(modifiedInput)
	// 	exists = false
	// }

	return u.hashToShort(url)
}

func (u *URLResolver) ShortToLong(short string) string {
	r, err := u.storage.FindByShort(short)
	if err != nil {
		fmt.Println("ShortToLong got error", err.Error())
		return ""
	}

	return r.Original
}
