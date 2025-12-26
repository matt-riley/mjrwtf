package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

)

// TestE2E_FullWorkflow tests the complete end-to-end workflow:
// 1. Authenticate with valid token
// 2. Create a shortened URL
// 3. Redirect using the short code
// 4. Verify analytics were recorded
func TestE2E_FullWorkflow(t *testing.T) {
	// Setup test database and server
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Step 1: Create shortened URL with authentication
	t.Run("create_url", func(t *testing.T) {
		reqBody := `{"original_url":"https://example.com/test-page"}`
		req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if response["short_code"] == nil || response["short_code"] == "" {
			t.Error("expected short_code in response")
		}
		if response["short_url"] == nil {
			t.Error("expected short_url in response")
		}
		if response["original_url"] != "https://example.com/test-page" {
			t.Errorf("expected original_url to match, got %v", response["original_url"])
		}

		shortCode := response["short_code"].(string)
		t.Logf("Created short URL with code: %s", shortCode)

		// Step 2: Perform redirect using short code
		t.Run("redirect", func(t *testing.T) {
			redirectReq := httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
			redirectReq.Header.Set("Referer", "https://google.com")
			redirectReq.Header.Set("User-Agent", "Mozilla/5.0 (Test Browser)")

			redirectRec := httptest.NewRecorder()
			srv.router.ServeHTTP(redirectRec, redirectReq)

			if redirectRec.Code != http.StatusFound {
				t.Fatalf("expected status %d, got %d", http.StatusFound, redirectRec.Code)
			}

			location := redirectRec.Header().Get("Location")
			if location != "https://example.com/test-page" {
				t.Errorf("expected redirect to https://example.com/test-page, got %s", location)
			}

			t.Logf("Successfully redirected to: %s", location)
		})

		// Step 3: Verify analytics were recorded
		t.Run("verify_analytics", func(t *testing.T) {
			// Give initial time for async workers to process
			time.Sleep(100 * time.Millisecond)

			// Poll for async click recording instead of using only a fixed sleep
			const (
				analyticsTimeout      = 5 * time.Second
				analyticsPollInterval = 50 * time.Millisecond
			)

			deadline := time.Now().Add(analyticsTimeout)
			var (
				lastStatus int
				lastBody   string
			)

			for time.Now().Before(deadline) {
				analyticsReq := httptest.NewRequest(http.MethodGet, "/api/urls/"+shortCode+"/analytics", nil)
				analyticsReq.Header.Set("Authorization", "Bearer test-token")

				analyticsRec := httptest.NewRecorder()
				srv.router.ServeHTTP(analyticsRec, analyticsReq)

				lastStatus = analyticsRec.Code
				lastBody = analyticsRec.Body.String()

				if analyticsRec.Code == http.StatusOK {
					var analytics map[string]interface{}
					if err := json.Unmarshal(analyticsRec.Body.Bytes(), &analytics); err != nil {
						// If parsing fails, keep retrying until timeout
						time.Sleep(analyticsPollInterval)
						continue
					}

					// Verify click count is at least 1
					if totalClicks, ok := analytics["total_clicks"].(float64); ok && totalClicks >= 1 {
						t.Logf("Analytics verified: %d total clicks", int(totalClicks))
						return
					}
				}

				time.Sleep(analyticsPollInterval)
			}

			t.Fatalf("expected analytics to be recorded within %s, last status %d, last body: %s", analyticsTimeout, lastStatus, lastBody)
		})
	})
}

// TestE2E_AuthenticationFlow tests the authentication workflow
func TestE2E_AuthenticationFlow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		description    string
	}{
		{
			name:           "valid_token",
			authHeader:     "Bearer test-token",
			expectedStatus: http.StatusCreated,
			description:    "Valid authentication token should succeed",
		},
		{
			name:           "invalid_token",
			authHeader:     "Bearer wrong-token",
			expectedStatus: http.StatusUnauthorized,
			description:    "Invalid token should be rejected",
		},
		{
			name:           "missing_token",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
			description:    "Missing token should be rejected",
		},
		{
			name:           "malformed_header",
			authHeader:     "InvalidFormat",
			expectedStatus: http.StatusUnauthorized,
			description:    "Malformed auth header should be rejected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := `{"original_url":"https://example.com"}`
			req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(reqBody))
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("%s: expected status %d, got %d", tt.description, tt.expectedStatus, rec.Code)
			}
		})
	}
}

