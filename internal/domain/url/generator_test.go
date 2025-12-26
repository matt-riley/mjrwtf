package url

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// MockRepository is a mock implementation of Repository for testing
type MockRepository struct {
	urls      map[string]*URL
	createErr error
	findErr   error
}

func NewMockRepository() *MockRepository {
	return &MockRepository{
		urls: make(map[string]*URL),
	}
}

func (m *MockRepository) Create(ctx context.Context, url *URL) error {
	if m.createErr != nil {
		return m.createErr
	}
	if _, exists := m.urls[url.ShortCode]; exists {
		return ErrDuplicateShortCode
	}
	m.urls[url.ShortCode] = url
	return nil
}

func (m *MockRepository) FindByShortCode(ctx context.Context, shortCode string) (*URL, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if url, exists := m.urls[shortCode]; exists {
		return url, nil
	}
	return nil, ErrURLNotFound
}

func (m *MockRepository) Delete(ctx context.Context, shortCode string) error {
	delete(m.urls, shortCode)
	return nil
}

func (m *MockRepository) List(ctx context.Context, createdBy string, limit, offset int) ([]*URL, error) {
	return nil, errors.New("not implemented")
}

func (m *MockRepository) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*URL, error) {
	return nil, errors.New("not implemented")
}

func (m *MockRepository) Count(ctx context.Context, createdBy string) (int, error) {
	return 0, nil
}

func TestNewGenerator(t *testing.T) {
	repo := NewMockRepository()

	tests := []struct {
		name    string
		config  GeneratorConfig
		wantErr error
	}{
		{
			name: "valid default config",
			config: GeneratorConfig{
				CodeLength: 6,
				MaxRetries: 3,
			},
			wantErr: nil,
		},
		{
			name: "valid custom config",
			config: GeneratorConfig{
				CodeLength: 8,
				MaxRetries: 5,
			},
			wantErr: nil,
		},
		{
			name: "minimum code length",
			config: GeneratorConfig{
				CodeLength: 3,
				MaxRetries: 3,
			},
			wantErr: nil,
		},
		{
			name: "maximum code length",
			config: GeneratorConfig{
				CodeLength: 20,
				MaxRetries: 3,
			},
			wantErr: nil,
		},
		{
			name: "code length too short",
			config: GeneratorConfig{
				CodeLength: 2,
				MaxRetries: 3,
			},
			wantErr: ErrInvalidCodeLength,
		},
		{
			name: "code length too long",
			config: GeneratorConfig{
				CodeLength: 21,
				MaxRetries: 3,
			},
			wantErr: ErrInvalidCodeLength,
		},
		{
			name: "zero max retries gets set to 1",
			config: GeneratorConfig{
				CodeLength: 6,
				MaxRetries: 0,
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator(repo, tt.config)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("NewGenerator() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("NewGenerator() unexpected error = %v", err)
				return
			}

			if gen.codeLength != tt.config.CodeLength {
				t.Errorf("NewGenerator() codeLength = %v, want %v", gen.codeLength, tt.config.CodeLength)
			}

			expectedRetries := tt.config.MaxRetries
			if expectedRetries < 1 {
				expectedRetries = 1
			}
			if gen.maxRetries != expectedRetries {
				t.Errorf("NewGenerator() maxRetries = %v, want %v", gen.maxRetries, expectedRetries)
			}
		})
	}
}

func TestDefaultGeneratorConfig(t *testing.T) {
	config := DefaultGeneratorConfig()

	if config.CodeLength != 6 {
		t.Errorf("DefaultGeneratorConfig() CodeLength = %v, want 6", config.CodeLength)
	}

	if config.MaxRetries != 3 {
		t.Errorf("DefaultGeneratorConfig() MaxRetries = %v, want 3", config.MaxRetries)
	}
}

