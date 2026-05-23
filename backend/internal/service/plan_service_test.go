package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
)

func TestPlanService_List_ReturnsDTO(t *testing.T) {
	svc := NewPlanService(&stubPlanRepo{plan: &model.Plan{
		Code:       "pro",
		Name:       "Pro",
		PriceCents: 900,
		MaxURLs:    100,
		Features:   map[string]bool{"webhooks": true},
	}})
	out, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(out))
	}
	if out[0].Code != "pro" || out[0].PriceCents != 900 {
		t.Errorf("unexpected plan: %+v", out[0])
	}
	if !out[0].Features["webhooks"] {
		t.Error("expected webhooks feature flag to be passed through")
	}
}

func TestPlanService_List_EmptyRepo(t *testing.T) {
	svc := NewPlanService(&stubPlanRepo{})
	out, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty list, got %d entries", len(out))
	}
}

func TestPlanService_List_PropagatesError(t *testing.T) {
	wantErr := errors.New("db blew up")
	svc := NewPlanService(&errPlanRepo{err: wantErr})
	if _, err := svc.List(context.Background()); !errors.Is(err, wantErr) {
		t.Errorf("expected propagated error, got %v", err)
	}
}

// errPlanRepo satisfies repository.PlanRepository and fails on every call.
type errPlanRepo struct{ err error }

func (r *errPlanRepo) GetByCode(_ context.Context, _ string) (*model.Plan, error) {
	return nil, r.err
}

func (r *errPlanRepo) List(_ context.Context) ([]model.Plan, error) { return nil, r.err }

func (r *errPlanRepo) UpdateFeatures(_ context.Context, _ string, _ map[string]bool) (*model.Plan, error) {
	return nil, r.err
}
