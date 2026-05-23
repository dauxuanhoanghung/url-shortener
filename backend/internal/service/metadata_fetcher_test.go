package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
)

// ── parseMetadata ─────────────────────────────────────────────────────────────

func TestParseMetadata_Title(t *testing.T) {
	body := []byte(`<html><head><title>Hello World</title></head></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.Title == nil || *r.Title != "Hello World" {
		t.Errorf("title: got %v want %q", r.Title, "Hello World")
	}
}

func TestParseMetadata_Description(t *testing.T) {
	body := []byte(`<html><head><meta name="description" content="Great page"></head></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.Description == nil || *r.Description != "Great page" {
		t.Errorf("description: got %v want %q", r.Description, "Great page")
	}
}

func TestParseMetadata_OgImage(t *testing.T) {
	body := []byte(`<html><head><meta property="og:image" content="https://example.com/img.png"></head></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.OgImage == nil || *r.OgImage != "https://example.com/img.png" {
		t.Errorf("og:image: got %v want %q", r.OgImage, "https://example.com/img.png")
	}
}

func TestParseMetadata_Favicon(t *testing.T) {
	body := []byte(`<html><head><link rel="icon" href="/favicon.ico"></head></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.FaviconURL == nil || *r.FaviconURL != "/favicon.ico" {
		t.Errorf("favicon: got %v want %q", r.FaviconURL, "/favicon.ico")
	}
}

func TestParseMetadata_AllFields(t *testing.T) {
	body := []byte(`<!DOCTYPE html><html><head>
		<title>  My Page  </title>
		<meta name="description" content="A description">
		<meta property="og:image" content="https://img.example.com/og.png">
		<link rel="shortcut icon" href="/static/favicon.ico">
	</head><body></body></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.Title == nil || *r.Title != "My Page" {
		t.Errorf("title: got %v", r.Title)
	}
	if r.Description == nil || *r.Description != "A description" {
		t.Errorf("description: got %v", r.Description)
	}
	if r.OgImage == nil || *r.OgImage != "https://img.example.com/og.png" {
		t.Errorf("og:image: got %v", r.OgImage)
	}
	if r.FaviconURL == nil || *r.FaviconURL != "/static/favicon.ico" {
		t.Errorf("favicon: got %v", r.FaviconURL)
	}
}

func TestParseMetadata_EmptyBody(t *testing.T) {
	var r model.FetchResult
	parseMetadata([]byte(""), &r)
	if r.Title != nil || r.Description != nil || r.OgImage != nil || r.FaviconURL != nil {
		t.Error("expected all fields nil for empty body")
	}
}

func TestParseMetadata_MissingFields(t *testing.T) {
	body := []byte(`<html><head></head><body>Just text</body></html>`)
	var r model.FetchResult
	parseMetadata(body, &r)
	if r.Title != nil {
		t.Errorf("expected nil title, got %q", *r.Title)
	}
}

// ── MetadataFetcher via httptest ──────────────────────────────────────────────

func TestMetadataFetcher_200_ParsesMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><head><title>Test Page</title></head></html>`))
	}))
	defer srv.Close()

	f := NewMetadataFetcher()
	result := f.Fetch(srv.URL)

	if result.FetchStatus != model.FetchStatusOK {
		t.Errorf("fetch_status: got %q want ok", result.FetchStatus)
	}
	if result.HTTPStatus != 200 {
		t.Errorf("http_status: got %d want 200", result.HTTPStatus)
	}
	if result.Title == nil || *result.Title != "Test Page" {
		t.Errorf("title: got %v want %q", result.Title, "Test Page")
	}
}

func TestMetadataFetcher_404_ReturnsFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	result := NewMetadataFetcher().Fetch(srv.URL)

	if result.FetchStatus != model.FetchStatusFailed {
		t.Errorf("fetch_status: got %q want failed", result.FetchStatus)
	}
	if result.HTTPStatus != 404 {
		t.Errorf("http_status: got %d want 404", result.HTTPStatus)
	}
}

func TestMetadataFetcher_500_ReturnsFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	result := NewMetadataFetcher().Fetch(srv.URL)

	if result.FetchStatus != model.FetchStatusFailed {
		t.Errorf("fetch_status: got %q want failed", result.FetchStatus)
	}
}

func TestMetadataFetcher_NetworkError_ReturnsFailed(t *testing.T) {
	result := NewMetadataFetcher().Fetch("http://127.0.0.1:1")
	if result.FetchStatus != model.FetchStatusFailed {
		t.Errorf("fetch_status: got %q want failed", result.FetchStatus)
	}
}

func TestMetadataFetcher_InvalidURL_ReturnsFailed(t *testing.T) {
	result := NewMetadataFetcher().Fetch("://not-a-url")
	if result.FetchStatus != model.FetchStatusFailed {
		t.Errorf("fetch_status: got %q want failed", result.FetchStatus)
	}
}

func TestMetadataFetcher_SetsUserAgent(t *testing.T) {
	var gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA = r.Header.Get("User-Agent")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	NewMetadataFetcher().Fetch(srv.URL)

	if gotUA != "urlshortener-bot/1.0" {
		t.Errorf("user-agent: got %q want %q", gotUA, "urlshortener-bot/1.0")
	}
}
