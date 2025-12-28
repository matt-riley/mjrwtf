package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// mockRepository is a mock implementation of url.Repository for testing
type mockRepository struct {
	urls map[string]*url.URL
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		urls: make(map[string]*url.URL),
	}
}

func (m *mockRepository) Create(ctx context.Context, u *url.URL) error {
	if _, exists := m.urls[u.ShortCode]; exists {
		return url.ErrDuplicateShortCode
	}
	m.urls[u.ShortCode] = u
	return nil
}

func (m *mockRepository) FindByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	if u, exists := m.urls[shortCode]; exists {
		return u, nil
	}
	return nil, url.ErrURLNotFound
}

func (m *mockRepository) Delete(ctx context.Context, shortCode string) error {
	delete(m.urls, shortCode)
	return nil
}

func (m *mockRepository) List(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
	return nil, nil
}

func (m *mockRepository) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	return nil, nil
}

func (m *mockRepository) Count(ctx context.Context, createdBy string) (int, error) {
	return 0, nil
}

func TestNewCreateURLUseCase(t *testing.T) {
	repo := newMockRepository()
	gen, err := url.NewGenerator(repo, url.DefaultGeneratorConfig())
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	uc := NewCreateURLUseCase(gen, "https://mjr.wtf")

	if uc == nil {
		t.Error("NewCreateURLUseCase() returned nil")
	}

	if uc.generator != gen {
		t.Error("NewCreateURLUseCase() generator not set correctly")
	}

	if uc.baseURL != "https://mjr.wtf" {
		t.Errorf("NewCreateURLUseCase() baseURL = %v, want https://mjr.wtf", uc.baseURL)
	}
}

func TestCreateURLUseCase_Execute(t *testing.T) {
	tests := []struct {
		name        string
		request     CreateURLRequest
		baseURL     string
		setupMock   func(*mockRepository)
		wantErr     error
		validateRes func(*testing.T, *CreateURLResponse)
	}{
		{
			name: "successful creation",
			request: CreateURLRequest{
				OriginalURL: "https://example.com",
				CreatedBy:   "user1",
			},
			baseURL: "https://mjr.wtf",
			wantErr: nil,
			validateRes: func(t *testing.T, res *CreateURLResponse) {
				if res == nil {
					t.Fatal("Execute() response is nil")
				}
				if res.OriginalURL != "https://example.com" {
					t.Errorf("Execute() OriginalURL = %v, want https://example.com", res.OriginalURL)
				}
				if len(res.ShortCode) != 6 {
					t.Errorf("Execute() ShortCode length = %v, want 6", len(res.ShortCode))
				}
				expectedPrefix := "https://mjr.wtf/"
				if !strings.HasPrefix(res.ShortURL, expectedPrefix) {
					t.Errorf("Execute() ShortURL = %v, want prefix %v", res.ShortURL, expectedPrefix)
				}
				expectedShortURL := "https://mjr.wtf/" + res.ShortCode
				if res.ShortURL != expectedShortURL {
					t.Errorf("Execute() ShortURL = %v, want %v", res.ShortURL, expectedShortURL)
				}
			},
		},
		{
			name: "successful creation with path",
			request: CreateURLRequest{
				OriginalURL: "https://example.com/path/to/page",
				CreatedBy:   "api-key-123",
			},
			baseURL: "https://short.link",
			wantErr: nil,
			validateRes: func(t *testing.T, res *CreateURLResponse) {
				if res == nil {
					t.Fatal("Execute() response is nil")
				}
				if res.OriginalURL != "https://example.com/path/to/page" {
					t.Errorf("Execute() OriginalURL = %v, want https://example.com/path/to/page", res.OriginalURL)
				}
				if !strings.HasPrefix(res.ShortURL, "https://short.link/") {
					t.Errorf("Execute() ShortURL = %v, want prefix https://short.link/", res.ShortURL)
				}
			},
		},
		{
			name: "successful creation with query params",
			request: CreateURLRequest{
				OriginalURL: "https://example.com/page?foo=bar&baz=qux",
				CreatedBy:   "system",
			},
			baseURL: "https://mjr.wtf",
			wantErr: nil,
			validateRes: func(t *testing.T, res *CreateURLResponse) {
				if res == nil {
					t.Fatal("Execute() response is nil")
				}
				if res.OriginalURL != "https://example.com/page?foo=bar&baz=qux" {
					t.Errorf("Execute() OriginalURL = %v, want https://example.com/page?foo=bar&baz=qux", res.OriginalURL)
				}
			},
		},
		{
			name: "invalid URL - empty",
			request: CreateURLRequest{
				OriginalURL: "",
				CreatedBy:   "user1",
			},
			baseURL: "https://mjr.wtf",
			wantErr: url.ErrEmptyOriginalURL,
		},
		{
			name: "invalid URL - no scheme",
			request: CreateURLRequest{
				OriginalURL: "example.com",
				CreatedBy:   "user1",
			},
			baseURL: "https://mjr.wtf",
			wantErr: url.ErrMissingURLScheme,
		},
		{
			name: "invalid URL - bad scheme",
			request: CreateURLRequest{
				OriginalURL: "ftp://example.com",
				CreatedBy:   "user1",
			},
			baseURL: "https://mjr.wtf",
			wantErr: url.ErrInvalidURLScheme,
		},
		{
			name: "invalid URL - no host",
			request: CreateURLRequest{
				OriginalURL: "https://",
				CreatedBy:   "user1",
			},
			baseURL: "https://mjr.wtf",
			wantErr: url.ErrMissingURLHost,
		},
		{
			name: "empty created by",
			request: CreateURLRequest{
				OriginalURL: "https://example.com",
				CreatedBy:   "",
			},
			baseURL: "https://mjr.wtf",
			wantErr: url.ErrInvalidCreatedBy,
		},
		{
			name: "whitespace only created by",
			request: CreateURLRequest{
				OriginalURL: "https://example.com",
				CreatedBy:   "   ",
			},
			baseURL: "https://mjr.wtf",
			wantErr: url.ErrInvalidCreatedBy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			gen, err := url.NewGenerator(repo, url.DefaultGeneratorConfig())
			if err != nil {
				t.Fatalf("NewGenerator() error = %v", err)
			}

			uc := NewCreateURLUseCase(gen, tt.baseURL)
			res, err := uc.Execute(context.Background(), tt.request)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("Execute() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("Execute() unexpected error = %v", err)
				return
			}

			if tt.validateRes != nil {
				tt.validateRes(t, res)
			}
		})
	}
}

