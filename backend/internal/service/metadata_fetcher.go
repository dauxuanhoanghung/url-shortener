package service

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
)

// MetadataFetcher fetches and parses page metadata from a URL.
type MetadataFetcher interface {
	Fetch(targetURL string) model.FetchResult
}

type metadataFetcher struct {
	client *http.Client
}

func NewMetadataFetcher() MetadataFetcher {
	return &metadataFetcher{
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

func (f *metadataFetcher) Fetch(targetURL string) model.FetchResult {
	req, err := http.NewRequest(http.MethodGet, targetURL, nil)
	if err != nil {
		return model.FetchResult{FetchStatus: model.FetchStatusFailed}
	}
	req.Header.Set("User-Agent", "urlshortener-bot/1.0")

	resp, err := f.client.Do(req)
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
	resolveRelativeURLs(targetURL, &result)
	return result
}

// resolveRelativeURLs converts relative URLs in the result (e.g. /favicon.ico) to absolute
// ones using the base URL of the fetched page.
func resolveRelativeURLs(baseURL string, r *model.FetchResult) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	resolve := func(raw *string) {
		if raw == nil || *raw == "" {
			return
		}
		ref, err := url.Parse(*raw)
		if err != nil || ref.IsAbs() {
			return
		}
		abs := base.ResolveReference(ref).String()
		*raw = abs
	}
	resolve(r.FaviconURL)
	resolve(r.OgImage)
}

// parseMetadata extracts title, description, og:image and favicon from raw HTML bytes.
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

func extractMeta(s, attr, key string) string {
	lower := strings.ToLower(s)
	search := attr + `="` + strings.ToLower(key) + `"`
	idx := strings.Index(lower, search)
	if idx < 0 {
		return ""
	}
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
