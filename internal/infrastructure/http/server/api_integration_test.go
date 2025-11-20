package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/matt-riley/mjrwtf/internal/infrastructure/config"
)

// TestAPIEndpoints_CreateURL tests the POST /api/urls endpoint
func TestAPIEndpoints_CreateURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv, err := New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	tests := []struct {
		name           string
		authToken      string
		requestBody    string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "successful creation with auth",
			authToken:      "test-token",
			requestBody:    `{"original_url":"https://example.com"}`,
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["short_code"] == nil || body["short_code"] == "" {
					t.Error("expected short_code in response")
				}
				if body["short_url"] == nil || body["short_url"] == "" {
					t.Error("expected short_url in response")
				}
				if body["original_url"] != "https://example.com" {
					t.Errorf("expected original_url to be https://example.com, got %v", body["original_url"])
				}
			},
		},
		{
			name:           "missing auth token",
			authToken:      "",
			requestBody:    `{"original_url":"https://example.com"}`,
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("expected error in response")
				}
			},
		},
		{
			name:           "invalid auth token",
			authToken:      "wrong-token",
			requestBody:    `{"original_url":"https://example.com"}`,
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("expected error in response")
				}
			},
		},
		{
			name:           "empty original URL",
			authToken:      "test-token",
			requestBody:    `{"original_url":""}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("expected error in response")
				}
			},
		},
		{
			name:           "invalid URL format",
			authToken:      "test-token",
			requestBody:    `{"original_url":"not-a-valid-url"}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("expected error in response")
				}
			},
		},
		{
			name:           "invalid JSON",
			authToken:      "test-token",
			requestBody:    `{invalid json}`,
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("expected error in response")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")
			if tt.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.authToken)
			}

			rec := httptest.NewRecorder()
			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}

// TestAPIEndpoints_ListURLs tests the GET /api/urls endpoint
func TestAPIEndpoints_ListURLs(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv, err := New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// First, create some URLs
	createURL := func(originalURL string) {
		body := `{"original_url":"` + originalURL + `"}`
		req := httptest.NewRequest(http.MethodPost, "/api/urls", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test-token")
		rec := httptest.NewRecorder()
		srv.router.ServeHTTP(rec, req)
	}

	createURL("https://example.com/1")
	createURL("https://example.com/2")
	createURL("https://example.com/3")

	tests := []struct {
		name           string
		authToken      string
		queryParams    string
		expectedStatus int
		checkResponse  func(t *testing.T, body map[string]interface{})
	}{
		{
			name:           "successful list with auth",
			authToken:      "test-token",
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				urls, ok := body["urls"].([]interface{})
				if !ok {
					t.Error("expected urls array in response")
					return
				}
				if len(urls) != 3 {
					t.Errorf("expected 3 URLs, got %d", len(urls))
				}
			},
		},
		{
			name:           "list with pagination",
			authToken:      "test-token",
			queryParams:    "?limit=2&offset=0",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				urls, ok := body["urls"].([]interface{})
				if !ok {
					t.Error("expected urls array in response")
					return
				}
				if len(urls) != 2 {
					t.Errorf("expected 2 URLs with limit=2, got %d", len(urls))
				}
				if body["limit"] != float64(2) {
					t.Errorf("expected limit to be 2, got %v", body["limit"])
				}
			},
		},
		{
			name:           "missing auth token",
			authToken:      "",
			queryParams:    "",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("expected error in response")
				}
			},
		},
		{
			name:           "invalid auth token",
			authToken:      "wrong-token",
			queryParams:    "",
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, body map[string]interface{}) {
				if body["error"] == nil {
					t.Error("expected error in response")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/urls"+tt.queryParams, nil)
			if tt.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.authToken)
			}

			rec := httptest.NewRecorder()
			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			var body map[string]interface{}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, body)
			}
		})
	}
}