func TestCreateURLUseCase_Execute_CollisionHandling(t *testing.T) {
	t.Run("handles collision and retries", func(t *testing.T) {
		repo := newMockRepository()
		gen, err := url.NewGenerator(repo, url.GeneratorConfig{
			CodeLength: 3, // Short code to increase collision probability
			MaxRetries: 5,
		})
		if err != nil {
			t.Fatalf("NewGenerator() error = %v", err)
		}

		uc := NewCreateURLUseCase(gen, "https://mjr.wtf")

		// Create multiple URLs - should all succeed despite potential collisions
		created := make(map[string]bool)
		for i := 0; i < 10; i++ {
			req := CreateURLRequest{
				OriginalURL: "https://example.com",
				CreatedBy:   "user1",
			}

			res, err := uc.Execute(context.Background(), req)
			if err != nil {
				t.Errorf("Execute() unexpected error = %v", err)
				continue
			}

			if created[res.ShortCode] {
				t.Errorf("Execute() generated duplicate short code: %s", res.ShortCode)
			}
			created[res.ShortCode] = true
		}

		if len(created) != 10 {
			t.Errorf("Execute() created %d unique codes, want 10", len(created))
		}
	})
}

func TestCreateURLUseCase_Execute_MaxRetriesExceeded(t *testing.T) {
	repo := newMockRepository()

	// Create a mock that always reports collisions
	attempts := 0
	mockRepo := &mockAlwaysCollisionRepo{
		wrapped:  repo,
		attempts: &attempts,
	}

	gen, err := url.NewGenerator(mockRepo, url.GeneratorConfig{
		CodeLength: 6,
		MaxRetries: 2,
	})
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	uc := NewCreateURLUseCase(gen, "https://mjr.wtf")

	req := CreateURLRequest{
		OriginalURL: "https://example.com",
		CreatedBy:   "user1",
	}

	_, err = uc.Execute(context.Background(), req)
	if err == nil {
		t.Error("Execute() expected error, got nil")
		return
	}

	if !errors.Is(err, url.ErrMaxRetriesExceeded) {
		t.Errorf("Execute() error = %v, want %v", err, url.ErrMaxRetriesExceeded)
	}
}

func TestCreateURLUseCase_Execute_BaseURLFormatting(t *testing.T) {
	tests := []struct {
		name       string
		baseURL    string
		wantPrefix string
	}{
		{
			name:       "base URL without trailing slash",
			baseURL:    "https://mjr.wtf",
			wantPrefix: "https://mjr.wtf/",
		},
		{
			name:       "base URL with trailing slash",
			baseURL:    "https://mjr.wtf/",
			wantPrefix: "https://mjr.wtf/",
		},
		{
			name:       "base URL with subdomain",
			baseURL:    "https://s.example.com",
			wantPrefix: "https://s.example.com/",
		},
		{
			name:       "base URL with port",
			baseURL:    "http://localhost:8080",
			wantPrefix: "http://localhost:8080/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			gen, err := url.NewGenerator(repo, url.DefaultGeneratorConfig())
			if err != nil {
				t.Fatalf("NewGenerator() error = %v", err)
			}

			uc := NewCreateURLUseCase(gen, tt.baseURL)

			req := CreateURLRequest{
				OriginalURL: "https://example.com",
				CreatedBy:   "user1",
			}

			res, err := uc.Execute(context.Background(), req)
			if err != nil {
				t.Fatalf("Execute() unexpected error = %v", err)
			}

			if !strings.HasPrefix(res.ShortURL, tt.wantPrefix) {
				t.Errorf("Execute() ShortURL = %v, want prefix %v", res.ShortURL, tt.wantPrefix)
			}
		})
	}
}

// mockAlwaysCollisionRepo simulates a repository where all codes collide
type mockAlwaysCollisionRepo struct {
	wrapped  *mockRepository
	attempts *int
}

func (m *mockAlwaysCollisionRepo) Create(ctx context.Context, u *url.URL) error {
	return m.wrapped.Create(ctx, u)
}

func (m *mockAlwaysCollisionRepo) FindByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	*m.attempts++
	// Always return a URL to simulate collision
	return &url.URL{
		ID:          1,
		ShortCode:   shortCode,
		OriginalURL: "https://example.com",
		CreatedBy:   "system",
	}, nil
}

func (m *mockAlwaysCollisionRepo) Delete(ctx context.Context, shortCode string) error {
	return m.wrapped.Delete(ctx, shortCode)
}

func (m *mockAlwaysCollisionRepo) List(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
	return m.wrapped.List(ctx, createdBy, limit, offset)
}

func (m *mockAlwaysCollisionRepo) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	return m.wrapped.ListByCreatedByAndTimeRange(ctx, createdBy, startTime, endTime)
}

func (m *mockAlwaysCollisionRepo) Count(ctx context.Context, createdBy string) (int, error) {
	return m.wrapped.Count(ctx, createdBy)
}
