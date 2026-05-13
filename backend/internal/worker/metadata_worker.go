package worker

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/dauxuanhoanghung/url-shortener/internal/cache"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
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

// Config bundles all dependencies for the worker pool.
type MetadataWorkerConfig struct {
	MetaRepo repository.URLMetadataRepository
	URLRepo  repository.URLRepository
	Cache    cache.Cache
	Notifier sse.Notifier
	Logger   *zap.Logger
	Workers  int
}

type metadataWorker struct {
	jobs     chan MetadataJob
	metaRepo repository.URLMetadataRepository
	urlRepo  repository.URLRepository
	cache    cache.Cache
	notifier sse.Notifier
	log      *zap.Logger
	wg       sync.WaitGroup
	client   *http.Client
}

func NewMetadataWorker(cfg MetadataWorkerConfig) MetadataWorker {
	n := cfg.Workers
	if n <= 0 {
		n = 4
	}
	return &metadataWorker{
		jobs:     make(chan MetadataJob, n*20),
		metaRepo: cfg.MetaRepo,
		urlRepo:  cfg.URLRepo,
		cache:    cfg.Cache,
		notifier: cfg.Notifier,
		log:      cfg.Logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
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
	// Number of goroutines matches channel buffer / 20.
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
	result := w.fetch(job.TargetURL)

	if err := w.metaRepo.UpdateFetched(ctx, job.MetadataID, result); err != nil {
		w.log.Error("failed to update url metadata", zap.Error(err), zap.String("metadata_id", job.MetadataID.String()))
	}

	switch result.FetchStatus {
	case model.FetchStatusOK:
		// Nothing to do — list response picks up metadata on next poll.

	case model.FetchStatusFailed:
		httpStatus := result.HTTPStatus
		// Only hard-delete on 4xx (dead link). 5xx is transient, 0 = network error.
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

// fetch performs the HTTP GET and parses page metadata from the first 64 KB.
func (w *metadataWorker) fetch(targetURL string) model.FetchResult {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return model.FetchResult{FetchStatus: model.FetchStatusFailed}
	}
	req.Header.Set("User-Agent", "urlshortener-bot/1.0")

	resp, err := w.client.Do(req)
	if err != nil {
		return model.FetchResult{FetchStatus: model.FetchStatusFailed, HTTPStatus: 0}
	}
	defer resp.Body.Close()

	status := int32(resp.StatusCode)

	if resp.StatusCode >= 400 {
		return model.FetchResult{FetchStatus: model.FetchStatusFailed, HTTPStatus: status}
	}

	// 5xx — transient, record but don't delete.
	if resp.StatusCode >= 500 {
		return model.FetchResult{FetchStatus: model.FetchStatusFailed, HTTPStatus: status}
	}

	// 2xx — parse metadata.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	result := model.FetchResult{FetchStatus: model.FetchStatusOK, HTTPStatus: status}
	parseMetadata(body, &result)
	return result
}

// parseMetadata extracts title, description, og:image and favicon from raw HTML bytes.
// No external HTML parser is used — we only need a handful of specific tags.
func parseMetadata(body []byte, r *model.FetchResult) {
	s := string(body)

	if v := extractTag(s, "<title", ">", "</title>"); v != "" {
		r.Title = &v
	}
	if v := extractMeta(s, "name", "description"); v != "" {
		r.Description = &v
	}
	if v := extractMeta(s, "property", "og:image"); v != "" {
		r.OgImage = &v
	}
	if v := extractLinkHref(s, "icon"); v != "" {
		r.FaviconURL = &v
	}
}

// extractTag finds the first occurrence of <tag ... > ... </close> and returns inner text.
func extractTag(s, openPrefix, afterAttr, closeTag string) string {
	i := strings.Index(strings.ToLower(s), strings.ToLower(openPrefix))
	if i < 0 {
		return ""
	}
	j := strings.Index(s[i:], afterAttr)
	if j < 0 {
		return ""
	}
	start := i + j + len(afterAttr)
	k := strings.Index(strings.ToLower(s[start:]), strings.ToLower(closeTag))
	if k < 0 {
		return ""
	}
	return strings.TrimSpace(s[start : start+k])
}

// extractMeta finds <meta name/property="key" content="value"> or reversed attr order.
func extractMeta(s, attr, key string) string {
	lower := strings.ToLower(s)
	search := attr + `="` + strings.ToLower(key) + `"`
	idx := strings.Index(lower, search)
	if idx < 0 {
		return ""
	}
	// Find the enclosing <meta ...> tag boundary (scan back for '<').
	tagStart := strings.LastIndex(lower[:idx], "<meta")
	if tagStart < 0 {
		return ""
	}
	tagEnd := strings.Index(lower[tagStart:], ">")
	if tagEnd < 0 {
		return ""
	}
	tag := s[tagStart : tagStart+tagEnd+1]
	return extractAttrValue(tag, "content")
}

// extractLinkHref finds <link rel="icon" href="..."> (any rel containing the word).
func extractLinkHref(s, relContains string) string {
	lower := strings.ToLower(s)
	search := `rel="`
	offset := 0
	for {
		idx := strings.Index(lower[offset:], search)
		if idx < 0 {
			break
		}
		abs := offset + idx
		valStart := abs + len(search)
		valEnd := strings.Index(lower[valStart:], `"`)
		if valEnd < 0 {
			break
		}
		relVal := lower[valStart : valStart+valEnd]
		if strings.Contains(relVal, relContains) {
			tagStart := strings.LastIndex(lower[:abs], "<link")
			if tagStart >= 0 {
				tagEnd := strings.Index(lower[tagStart:], ">")
				if tagEnd >= 0 {
					tag := s[tagStart : tagStart+tagEnd+1]
					if v := extractAttrValue(tag, "href"); v != "" {
						return v
					}
				}
			}
		}
		offset = valStart + valEnd + 1
	}
	return ""
}

// extractAttrValue returns the value of attr="..." within a single tag string.
func extractAttrValue(tag, attr string) string {
	lower := strings.ToLower(tag)
	search := attr + `="`
	idx := strings.Index(lower, search)
	if idx < 0 {
		return ""
	}
	start := idx + len(search)
	end := strings.Index(tag[start:], `"`)
	if end < 0 {
		return ""
	}
	return strings.TrimSpace(tag[start : start+end])
}

