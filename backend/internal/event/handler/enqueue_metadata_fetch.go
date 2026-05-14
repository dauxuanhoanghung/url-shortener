package handler

import (
	"context"

	"github.com/dauxuanhoanghung/url-shortener/internal/event"
	"github.com/dauxuanhoanghung/url-shortener/internal/worker"
)

// EnqueueMetadataFetch returns an event.HandlerFunc that submits a metadata
// fetch job to the worker pool when a new URL is created.
func EnqueueMetadataFetch(w worker.MetadataWorker) event.HandlerFunc {
	return func(ctx context.Context, ev any) error {
		e := ev.(event.URLCreated)
		w.Submit(worker.MetadataJob{
			MetadataID: e.MetadataID,
			URLID:      e.URL.ID,
			ShortCode:  e.URL.ShortCode,
			UserID:     e.UserID,
			TargetURL:  e.URL.OriginalURL,
		})
		return nil
	}
}
