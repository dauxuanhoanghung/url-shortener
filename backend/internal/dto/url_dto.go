package dto

import "time"

type CreateURLRequest struct {
	OriginalURL string   `json:"original_url" binding:"required,url"`
	Tags        []string `json:"tags" binding:"omitempty,max=20,dive,min=1,max=50"`
}

// UpdateURLTagsRequest replaces the full tag list. Omitted or empty tags
// clears all tags for the URL.
type UpdateURLTagsRequest struct {
	Tags []string `json:"tags" binding:"omitempty,max=20,dive,min=1,max=50"`
}

type URLTagsResponse struct {
	Tags []string `json:"tags"`
}

type URLMetadataResponse struct {
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
	OgImage     *string `json:"og_image,omitempty"`
	FaviconURL  *string `json:"favicon_url,omitempty"`
	FetchStatus string  `json:"fetch_status"`
}

type URLResponse struct {
	ID          string               `json:"id"`
	ShortCode   string               `json:"short_code"`
	ShortURL    string               `json:"short_url"`
	OriginalURL string               `json:"original_url"`
	ClickCount  int64                `json:"click_count"`
	CreatedAt   time.Time            `json:"created_at"`
	LastAccess  *time.Time           `json:"last_accessed_at,omitempty"`
	Metadata    *URLMetadataResponse `json:"metadata"`
	Tags        []string             `json:"tags"`
}

type ListURLResponse struct {
	URLs  []URLResponse `json:"urls"`
	Total int           `json:"total"`
}
