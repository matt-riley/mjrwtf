package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

func TestNewDeleteURLUseCase(t *testing.T) {
	repo := newMockRepository()
	uc := NewDeleteURLUseCase(repo)

	if uc == nil {
		t.Error("NewDeleteURLUseCase() returned nil")
	}

	if uc.urlRepo != repo {
		t.Error("NewDeleteURLUseCase() urlRepo not set correctly")
	}
}

func TestDeleteURLUseCase_Execute(t *testing.T) {
	tests := []struct {
		name      string
		request   DeleteURLRequest
		setupMock func(*mockRepository)
		wantErr   error
		wantRes   *DeleteURLResponse
	}{
		{
			name: "successful deletion",
			request: DeleteURLRequest{
				ShortCode:   "test123",
				RequestedBy: "user1",
			},
			setupMock: func(repo *mockRepository) {
				repo.urls["test123"] = &url.URL{
					ID:          1,
					ShortCode:   "test123",
					OriginalURL: "https://example.com",
					CreatedAt:   time.Now(),
					CreatedBy:   "user1",
				}
			},
			wantErr: nil,
			wantRes: &DeleteURLResponse{Success: true},
		},
		{
			name: "unauthorized deletion - different user",
			request: DeleteURLRequest{
				ShortCode:   "test456",
				RequestedBy: "user2",
			},
			setupMock: func(repo *mockRepository) {
				repo.urls["test456"] = &url.URL{
					ID:          2,
					ShortCode:   "test456",
					OriginalURL: "https://example.com",
					CreatedAt:   time.Now(),
					CreatedBy:   "user1", // Created by different user
				}
			},
			wantErr: url.ErrUnauthorizedDeletion,
			wantRes: nil,
		},
		{
			name: "URL not found",
			request: DeleteURLRequest{
				ShortCode:   "nonexistent",
				RequestedBy: "user1",
			},
			setupMock: func(repo *mockRepository) {
				// Don't add any URL
			},
			wantErr: url.ErrURLNotFound,
			wantRes: nil,
		},
		{
			name: "empty short code",
			request: DeleteURLRequest{
				ShortCode:   "",
				RequestedBy: "user1",
			},
			setupMock: func(repo *mockRepository) {},
			wantErr:   url.ErrEmptyShortCode,
			wantRes:   nil,
		},
		{
			name: "invalid short code - too short",
			request: DeleteURLRequest{
				ShortCode:   "ab",
				RequestedBy: "user1",
			},
			setupMock: func(repo *mockRepository) {},
			wantErr:   url.ErrInvalidShortCode,
			wantRes:   nil,
		},
		{
			name: "invalid short code - invalid characters",
			request: DeleteURLRequest{
				ShortCode:   "test@123",
				RequestedBy: "user1",
			},
			setupMock: func(repo *mockRepository) {},
			wantErr:   url.ErrInvalidShortCode,
			wantRes:   nil,
		},
		{
			name: "empty requested by",
			request: DeleteURLRequest{
				ShortCode:   "test789",
				RequestedBy: "",
			},
			setupMock: func(repo *mockRepository) {},
			wantErr:   url.ErrInvalidCreatedBy,
			wantRes:   nil,
		},
		{
			name: "successful deletion with different creator format",
			request: DeleteURLRequest{
				ShortCode:   "api123",
				RequestedBy: "api-key-xyz",
			},
			setupMock: func(repo *mockRepository) {
				repo.urls["api123"] = &url.URL{
					ID:          3,
					ShortCode:   "api123",
					OriginalURL: "https://api.example.com",
					CreatedAt:   time.Now(),
					CreatedBy:   "api-key-xyz",
				}
			},
			wantErr: nil,
			wantRes: &DeleteURLResponse{Success: true},
		},
		{
			name: "case sensitive creator match",
			request: DeleteURLRequest{
				ShortCode:   "case123",
				RequestedBy: "User1",
			},
			setupMock: func(repo *mockRepository) {
				repo.urls["case123"] = &url.URL{
					ID:          4,
					ShortCode:   "case123",
					OriginalURL: "https://example.com",
					CreatedAt:   time.Now(),
					CreatedBy:   "user1", // Different case
				}
			},
			wantErr: url.ErrUnauthorizedDeletion,
			wantRes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepository()
			if tt.setupMock != nil {
				tt.setupMock(repo)
			}

			uc := NewDeleteURLUseCase(repo)
			res, err := uc.Execute(context.Background(), tt.request)

			// Check error
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

			// Check response
			if res == nil {
				t.Fatal("Execute() response is nil")
			}

			if tt.wantRes != nil {
				if res.Success != tt.wantRes.Success {
					t.Errorf("Execute() Success = %v, want %v", res.Success, tt.wantRes.Success)
				}
			}

			// Verify deletion actually happened for successful cases
			if tt.wantErr == nil {
				_, err := repo.FindByShortCode(context.Background(), tt.request.ShortCode)
				if !errors.Is(err, url.ErrURLNotFound) {
					t.Errorf("Execute() URL still exists after deletion, want ErrURLNotFound")
				}
			}
		})
	}
}

