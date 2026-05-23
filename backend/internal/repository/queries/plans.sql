-- name: GetPlanByCode :one
SELECT code, name, price_cents, max_urls, max_domains, max_team_members,
       analytics_retention_days, api_rate_limit_per_min, features, created_at
FROM plans
WHERE code = $1;

-- name: ListPlans :many
SELECT code, name, price_cents, max_urls, max_domains, max_team_members,
       analytics_retention_days, api_rate_limit_per_min, features, created_at
FROM plans
ORDER BY price_cents ASC;

-- name: UpdatePlanFeatures :one
UPDATE plans SET features = $2
WHERE code = $1
RETURNING code, name, price_cents, max_urls, max_domains, max_team_members,
          analytics_retention_days, api_rate_limit_per_min, features, created_at;
