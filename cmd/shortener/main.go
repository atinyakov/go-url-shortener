package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
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

func (u *URLResolver) longToShort(url string) string {
	if short, exists := u.ltos[url]; exists {
		return short
	}

	short := u.hashToShort(url)
	u.ltos[url] = short
	u.stol[short] = url
	return short
}

func (u *URLResolver) shortToLong(short string) string {
	if long, exists := u.stol[short]; exists {
		return long
	}
	return ""
}

func handle(resolver *URLResolver) http.HandlerFunc {
	return func(res http.ResponseWriter, req *http.Request) {

		if req.Method == http.MethodPost {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				res.WriteHeader(http.StatusBadRequest)
			}

			fmt.Println(string(body))
			shortURL := resolver.longToShort(string(body))
			fmt.Println(string(shortURL))
			res.WriteHeader(http.StatusCreated)

			_, writeErr := res.Write([]byte("http://localhost:8080/" + shortURL))
			if writeErr != nil {
				res.WriteHeader(http.StatusInternalServerError)
			}
			return
		}

		if req.Method == http.MethodGet {
			// Extract the short URL from the request path
			shortURL := strings.TrimPrefix(req.URL.Path, "/")
			if shortURL == "" {
				http.Error(res, "Short URL is required", http.StatusBadRequest)
				return
			}

			longURL := resolver.shortToLong(shortURL)
			if longURL == "" {
				http.Error(res, "URL not found", http.StatusNotFound)
				return
			}

			fmt.Println("Long", longURL)

			// fmt.Fprintf(res, "Long URL: %s\n", longURL)
			res.Header().Add("Location", longURL)
			res.WriteHeader(http.StatusTemporaryRedirect)
			return
		}

		res.WriteHeader(http.StatusBadRequest)

	}
}

func main() {
	mux := http.NewServeMux()
	resolver := NewURLResolver(8)

	mux.HandleFunc(`/`, handle(resolver))

	err := http.ListenAndServe(`:8080`, mux)
	if err != nil {
		panic(err)
	}
}