func TestGenerator_GenerateShortCode(t *testing.T) {
	repo := NewMockRepository()

	tests := []struct {
		name       string
		codeLength int
	}{
		{
			name:       "length 3",
			codeLength: 3,
		},
		{
			name:       "length 6",
			codeLength: 6,
		},
		{
			name:       "length 8",
			codeLength: 8,
		},
		{
			name:       "length 10",
			codeLength: 10,
		},
		{
			name:       "length 20",
			codeLength: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gen, err := NewGenerator(repo, GeneratorConfig{
				CodeLength: tt.codeLength,
				MaxRetries: 3,
			})
			if err != nil {
				t.Fatalf("NewGenerator() error = %v", err)
			}

			code, err := gen.GenerateShortCode()
			if err != nil {
				t.Errorf("GenerateShortCode() error = %v", err)
				return
			}

			if len(code) != tt.codeLength {
				t.Errorf("GenerateShortCode() length = %v, want %v", len(code), tt.codeLength)
			}

			// Verify all characters are from base62 charset
			for _, char := range code {
				if !strings.ContainsRune(base62Chars, char) {
					t.Errorf("GenerateShortCode() contains invalid character: %c", char)
				}
			}
		})
	}
}

func TestGenerator_GenerateShortCode_IsRandom(t *testing.T) {
	repo := NewMockRepository()
	gen, err := NewGenerator(repo, DefaultGeneratorConfig())
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	// Generate multiple codes and verify they're different
	codes := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		code, err := gen.GenerateShortCode()
		if err != nil {
			t.Errorf("GenerateShortCode() error = %v", err)
			return
		}
		codes[code] = true
	}

	// We expect high uniqueness (at least 95% unique for 100 iterations with 6-char base62)
	uniquePercent := (float64(len(codes)) / float64(iterations)) * 100
	if uniquePercent < 95 {
		t.Errorf("GenerateShortCode() uniqueness = %.2f%%, want >= 95%%", uniquePercent)
	}
}

func TestGenerator_GenerateUniqueShortCode(t *testing.T) {
	t.Run("no collision", func(t *testing.T) {
		repo := NewMockRepository()
		gen, err := NewGenerator(repo, DefaultGeneratorConfig())
		if err != nil {
			t.Fatalf("NewGenerator() error = %v", err)
		}

		code, err := gen.GenerateUniqueShortCode(context.Background())
		if err != nil {
			t.Errorf("GenerateUniqueShortCode() error = %v", err)
			return
		}

		if len(code) != 6 {
			t.Errorf("GenerateUniqueShortCode() length = %v, want 6", len(code))
		}
	})

	t.Run("collision resolved on retry", func(t *testing.T) {
		repo := NewMockRepository()
		gen, err := NewGenerator(repo, GeneratorConfig{
			CodeLength: 3, // Use shorter code to increase collision probability
			MaxRetries: 5,
		})
		if err != nil {
			t.Fatalf("NewGenerator() error = %v", err)
		}

		// Pre-populate repository with some codes to simulate collisions
		existingCodes := []string{"abc", "def", "ghi", "jkl", "mno"}
		for _, code := range existingCodes {
			url, err := NewURL(code, "https://example.com", "system")
			if err != nil {
				t.Fatalf("NewURL() error = %v", err)
			}
			repo.Create(context.Background(), url)
		}

		// Try to generate unique code - should succeed after some retries
		code, err := gen.GenerateUniqueShortCode(context.Background())
		if err != nil {
			t.Errorf("GenerateUniqueShortCode() error = %v", err)
			return
		}

		// Verify generated code is not in existing codes
		for _, existingCode := range existingCodes {
			if code == existingCode {
				t.Errorf("GenerateUniqueShortCode() generated existing code: %s", code)
			}
		}
	})

	t.Run("max retries exceeded", func(t *testing.T) {
		repo := NewMockRepository()

		// Create a generator with low retry count
		gen, err := NewGenerator(repo, GeneratorConfig{
			CodeLength: 6,
			MaxRetries: 2,
		})
		if err != nil {
			t.Fatalf("NewGenerator() error = %v", err)
		}

		// Mock repository to always return a collision
		repo.findErr = nil // Reset any previous errors

		// Override the GenerateShortCode to always return the same code
		// We'll test this by creating a mock that simulates constant collisions
		attempts := 0
		originalGen := gen
		testGen := &Generator{
			codeLength: originalGen.codeLength,
			maxRetries: originalGen.maxRetries,
			repository: &mockAlwaysCollisionRepo{wrapped: repo, attempts: &attempts},
		}

		_, err = testGen.GenerateUniqueShortCode(context.Background())
		if err != ErrMaxRetriesExceeded {
			t.Errorf("GenerateUniqueShortCode() error = %v, want %v", err, ErrMaxRetriesExceeded)
		}

		if attempts != 2 {
			t.Errorf("GenerateUniqueShortCode() attempts = %v, want 2", attempts)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := NewMockRepository()
		repo.findErr = errors.New("database error")

		gen, err := NewGenerator(repo, DefaultGeneratorConfig())
		if err != nil {
			t.Fatalf("NewGenerator() error = %v", err)
		}

		_, err = gen.GenerateUniqueShortCode(context.Background())
		if err == nil {
			t.Error("GenerateUniqueShortCode() expected error, got nil")
		}
		if err == ErrMaxRetriesExceeded {
			t.Error("GenerateUniqueShortCode() should not return ErrMaxRetriesExceeded for repository errors")
		}
	})
}

// mockAlwaysCollisionRepo simulates a repository where all codes collide
type mockAlwaysCollisionRepo struct {
	wrapped  *MockRepository
	attempts *int
}

func (m *mockAlwaysCollisionRepo) Create(ctx context.Context, url *URL) error {
	return m.wrapped.Create(ctx, url)
}

func (m *mockAlwaysCollisionRepo) FindByShortCode(ctx context.Context, shortCode string) (*URL, error) {
	*m.attempts++
	// Always return a URL to simulate collision
	return &URL{
		ID:          1,
		ShortCode:   shortCode,
		OriginalURL: "https://example.com",
		CreatedBy:   "system",
		CreatedAt:   time.Now(),
	}, nil
}

func (m *mockAlwaysCollisionRepo) Delete(ctx context.Context, shortCode string) error {
	return m.wrapped.Delete(ctx, shortCode)
}

func (m *mockAlwaysCollisionRepo) List(ctx context.Context, createdBy string, limit, offset int) ([]*URL, error) {
	return m.wrapped.List(ctx, createdBy, limit, offset)
}

func (m *mockAlwaysCollisionRepo) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*URL, error) {
	return m.wrapped.ListByCreatedByAndTimeRange(ctx, createdBy, startTime, endTime)
}