// TestE2E_ErrorScenarios tests various error scenarios
func TestE2E_ErrorScenarios(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	t.Run("not_found_short_code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status %d for nonexistent short code, got %d", http.StatusNotFound, rec.Code)
		}
	})

	t.Run("invalid_url_format", func(t *testing.T) {
		reqBody := `{"original_url":"not-a-valid-url"}`
		req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for invalid URL, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("empty_url", func(t *testing.T) {
		reqBody := `{"original_url":""}`
		req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for empty URL, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("malformed_json", func(t *testing.T) {
		reqBody := `{"original_url": invalid json}`
		req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status %d for malformed JSON, got %d", http.StatusBadRequest, rec.Code)
		}
	})

	t.Run("analytics_for_nonexistent_url", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls/nonexistent/analytics", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("expected status %d for analytics of nonexistent URL, got %d", http.StatusNotFound, rec.Code)
		}
	})
}

// TestE2E_APIEndpoints tests all API endpoints comprehensively
func TestE2E_APIEndpoints(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	var createdShortCode string

	// Test POST /api/urls - Create URL
	t.Run("create_url", func(t *testing.T) {
		reqBody := `{"original_url":"https://example.com/api-test"}`
		req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(reqBody))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d. Body: %s", http.StatusCreated, rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		createdShortCode = response["short_code"].(string)
		t.Logf("Created URL with short code: %s", createdShortCode)
	})

	// Test GET /api/urls - List URLs
	t.Run("list_urls", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		urls, ok := response["urls"].([]interface{})
		if !ok {
			t.Fatal("expected urls array in response")
		}

		if len(urls) == 0 {
			t.Error("expected at least one URL in list")
		}

		t.Logf("Found %d URLs in list", len(urls))
	})

	// Test GET /{shortCode}/analytics - Get Analytics
	t.Run("get_analytics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/urls/"+createdShortCode+"/analytics", nil)
		req.Header.Set("Authorization", "Bearer test-token")

		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d. Body: %s", http.StatusOK, rec.Code, rec.Body.String())
		}

		var analytics map[string]interface{}
		if err := json.Unmarshal(rec.Body.Bytes(), &analytics); err != nil {
			t.Fatalf("failed to parse analytics: %v", err)
		}

		if analytics["short_code"] != createdShortCode {
			t.Errorf("expected short_code %s, got %v", createdShortCode, analytics["short_code"])
		}

		t.Logf("Analytics: %v", analytics)
	})

	// Test DELETE /api/urls/{shortCode} - Delete URL
	t.Run("delete_url", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/api/urls/"+createdShortCode, nil)
		req.Header.Set("Authorization", "Bearer test-token")

		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusNoContent && rec.Code != http.StatusOK {
			t.Fatalf("expected status %d or %d, got %d. Body: %s", http.StatusNoContent, http.StatusOK, rec.Code, rec.Body.String())
		}

		t.Logf("Successfully deleted URL: %s", createdShortCode)

		// Verify URL is actually deleted
		verifyReq := httptest.NewRequest(http.MethodGet, "/"+createdShortCode, nil)
		verifyRec := httptest.NewRecorder()
		srv.router.ServeHTTP(verifyRec, verifyReq)

		if verifyRec.Code != http.StatusNotFound {
			t.Errorf("expected deleted URL to return 404, got %d", verifyRec.Code)
		}
	})
}

