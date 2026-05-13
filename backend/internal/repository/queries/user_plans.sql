-- name: CreateUserPlan :one
INSERT INTO user_plans (user_id, plan_code, started_at, updated_at)
VALUES ($1, $2, $3, $4)
RETURNING id, user_id, plan_code, started_at, updated_at;

-- name: GetUserPlan :one
SELECT up.id, up.user_id, up.plan_code, up.started_at, up.updated_at
FROM user_plans up
WHERE up.user_id = $1;

-- name: UpdateUserPlan :one
UPDATE user_plans
SET plan_code = $2, updated_at = NOW()
WHERE user_id = $1
RETURNING id, user_id, plan_code, started_at, updated_at;