// TestAPIEndpoints_DeleteURL tests the DELETE /api/urls/:shortCode endpoint
func TestAPIEndpoints_DeleteURL(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv, err := New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create a URL to delete
	createReq := httptest.NewRequest(http.MethodPost, "/api/urls",
		bytes.NewBufferString(`{"original_url":"https://example.com"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer test-token")
	createRec := httptest.NewRecorder()
	srv.router.ServeHTTP(createRec, createReq)

	var createResp map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResp)
	shortCode := createResp["short_code"].(string)

	tests := []struct {
		name           string
		authToken      string
		shortCode      string
		expectedStatus int
		checkResponse  func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:           "successful deletion with auth",
			authToken:      "test-token",
			shortCode:      shortCode,
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				if rec.Body.Len() != 0 {
					t.Error("expected empty response body for no content")
				}
			},
		},
		{
			name:           "missing auth token",
			authToken:      "",
			shortCode:      "abc123",
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  nil,
		},
		{
			name:           "invalid auth token",
			authToken:      "wrong-token",
			shortCode:      "abc123",
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  nil,
		},
		{
			name:           "non-existent short code",
			authToken:      "test-token",
			shortCode:      "nonexistent",
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var body map[string]interface{}
				if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if body["error"] == nil {
					t.Error("expected error in response")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodDelete, "/api/urls/"+tt.shortCode, nil)
			if tt.authToken != "" {
				req.Header.Set("Authorization", "Bearer "+tt.authToken)
			}

			rec := httptest.NewRecorder()
			srv.router.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rec.Code)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

// TestAPIEndpoints_FullWorkflow tests a complete workflow: create, list, delete
func TestAPIEndpoints_FullWorkflow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := &config.Config{
		ServerPort:     8080,
		BaseURL:        "http://localhost:8080",
		DatabaseURL:    "test.db",
		AuthToken:      "test-token",
		AllowedOrigins: "*",
	}

	srv, err := New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Step 1: Create a URL
	createReq := httptest.NewRequest(http.MethodPost, "/api/urls",
		bytes.NewBufferString(`{"original_url":"https://example.com/test"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createReq.Header.Set("Authorization", "Bearer test-token")
	createRec := httptest.NewRecorder()
	srv.router.ServeHTTP(createRec, createReq)

	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected status %d when creating URL, got %d", http.StatusCreated, createRec.Code)
	}

	var createResp map[string]interface{}
	json.Unmarshal(createRec.Body.Bytes(), &createResp)
	shortCode := createResp["short_code"].(string)

	// Step 2: List URLs and verify it's there
	listReq := httptest.NewRequest(http.MethodGet, "/api/urls", nil)
	listReq.Header.Set("Authorization", "Bearer test-token")
	listRec := httptest.NewRecorder()
	srv.router.ServeHTTP(listRec, listReq)

	if listRec.Code != http.StatusOK {
		t.Fatalf("expected status %d when listing URLs, got %d", http.StatusOK, listRec.Code)
	}

	var listResp map[string]interface{}
	json.Unmarshal(listRec.Body.Bytes(), &listResp)
	urls := listResp["urls"].([]interface{})

	found := false
	for _, urlData := range urls {
		urlMap := urlData.(map[string]interface{})
		if urlMap["short_code"] == shortCode {
			found = true
			break
		}
	}

	if !found {
		t.Error("created URL not found in list")
	}

	// Step 3: Delete the URL
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/urls/"+shortCode, nil)
	deleteReq.Header.Set("Authorization", "Bearer test-token")
	deleteRec := httptest.NewRecorder()
	srv.router.ServeHTTP(deleteRec, deleteReq)

	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("expected status %d when deleting URL, got %d", http.StatusNoContent, deleteRec.Code)
	}

	// Step 4: Verify it's deleted by trying to delete again
	deleteReq2 := httptest.NewRequest(http.MethodDelete, "/api/urls/"+shortCode, nil)
	deleteReq2.Header.Set("Authorization", "Bearer test-token")
	deleteRec2 := httptest.NewRecorder()
	srv.router.ServeHTTP(deleteRec2, deleteReq2)

	if deleteRec2.Code != http.StatusNotFound {
		t.Errorf("expected status %d when deleting non-existent URL, got %d", http.StatusNotFound, deleteRec2.Code)
	}
}
