package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dauxuanhoanghung/url-shortener/internal/model"
	"github.com/google/uuid"
)

func newTagService(tagRepo *stubTagRepo, urlRepo *stubURLRepo) TagService {
	return NewTagService(tagRepo, urlRepo)
}

// ── SetTags ───────────────────────────────────────────────────────────────────

func TestTagService_SetTags_NormalizesAndPersists(t *testing.T) {
	tagRepo := &stubTagRepo{}
	svc := newTagService(tagRepo, &stubURLRepo{})
	urlID := uuid.New()

	got, err := svc.SetTags(context.Background(), urlID, []string{"  Work ", "", "docs", "WORK", "Docs"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"Work", "docs"} // trimmed, empties dropped, case-insensitive dedupe keeps first casing
	if !reflect.DeepEqual(got, want) {
		t.Errorf("normalized tags: got %v want %v", got, want)
	}
	if !reflect.DeepEqual(tagRepo.tags[urlID], want) {
		t.Errorf("persisted tags: got %v want %v", tagRepo.tags[urlID], want)
	}
}

func TestTagService_SetTags_EmptyList_ClearsTags(t *testing.T) {
	urlID := uuid.New()
	tagRepo := &stubTagRepo{tags: map[uuid.UUID][]string{urlID: {"old"}}}
	svc := newTagService(tagRepo, &stubURLRepo{})

	got, err := svc.SetTags(context.Background(), urlID, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no tags, got %v", got)
	}
	if len(tagRepo.tags[urlID]) != 0 {
		t.Errorf("expected repo tags cleared, got %v", tagRepo.tags[urlID])
	}
}

func TestTagService_SetTags_TooManyTags(t *testing.T) {
	svc := newTagService(&stubTagRepo{}, &stubURLRepo{})

	tags := make([]string, maxTagsPerURL+1)
	for i := range tags {
		tags[i] = "tag" + string(rune('a'+i%26)) + string(rune('a'+i/26))
	}

	_, err := svc.SetTags(context.Background(), uuid.New(), tags)
	if !errors.Is(err, ErrTagLimitExceeded) {
		t.Errorf("expected ErrTagLimitExceeded, got %v", err)
	}
}

func TestTagService_SetTags_TagTooLong(t *testing.T) {
	svc := newTagService(&stubTagRepo{}, &stubURLRepo{})

	_, err := svc.SetTags(context.Background(), uuid.New(), []string{strings.Repeat("x", maxTagLength+1)})
	if !errors.Is(err, ErrInvalidTag) {
		t.Errorf("expected ErrInvalidTag, got %v", err)
	}
}

func TestTagService_SetTags_RepoError_Propagates(t *testing.T) {
	repoErr := errors.New("db down")
	svc := newTagService(&stubTagRepo{setErr: repoErr}, &stubURLRepo{})

	_, err := svc.SetTags(context.Background(), uuid.New(), []string{"a"})
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error to propagate, got %v", err)
	}
}

// ── UpdateTags ────────────────────────────────────────────────────────────────

func TestTagService_UpdateTags_HappyPath(t *testing.T) {
	userID := uuid.New()
	urlID := uuid.New()
	urlRepo := &stubURLRepo{storedURL: &model.ShortURL{ID: urlID, UserID: userID}}
	tagRepo := &stubTagRepo{}
	svc := newTagService(tagRepo, urlRepo)

	got, err := svc.UpdateTags(context.Background(), urlID, userID, []string{"news", "go"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"news", "go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("tags: got %v want %v", got, want)
	}
	if !reflect.DeepEqual(tagRepo.tags[urlID], want) {
		t.Errorf("persisted tags: got %v want %v", tagRepo.tags[urlID], want)
	}
}

func TestTagService_UpdateTags_URLNotFound(t *testing.T) {
	svc := newTagService(&stubTagRepo{}, &stubURLRepo{})

	_, err := svc.UpdateTags(context.Background(), uuid.New(), uuid.New(), []string{"a"})
	if !errors.Is(err, ErrURLNotFound) {
		t.Errorf("expected ErrURLNotFound, got %v", err)
	}
}

func TestTagService_UpdateTags_Forbidden(t *testing.T) {
	urlID := uuid.New()
	ownerID := uuid.New()
	otherID := uuid.New()
	urlRepo := &stubURLRepo{storedURL: &model.ShortURL{ID: urlID, UserID: ownerID}}
	tagRepo := &stubTagRepo{}
	svc := newTagService(tagRepo, urlRepo)

	_, err := svc.UpdateTags(context.Background(), urlID, otherID, []string{"a"})
	if !errors.Is(err, ErrURLForbidden) {
		t.Errorf("expected ErrURLForbidden, got %v", err)
	}
	if tagRepo.setCalls != 0 {
		t.Error("tag repo must not be touched when ownership check fails")
	}
}

// ── TagsForURLs ───────────────────────────────────────────────────────────────

func TestTagService_TagsForURLs_BatchFetch(t *testing.T) {
	id1, id2 := uuid.New(), uuid.New()
	tagRepo := &stubTagRepo{tags: map[uuid.UUID][]string{
		id1: {"a", "b"},
		id2: {"c"},
	}}
	svc := newTagService(tagRepo, &stubURLRepo{})

	got, err := svc.TagsForURLs(context.Background(), []uuid.UUID{id1, id2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got[id1], []string{"a", "b"}) {
		t.Errorf("id1 tags: got %v", got[id1])
	}
	if !reflect.DeepEqual(got[id2], []string{"c"}) {
		t.Errorf("id2 tags: got %v", got[id2])
	}
}

func TestTagService_TagsForURLs_EmptyInput(t *testing.T) {
	svc := newTagService(&stubTagRepo{}, &stubURLRepo{})

	got, err := svc.TagsForURLs(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil empty map")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}
