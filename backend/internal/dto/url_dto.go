package dto

import "time"

type CreateURLRequest struct {
	OriginalURL string `json:"original_url" binding:"required,url"`
}

type URLResponse struct {
	ID          string     `json:"id"`
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	ClickCount  int64      `json:"click_count"`
	CreatedAt   time.Time  `json:"created_at"`
	LastAccess  *time.Time `json:"last_accessed_at,omitempty"`
}

type ListURLResponse struct {
	URLs  []URLResponse `json:"urls"`
	Total int           `json:"total"`
}
