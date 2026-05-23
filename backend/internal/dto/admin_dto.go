package dto

type AdminUserResponse struct {
	ID            string  `json:"id"`
	Email         string  `json:"email"`
	Role          string  `json:"role"`
	PlanCode      string  `json:"plan_code"`
	EmailVerified bool    `json:"email_verified"`
	Disabled      bool    `json:"disabled"`
	DisabledAt    *string `json:"disabled_at,omitempty"`
	CreatedAt     string  `json:"created_at"`
}

type AdminUserListResponse struct {
	Users []AdminUserResponse `json:"users"`
	Total int64               `json:"total"`
}

type AdminAuditEntryResponse struct {
	ID         string          `json:"id"`
	ActorID    *string         `json:"actor_id,omitempty"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type,omitempty"`
	TargetID   string          `json:"target_id,omitempty"`
	Before     map[string]any  `json:"before,omitempty"`
	After      map[string]any  `json:"after,omitempty"`
	CreatedAt  string          `json:"created_at"`
}

type AdminAuditListResponse struct {
	Entries []AdminAuditEntryResponse `json:"entries"`
	Total   int64                     `json:"total"`
}

type UpdatePlanFeaturesRequest struct {
	Features map[string]bool `json:"features" binding:"required"`
}
