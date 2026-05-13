package dto

type PlanResponse struct {
	Code                   string          `json:"code"`
	Name                   string          `json:"name"`
	PriceCents             int32           `json:"price_cents"`
	MaxURLs                int32           `json:"max_urls"`
	MaxDomains             int32           `json:"max_domains"`
	MaxTeamMembers         int32           `json:"max_team_members"`
	AnalyticsRetentionDays int32           `json:"analytics_retention_days"`
	APIRateLimitPerMin     *int32          `json:"api_rate_limit_per_min"`
	Features               map[string]bool `json:"features"`
}
