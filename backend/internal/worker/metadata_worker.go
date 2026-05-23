package worker

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/dauxuanhoanghung/url-shortener/internal/cache"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/dauxuanhoanghung/url-shortener/internal/service"
	"github.com/dauxuanhoanghung/url-shortener/internal/sse"
	"github.com/google/uuid"
)

// MetadataJob is submitted by URLService after a short URL is persisted.
type MetadataJob struct {
	MetadataID uuid.UUID
	URLID      uuid.UUID
	ShortCode  string
	UserID     uuid.UUID
	TargetURL  string
}

// MetadataWorker is the interface URLService depends on.
type MetadataWorker interface {
	Submit(job MetadataJob)
	Start(ctx context.Context)
	Stop()
}

// MetadataWorkerConfig bundles all dependencies for the worker pool.
type MetadataWorkerConfig struct {
	MetaRepo repository.URLMetadataRepository
	URLRepo  repository.URLRepository
	Cache    cache.Cache
	Notifier sse.Notifier
	Fetcher  service.MetadataFetcher
	Logger   *zap.Logger
	Workers  int
}

type metadataWorker struct {
	jobs     chan MetadataJob
	metaRepo repository.URLMetadataRepository
	urlRepo  repository.URLRepository
	cache    cache.Cache
	notifier sse.Notifier
	fetcher  service.MetadataFetcher
	log      *zap.Logger
	wg       sync.WaitGroup
}

func NewMetadataWorker(cfg MetadataWorkerConfig) MetadataWorker {
	n := cfg.Workers
	if n <= 0 {
		n = 4
	}
	fetcher := cfg.Fetcher
	if fetcher == nil {
		fetcher = service.NewMetadataFetcher()
	}
	return &metadataWorker{
		jobs:     make(chan MetadataJob, n*20),
		metaRepo: cfg.MetaRepo,
		urlRepo:  cfg.URLRepo,
		cache:    cfg.Cache,
		notifier: cfg.Notifier,
		fetcher:  fetcher,
		log:      cfg.Logger,
	}
}

func (w *metadataWorker) Submit(job MetadataJob) {
	select {
	case w.jobs <- job:
	default:
		w.log.Warn("metadata worker queue full, dropping job",
			zap.String("url_id", job.URLID.String()),
			zap.String("target", job.TargetURL),
		)
	}
}

func (w *metadataWorker) Start(ctx context.Context) {
	n := cap(w.jobs) / 20
	if n <= 0 {
		n = 4
	}
	for i := 0; i < n; i++ {
		w.wg.Add(1)
		go func() {
			defer w.wg.Done()
			for job := range w.jobs {
				w.process(ctx, job)
			}
		}()
	}
}

func (w *metadataWorker) Stop() {
	close(w.jobs)
	w.wg.Wait()
}

func (w *metadataWorker) process(ctx context.Context, job MetadataJob) {
	result := w.fetcher.Fetch(job.TargetURL)

	if err := w.metaRepo.UpdateFetched(ctx, job.MetadataID, result); err != nil {
		w.log.Error("failed to update url metadata", zap.Error(err), zap.String("metadata_id", job.MetadataID.String()))
	}

	switch result.FetchStatus {
	case model.FetchStatusOK:
		w.notifier.Notify(job.UserID, sse.Event{
			Type: "metadata_updated",
			Data: map[string]any{
				"url_id":      job.URLID.String(),
				"short_code":  job.ShortCode,
				"title":       result.Title,
				"description": result.Description,
				"og_image":    result.OgImage,
				"favicon_url": result.FaviconURL,
				"fetch_status": string(model.FetchStatusOK),
			},
		})

	case model.FetchStatusFailed:
		httpStatus := result.HTTPStatus
		// Only soft-delete on 4xx (dead link). 5xx is transient, 0 = network error.
		if httpStatus == 0 || (httpStatus >= 400 && httpStatus < 500) {
			if err := w.urlRepo.SoftDeleteByID(ctx, job.URLID); err != nil {
				w.log.Error("failed to soft-delete dead url", zap.Error(err), zap.String("url_id", job.URLID.String()))
			}
			_ = w.cache.Delete(ctx, "url:"+job.ShortCode)

			w.notifier.Notify(job.UserID, sse.Event{
				Type: "url_deleted",
				Data: map[string]any{
					"url_id":      job.URLID.String(),
					"short_code":  job.ShortCode,
					"reason":      "origin_unreachable",
					"http_status": httpStatus,
				},
			})
		}
	}
}
