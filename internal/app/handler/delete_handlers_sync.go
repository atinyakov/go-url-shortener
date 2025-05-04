//go:build test

package handler

import (
	"context"

	"github.com/atinyakov/go-url-shortener/internal/storage"
)

func callDeleteURLRecords(service URLServiceIface, ctx context.Context, records []storage.URLRecord) {
	service.DeleteURLRecords(ctx, records)
}