func TestDeleteURLUseCase_Execute_VerifyNoSideEffects(t *testing.T) {
	t.Run("failed deletion should not modify data", func(t *testing.T) {
		repo := newMockRepository()
		repo.urls["protected"] = &url.URL{
			ID:          1,
			ShortCode:   "protected",
			OriginalURL: "https://protected.com",
			CreatedAt:   time.Now(),
			CreatedBy:   "owner",
		}

		uc := NewDeleteURLUseCase(repo)

		// Attempt unauthorized deletion
		req := DeleteURLRequest{
			ShortCode:   "protected",
			RequestedBy: "attacker",
		}

		_, err := uc.Execute(context.Background(), req)
		if !errors.Is(err, url.ErrUnauthorizedDeletion) {
			t.Fatalf("Execute() error = %v, want ErrUnauthorizedDeletion", err)
		}

		// Verify URL still exists
		foundURL, err := repo.FindByShortCode(context.Background(), "protected")
		if err != nil {
			t.Errorf("FindByShortCode() error = %v, URL should still exist", err)
		}
		if foundURL == nil {
			t.Error("FindByShortCode() returned nil, URL should still exist")
		}
		if foundURL != nil && foundURL.CreatedBy != "owner" {
			t.Errorf("FindByShortCode() CreatedBy = %v, want owner", foundURL.CreatedBy)
		}
	})
}

func TestDeleteURLUseCase_Execute_MultipleUsers(t *testing.T) {
	t.Run("multiple users can each delete their own URLs", func(t *testing.T) {
		repo := newMockRepository()

		// Create URLs for different users
		repo.urls["user1-url"] = &url.URL{
			ID:          1,
			ShortCode:   "user1-url",
			OriginalURL: "https://user1.com",
			CreatedAt:   time.Now(),
			CreatedBy:   "user1",
		}
		repo.urls["user2-url"] = &url.URL{
			ID:          2,
			ShortCode:   "user2-url",
			OriginalURL: "https://user2.com",
			CreatedAt:   time.Now(),
			CreatedBy:   "user2",
		}

		uc := NewDeleteURLUseCase(repo)

		// User1 deletes their URL
		res1, err1 := uc.Execute(context.Background(), DeleteURLRequest{
			ShortCode:   "user1-url",
			RequestedBy: "user1",
		})
		if err1 != nil {
			t.Errorf("User1 deletion failed: %v", err1)
		}
		if res1 == nil || !res1.Success {
			t.Error("User1 deletion should succeed")
		}

		// User2 deletes their URL
		res2, err2 := uc.Execute(context.Background(), DeleteURLRequest{
			ShortCode:   "user2-url",
			RequestedBy: "user2",
		})
		if err2 != nil {
			t.Errorf("User2 deletion failed: %v", err2)
		}
		if res2 == nil || !res2.Success {
			t.Error("User2 deletion should succeed")
		}

		// Verify both URLs are deleted
		_, err := repo.FindByShortCode(context.Background(), "user1-url")
		if !errors.Is(err, url.ErrURLNotFound) {
			t.Error("user1-url should be deleted")
		}

		_, err = repo.FindByShortCode(context.Background(), "user2-url")
		if !errors.Is(err, url.ErrURLNotFound) {
			t.Error("user2-url should be deleted")
		}
	})
}

func TestDeleteURLUseCase_Execute_IdempotentCheck(t *testing.T) {
	t.Run("deleting already deleted URL returns not found", func(t *testing.T) {
		repo := newMockRepository()
		repo.urls["deleted"] = &url.URL{
			ID:          1,
			ShortCode:   "deleted",
			OriginalURL: "https://deleted.com",
			CreatedAt:   time.Now(),
			CreatedBy:   "user1",
		}

		uc := NewDeleteURLUseCase(repo)

		// First deletion
		req := DeleteURLRequest{
			ShortCode:   "deleted",
			RequestedBy: "user1",
		}
		res1, err1 := uc.Execute(context.Background(), req)
		if err1 != nil {
			t.Fatalf("First deletion failed: %v", err1)
		}
		if res1 == nil || !res1.Success {
			t.Fatal("First deletion should succeed")
		}

		// Second deletion attempt
		res2, err2 := uc.Execute(context.Background(), req)
		if !errors.Is(err2, url.ErrURLNotFound) {
			t.Errorf("Second deletion error = %v, want ErrURLNotFound", err2)
		}
		if res2 != nil {
			t.Error("Second deletion should return nil response")
		}
	})
}
