package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/dauxuanhoanghung/url-shortener/internal/dto"
	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/dauxuanhoanghung/url-shortener/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrCannotDisableSelf = errors.New("admin cannot disable their own account")
	ErrCannotDisableLastAdmin = errors.New("cannot disable the last remaining admin")
)

type AdminService interface {
	ListUsers(ctx context.Context, limit, offset int32) (*dto.AdminUserListResponse, error)
	GetUser(ctx context.Context, id uuid.UUID) (*dto.AdminUserResponse, error)
	SetUserDisabled(ctx context.Context, actorID, targetID uuid.UUID, disabled bool) (*dto.AdminUserResponse, error)
	UpdatePlanFeatures(ctx context.Context, actorID uuid.UUID, code string, features map[string]bool) (*model.Plan, error)
	ListAudit(ctx context.Context, limit, offset int32) (*dto.AdminAuditListResponse, error)
}

type adminService struct {
	userRepo     repository.UserRepository
	userPlanRepo repository.UserPlanRepository
	planRepo     repository.PlanRepository
	auditRepo    repository.AdminAuditRepository
}

func NewAdminService(
	userRepo repository.UserRepository,
	userPlanRepo repository.UserPlanRepository,
	planRepo repository.PlanRepository,
	auditRepo repository.AdminAuditRepository,
) AdminService {
	return &adminService{
		userRepo:     userRepo,
		userPlanRepo: userPlanRepo,
		planRepo:     planRepo,
		auditRepo:    auditRepo,
	}
}

func (s *adminService) ListUsers(ctx context.Context, limit, offset int32) (*dto.AdminUserListResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.userRepo.ListForAdmin(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	total, err := s.userRepo.CountForAdmin(ctx)
	if err != nil {
		return nil, err
	}
	out := dto.AdminUserListResponse{
		Users: make([]dto.AdminUserResponse, 0, len(rows)),
		Total: total,
	}
	for _, row := range rows {
		out.Users = append(out.Users, toAdminUserResponse(&row.User, row.PlanCode))
	}
	return &out, nil
}

func (s *adminService) GetUser(ctx context.Context, id uuid.UUID) (*dto.AdminUserResponse, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	planCode := ""
	if up, err := s.userPlanRepo.GetByUserID(ctx, id); err == nil && up != nil {
		planCode = up.PlanCode
	}
	resp := toAdminUserResponse(user, planCode)
	return &resp, nil
}

func (s *adminService) SetUserDisabled(ctx context.Context, actorID, targetID uuid.UUID, disabled bool) (*dto.AdminUserResponse, error) {
	if actorID == targetID {
		return nil, ErrCannotDisableSelf
	}
	target, err := s.userRepo.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}

	beforeJSON, _ := json.Marshal(map[string]any{
		"disabled": target.IsDisabled(),
	})

	var disabledAt *time.Time
	action := "user_enabled"
	if disabled {
		now := time.Now()
		disabledAt = &now
		action = "user_disabled"
	}

	if err := s.userRepo.SetDisabled(ctx, targetID, disabledAt); err != nil {
		return nil, err
	}

	afterJSON, _ := json.Marshal(map[string]any{
		"disabled": disabled,
	})
	if err := s.writeAudit(ctx, actorID, action, "user", targetID.String(), beforeJSON, afterJSON); err != nil {
		return nil, err
	}

	target.DisabledAt = disabledAt
	planCode := ""
	if up, err := s.userPlanRepo.GetByUserID(ctx, targetID); err == nil && up != nil {
		planCode = up.PlanCode
	}
	resp := toAdminUserResponse(target, planCode)
	return &resp, nil
}

func (s *adminService) UpdatePlanFeatures(ctx context.Context, actorID uuid.UUID, code string, features map[string]bool) (*model.Plan, error) {
	before, err := s.planRepo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	beforeJSON, _ := json.Marshal(map[string]any{"features": before.Features})

	updated, err := s.planRepo.UpdateFeatures(ctx, code, features)
	if err != nil {
		return nil, err
	}
	afterJSON, _ := json.Marshal(map[string]any{"features": updated.Features})

	if err := s.writeAudit(ctx, actorID, "plan_features_updated", "plan", code, beforeJSON, afterJSON); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *adminService) ListAudit(ctx context.Context, limit, offset int32) (*dto.AdminAuditListResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.auditRepo.List(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	total, err := s.auditRepo.Count(ctx)
	if err != nil {
		return nil, err
	}
	out := dto.AdminAuditListResponse{
		Entries: make([]dto.AdminAuditEntryResponse, 0, len(rows)),
		Total:   total,
	}
	for _, row := range rows {
		entry := dto.AdminAuditEntryResponse{
			ID:         row.ID.String(),
			Action:     row.Action,
			TargetType: row.TargetType,
			TargetID:   row.TargetID,
			CreatedAt:  row.CreatedAt.Format(time.RFC3339),
		}
		if row.ActorID != nil {
			s := row.ActorID.String()
			entry.ActorID = &s
		}
		if len(row.Before) > 0 {
			_ = json.Unmarshal(row.Before, &entry.Before)
		}
		if len(row.After) > 0 {
			_ = json.Unmarshal(row.After, &entry.After)
		}
		out.Entries = append(out.Entries, entry)
	}
	return &out, nil
}

func (s *adminService) writeAudit(ctx context.Context, actorID uuid.UUID, action, targetType, targetID string, before, after []byte) error {
	return s.auditRepo.Create(ctx, &repository.AdminAuditEntry{
		ID:         uuid.New(),
		ActorID:    &actorID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		Before:     before,
		After:      after,
		CreatedAt:  time.Now(),
	})
}

func toAdminUserResponse(u *model.User, planCode string) dto.AdminUserResponse {
	resp := dto.AdminUserResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		Role:          u.Role,
		PlanCode:      planCode,
		EmailVerified: u.IsEmailVerified(),
		Disabled:      u.IsDisabled(),
		CreatedAt:     u.CreatedAt.Format(time.RFC3339),
	}
	if u.DisabledAt != nil {
		s := u.DisabledAt.Format(time.RFC3339)
		resp.DisabledAt = &s
	}
	return resp
}