func (m *mockAlwaysCollisionRepo) Count(ctx context.Context, createdBy string) (int, error) {
	return m.wrapped.Count(ctx, createdBy)
}

func TestGenerator_ShortenURL(t *testing.T) {
	tests := []struct {
		name        string
		originalURL string
		createdBy   string
		wantErr     error
	}{
		{
			name:        "valid URL",
			originalURL: "https://example.com",
			createdBy:   "user1",
			wantErr:     nil,
		},
		{
			name:        "valid URL with path",
			originalURL: "https://example.com/path/to/page",
			createdBy:   "api-key-123",
			wantErr:     nil,
		},
		{
			name:        "valid URL with query params",
			originalURL: "https://example.com/page?foo=bar&baz=qux",
			createdBy:   "system",
			wantErr:     nil,
		},
		{
			name:        "invalid URL - no scheme",
			originalURL: "example.com",
			createdBy:   "user1",
			wantErr:     ErrMissingURLScheme,
		},
		{
			name:        "invalid URL - empty",
			originalURL: "",
			createdBy:   "user1",
			wantErr:     ErrEmptyOriginalURL,
		},
		{
			name:        "invalid URL - bad scheme",
			originalURL: "ftp://example.com",
			createdBy:   "user1",
			wantErr:     ErrInvalidURLScheme,
		},
		{
			name:        "empty created by",
			originalURL: "https://example.com",
			createdBy:   "",
			wantErr:     ErrInvalidCreatedBy,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := NewMockRepository()
			gen, err := NewGenerator(repo, DefaultGeneratorConfig())
			if err != nil {
				t.Fatalf("NewGenerator() error = %v", err)
			}

			url, err := gen.ShortenURL(context.Background(), tt.originalURL, tt.createdBy)

			if tt.wantErr != nil {
				if err == nil {
					t.Errorf("ShortenURL() error = nil, wantErr %v", tt.wantErr)
					return
				}
				if err != tt.wantErr && !strings.Contains(err.Error(), tt.wantErr.Error()) {
					t.Errorf("ShortenURL() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ShortenURL() unexpected error = %v", err)
				return
			}

			if url.OriginalURL != tt.originalURL {
				t.Errorf("ShortenURL() OriginalURL = %v, want %v", url.OriginalURL, tt.originalURL)
			}

			if url.CreatedBy != tt.createdBy {
				t.Errorf("ShortenURL() CreatedBy = %v, want %v", url.CreatedBy, tt.createdBy)
			}

			if len(url.ShortCode) != 6 {
				t.Errorf("ShortenURL() ShortCode length = %v, want 6", len(url.ShortCode))
			}

			// Verify short code is valid base62
			for _, char := range url.ShortCode {
				if !strings.ContainsRune(base62Chars, char) {
					t.Errorf("ShortenURL() ShortCode contains invalid character: %c", char)
				}
			}

			// Verify URL is in repository
			found, err := repo.FindByShortCode(context.Background(), url.ShortCode)
			if err != nil {
				t.Errorf("ShortenURL() URL not found in repository: %v", err)
				return
			}

			if found.OriginalURL != tt.originalURL {
				t.Errorf("ShortenURL() stored OriginalURL = %v, want %v", found.OriginalURL, tt.originalURL)
			}
		})
	}
}

