package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// mockURLRepository is a test double for URL repository
type mockListURLRepository struct {
	listFunc func(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error)
}

func (m *mockListURLRepository) List(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, createdBy, limit, offset)
	}
	return nil, nil
}

func (m *mockListURLRepository) Create(ctx context.Context, u *url.URL) error {
	return nil
}

func (m *mockListURLRepository) FindByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	return nil, nil
}

func (m *mockListURLRepository) Delete(ctx context.Context, shortCode string) error {
	return nil
}

func (m *mockListURLRepository) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	return nil, nil
}

func TestListURLsUseCase_Execute_Success(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	mockURLs := []*url.URL{
		{
			ID:          1,
			ShortCode:   "abc123",
			OriginalURL: "https://example.com",
			CreatedAt:   now,
			CreatedBy:   "user1",
		},
		{
			ID:          2,
			ShortCode:   "def456",
			OriginalURL: "https://example.org",
			CreatedAt:   now.Add(-1 * time.Hour),
			CreatedBy:   "user1",
		},
	}

	mockRepo := &mockListURLRepository{
		listFunc: func(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
			if createdBy != "user1" {
				return nil, nil
			}
			return mockURLs, nil
		},
	}

	useCase := NewListURLsUseCase(mockRepo)

	req := ListURLsRequest{
		CreatedBy: "user1",
		Limit:     20,
		Offset:    0,
	}

	resp, err := useCase.Execute(ctx, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if len(resp.URLs) != 2 {
		t.Errorf("expected 2 URLs, got %d", len(resp.URLs))
	}

	if resp.URLs[0].ShortCode != "abc123" {
		t.Errorf("expected first URL short code to be 'abc123', got %s", resp.URLs[0].ShortCode)
	}

	if resp.Total != 2 {
		t.Errorf("expected total to be 2, got %d", resp.Total)
	}

	if resp.Limit != 20 {
		t.Errorf("expected limit to be 20, got %d", resp.Limit)
	}

	if resp.Offset != 0 {
		t.Errorf("expected offset to be 0, got %d", resp.Offset)
	}
}

func TestListURLsUseCase_Execute_EmptyCreatedBy(t *testing.T) {
	ctx := context.Background()

	mockRepo := &mockListURLRepository{}
	useCase := NewListURLsUseCase(mockRepo)

	req := ListURLsRequest{
		CreatedBy: "",
		Limit:     20,
		Offset:    0,
	}

	resp, err := useCase.Execute(ctx, req)

	if !errors.Is(err, url.ErrInvalidCreatedBy) {
		t.Errorf("expected ErrInvalidCreatedBy, got %v", err)
	}

	if resp != nil {
		t.Error("expected nil response on error")
	}
}

func TestListURLsUseCase_Execute_DefaultPagination(t *testing.T) {
	ctx := context.Background()

	var capturedLimit, capturedOffset int

	mockRepo := &mockListURLRepository{
		listFunc: func(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
			capturedLimit = limit
			capturedOffset = offset
			return []*url.URL{}, nil
		},
	}

	useCase := NewListURLsUseCase(mockRepo)

	req := ListURLsRequest{
		CreatedBy: "user1",
		Limit:     0,  // Should default to 20
		Offset:    -1, // Should default to 0
	}

	_, err := useCase.Execute(ctx, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if capturedLimit != 20 {
		t.Errorf("expected default limit of 20, got %d", capturedLimit)
	}

	if capturedOffset != 0 {
		t.Errorf("expected default offset of 0, got %d", capturedOffset)
	}
}

func TestListURLsUseCase_Execute_MaxLimit(t *testing.T) {
	ctx := context.Background()

	var capturedLimit int

	mockRepo := &mockListURLRepository{
		listFunc: func(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
			capturedLimit = limit
			return []*url.URL{}, nil
		},
	}

	useCase := NewListURLsUseCase(mockRepo)

	req := ListURLsRequest{
		CreatedBy: "user1",
		Limit:     200, // Should be capped at 100
		Offset:    0,
	}

	_, err := useCase.Execute(ctx, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if capturedLimit != 100 {
		t.Errorf("expected max limit of 100, got %d", capturedLimit)
	}
}

func TestListURLsUseCase_Execute_RepositoryError(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New("database error")

	mockRepo := &mockListURLRepository{
		listFunc: func(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
			return nil, expectedErr
		},
	}

	useCase := NewListURLsUseCase(mockRepo)

	req := ListURLsRequest{
		CreatedBy: "user1",
		Limit:     20,
		Offset:    0,
	}

	resp, err := useCase.Execute(ctx, req)

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if resp != nil {
		t.Error("expected nil response on error")
	}
}

func TestListURLsUseCase_Execute_EmptyList(t *testing.T) {
	ctx := context.Background()

	mockRepo := &mockListURLRepository{
		listFunc: func(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
			return []*url.URL{}, nil
		},
	}

	useCase := NewListURLsUseCase(mockRepo)

	req := ListURLsRequest{
		CreatedBy: "user1",
		Limit:     20,
		Offset:    0,
	}

	resp, err := useCase.Execute(ctx, req)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if len(resp.URLs) != 0 {
		t.Errorf("expected 0 URLs, got %d", len(resp.URLs))
	}

	if resp.Total != 0 {
		t.Errorf("expected total to be 0, got %d", resp.Total)
	}
}
