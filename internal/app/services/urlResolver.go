package services

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

type URLResolver struct {
	numCharsShortLink int
	elements          string
	ltos              map[string]string
	stol              map[string]string
}

func NewURLResolver(numChars int) *URLResolver {
	return &URLResolver{
		numCharsShortLink: numChars,
		elements:          "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		ltos:              make(map[string]string),
		stol:              make(map[string]string),
	}
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
	if short, exists := u.ltos[url]; exists {
		return short
	}

	short := u.hashToShort(url)

	collisionCount := 0
	for _, exists := u.stol[short]; exists; {
		collisionCount++
		modifiedInput := fmt.Sprintf("%s%d", url, collisionCount)
		short = u.hashToShort(modifiedInput)
	}

	u.ltos[url] = short
	u.stol[short] = url
	return short
}

func (u *URLResolver) ShortToLong(short string) string {
	if long, exists := u.stol[short]; exists {
		return long
	}
	return ""
}