func TestGenerator_ShortenURL_DuplicateDetection(t *testing.T) {
	repo := NewMockRepository()
	gen, err := NewGenerator(repo, DefaultGeneratorConfig())
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	// Create first URL
	url1, err := gen.ShortenURL(context.Background(), "https://example.com", "user1")
	if err != nil {
		t.Fatalf("ShortenURL() error = %v", err)
	}

	// Create second URL - should get different short code
	url2, err := gen.ShortenURL(context.Background(), "https://example.com", "user1")
	if err != nil {
		t.Fatalf("ShortenURL() error = %v", err)
	}

	if url1.ShortCode == url2.ShortCode {
		t.Error("ShortenURL() generated duplicate short codes")
	}
}

func TestGenerator_ShortenURL_MaxRetriesExceeded(t *testing.T) {
	repo := NewMockRepository()
	gen, err := NewGenerator(repo, GeneratorConfig{
		CodeLength: 6,
		MaxRetries: 2,
	})
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	// Mock repository to always return collision
	attempts := 0
	gen.repository = &mockAlwaysCollisionRepo{wrapped: repo, attempts: &attempts}

	_, err = gen.ShortenURL(context.Background(), "https://example.com", "user1")
	if err != ErrMaxRetriesExceeded {
		t.Errorf("ShortenURL() error = %v, want %v", err, ErrMaxRetriesExceeded)
	}
}

func TestGenerator_ShortenURL_CreateError(t *testing.T) {
	repo := NewMockRepository()
	repo.createErr = errors.New("database error")

	gen, err := NewGenerator(repo, DefaultGeneratorConfig())
	if err != nil {
		t.Fatalf("NewGenerator() error = %v", err)
	}

	_, err = gen.ShortenURL(context.Background(), "https://example.com", "user1")
	if err == nil {
		t.Error("ShortenURL() expected error, got nil")
	}
	if !strings.Contains(err.Error(), "database error") {
		t.Errorf("ShortenURL() error = %v, want database error", err)
	}
}

// Benchmark tests
func BenchmarkGenerator_GenerateShortCode(b *testing.B) {
	repo := NewMockRepository()
	gen, err := NewGenerator(repo, DefaultGeneratorConfig())
	if err != nil {
		b.Fatalf("NewGenerator() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.GenerateShortCode()
		if err != nil {
			b.Fatalf("GenerateShortCode() error = %v", err)
		}
	}
}

func BenchmarkGenerator_GenerateUniqueShortCode(b *testing.B) {
	repo := NewMockRepository()
	gen, err := NewGenerator(repo, DefaultGeneratorConfig())
	if err != nil {
		b.Fatalf("NewGenerator() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		code, err := gen.GenerateUniqueShortCode(context.Background())
		if err != nil {
			b.Fatalf("GenerateUniqueShortCode() error = %v", err)
		}
		// Add to repository to simulate real usage
		url, err := NewURL(code, fmt.Sprintf("https://example.com/%d", i), "benchmark")
		if err != nil {
			b.Fatalf("NewURL() error = %v", err)
		}
		repo.Create(context.Background(), url)
	}
}

func BenchmarkGenerator_ShortenURL(b *testing.B) {
	repo := NewMockRepository()
	gen, err := NewGenerator(repo, DefaultGeneratorConfig())
	if err != nil {
		b.Fatalf("NewGenerator() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := gen.ShortenURL(context.Background(), fmt.Sprintf("https://example.com/%d", i), "benchmark")
		if err != nil {
			b.Fatalf("ShortenURL() error = %v", err)
		}
	}
}