// TestE2E_MultipleClicks tests analytics with multiple clicks
func TestE2E_MultipleClicks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	// Create URL
	reqBody := `{"original_url":"https://example.com/multi-click"}`
	req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(reqBody))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("failed to create URL: %s", rec.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal create URL response: %v", err)
	}
	shortCode := response["short_code"].(string)

	// Perform multiple clicks from different sources (sequentially to avoid race conditions)
	referrers := []string{
		"https://google.com",
		"https://twitter.com",
		"https://facebook.com",
	}

	for i, referrer := range referrers {
		clickReq := httptest.NewRequest(http.MethodGet, "/"+shortCode, nil)
		if referrer != "" {
			clickReq.Header.Set("Referer", referrer)
		}
		clickReq.Header.Set("User-Agent", fmt.Sprintf("Browser-%d", i))

		clickRec := httptest.NewRecorder()
		srv.router.ServeHTTP(clickRec, clickReq)

		if clickRec.Code != http.StatusFound {
			t.Errorf("click %d: expected status %d, got %d. Body: %s", i, http.StatusFound, clickRec.Code, clickRec.Body.String())
		}

		// Small delay to give async click recording workers time to process between requests
		time.Sleep(30 * time.Millisecond)
	}

	// Give initial time for async workers to process
	time.Sleep(100 * time.Millisecond)

	// Poll for async click recording to complete, waiting until all clicks are recorded or timeout
	deadline := time.Now().Add(5 * time.Second)

	var (
		analytics    map[string]interface{}
		totalClicks  float64
		ok           bool
		analyticsRec *httptest.ResponseRecorder
	)

	for {
		analyticsReq := httptest.NewRequest(http.MethodGet, "/api/urls/"+shortCode+"/analytics", nil)
		analyticsReq.Header.Set("Authorization", "Bearer test-token")

		analyticsRec = httptest.NewRecorder()
		srv.router.ServeHTTP(analyticsRec, analyticsReq)

		if analyticsRec.Code == http.StatusOK {
			analytics = make(map[string]interface{})
			if err := json.Unmarshal(analyticsRec.Body.Bytes(), &analytics); err == nil {
				if totalClicks, ok = analytics["total_clicks"].(float64); ok && int(totalClicks) >= len(referrers) {
					break
				}
			}
		}

		if time.Now().After(deadline) {
			t.Fatalf("timeout waiting for analytics to record %d clicks; last response code=%d body=%s", len(referrers), analyticsRec.Code, analyticsRec.Body.String())
		}

		time.Sleep(50 * time.Millisecond)
	}

	totalClicks, ok = analytics["total_clicks"].(float64)
	if !ok {
		t.Fatal("expected total_clicks in analytics")
	}

	expectedClicks := float64(len(referrers))
	if totalClicks < expectedClicks {
		t.Errorf("expected at least %.0f clicks, got %.0f", expectedClicks, totalClicks)
	}

	t.Logf("Successfully recorded %.0f clicks (expected %.0f)", totalClicks, expectedClicks)
}

// TestE2E_ConcurrentCreation tests concurrent URL creation
func TestE2E_ConcurrentCreation(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	const numRequests = 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(index int) {
			// NOTE: Small stagger to prevent overwhelming the in-memory SQLite database
			// with concurrent writes. The test database uses ":memory:", which cannot
			// enable WAL mode at all. Without WAL, SQLite serializes writes and still
			// returns SQLITE_BUSY under high concurrent load even when SetMaxOpenConns(1)
			// is used. This stagger offsets request start times to reduce contention.
			// In production, a file-based SQLite database runs in WAL mode and handles
			// concurrency better, so this workaround is not required there.
			time.Sleep(time.Duration(index) * 5 * time.Millisecond)

			reqBody := fmt.Sprintf(`{"original_url":"https://example.com/concurrent-%d"}`, index)
			req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(reqBody))
			req.Header.Set("Authorization", "Bearer test-token")
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			srv.router.ServeHTTP(rec, req)

			if rec.Code != http.StatusCreated {
				results <- fmt.Errorf("request %d failed with status %d: %s", index, rec.Code, rec.Body.String())
				return
			}
			results <- nil
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Error(err)
		} else {
			successCount++
		}
	}

	t.Logf("Successfully created %d/%d URLs concurrently", successCount, numRequests)
}

// TestE2E_HealthCheck tests the health check endpoint
func TestE2E_HealthCheck(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := testConfig()

	srv, err := New(cfg, db, testLogger())
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}
	defer srv.Shutdown(context.Background())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	srv.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	expected := `{"status":"ok"}`
	if rec.Body.String() != expected {
		t.Errorf("expected response %s, got %s", expected, rec.Body.String())
	}

	// Verify no authentication is required for health check
	t.Run("no_auth_required", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("health check should work without auth, got status %d", rec.Code)
		}
	})
}
