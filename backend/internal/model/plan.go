package model

import "time"

// Plan mirrors a row in the `plans` table. Entitlements are the source of
// truth for per-tier feature gates; see docs/22-subscription-plans.md.
type Plan struct {
	Code                   string
	Name                   string
	PriceCents             int32
	MaxURLs                int32
	MaxDomains             int32
	MaxTeamMembers         int32
	AnalyticsRetentionDays int32
	APIRateLimitPerMin     *int32          // NULL for Free (no API access)
	Features               map[string]bool // unmarshalled from JSONB
	CreatedAt              time.Time
}

// Feature returns true when the plan has the named feature enabled.
// Unknown features return false, so a typo in caller code fails closed.
func (p *Plan) Feature(name string) bool {
	if p == nil || p.Features == nil {
		return false
	}
	return p.Features[name]
}
