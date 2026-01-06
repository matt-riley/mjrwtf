package tui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/matt-riley/mjrwtf/internal/client"
	"github.com/matt-riley/mjrwtf/internal/tui/tui_config"
)

func TestModel_Update_Quit(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatalf("expected quit cmd")
	}
}

func TestModel_Update_ListURLsMsgSetsStatus(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	m.loading = true

	m2, _ := m.Update(listURLsMsg{urls: []tuiURL{}, total: 0})
	mm := m2.(model)
	if mm.loading {
		t.Fatalf("expected loading=false")
	}
	if !strings.Contains(mm.status, "Loaded") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_View_ShowsBaseURL(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	out := m.View()
	if !strings.Contains(out, "Base URL:") || !strings.Contains(out, "http://example") {
		t.Fatalf("expected base URL line")
	}
}

func TestStatusKindFromText(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   statusKind
	}{
		{"success_prefix_created", "Created: https://mjr.wtf/abc", statusKindSuccess},
		{"success_prefix_created_lower", "created: https://mjr.wtf/abc", statusKindSuccess},
		{"success_created_url_contains_failed", "Created: https://example.com/payment-failed", statusKindSuccess},
		{"success_prefix_deleted", "Deleted: abc", statusKindSuccess},
		{"success_prefix_deleted_lower", "deleted: abc", statusKindSuccess},
		{"success_contains", "ok - copied to clipboard", statusKindSuccess},
		{"error_failed_not_created", "Failed: resource not created", statusKindError},
		{"error_prefix_create_failed", "Create failed: unauthorized", statusKindError},
		{"error_prefix_list_failed", "List failed: 401", statusKindError},
		{"error_prefix_error", "Error: boom", statusKindError},
		{"error_contains_not_found", "Delete: abc not found", statusKindError},
		{"warning_prefix_warning", "Warning: something", statusKindWarning},
		{"warning_prefix_warn", "Warn: something", statusKindWarning},
		{"default_other", "Loading...", statusKindDefault},
		{"default_empty", " ", statusKindDefault},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := statusKindFromText(tt.status); got != tt.want {
				t.Fatalf("statusKindFromText(%q)=%v want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestModel_Update_CreateMode_Cancel(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	m.loading = false

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mm := m2.(model)
	if mm.mode != modeCreating {
		t.Fatalf("mode=%v", mm.mode)
	}

	m3, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm2 := m3.(model)
	if mm2.mode != modeBrowsing {
		t.Fatalf("mode=%v", mm2.mode)
	}
}

func TestModel_Update_CreateMode_InvalidURL(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	m.loading = false

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	mm := m2.(model)
	mm.createInput.SetValue("ftp://example.com")

	m3, _ := mm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm2 := m3.(model)
	if !strings.Contains(mm2.status, "http") {
		t.Fatalf("status=%q", mm2.status)
	}
	if mm2.createLoading {
		t.Fatalf("expected createLoading=false")
	}
}

func TestCreateURLCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/urls" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"original_url":"https://example.com"`) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"short_code":"abc123","short_url":"https://mjr.wtf/abc123","original_url":"https://example.com"}`))
	}))
	t.Cleanup(srv.Close)

	msg := createURLCmd(tui_config.Config{BaseURL: srv.URL, Token: "t"}, "https://example.com")().(createURLMsg)
	if msg.err != nil {
		t.Fatalf("expected nil err, got %v", msg.err)
	}
	if msg.resp == nil {
		t.Fatalf("expected resp")
	}
	if msg.resp.ShortCode != "abc123" {
		t.Fatalf("ShortCode=%q", msg.resp.ShortCode)
	}
}

func TestCreateURLCmd_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Unauthorized: invalid token"}`))
	}))
	t.Cleanup(srv.Close)

	msg := createURLCmd(tui_config.Config{BaseURL: srv.URL, Token: "t"}, "https://example.com")().(createURLMsg)
	if msg.err == nil {
		t.Fatalf("expected err")
	}
	apiErr, ok := msg.err.(*client.APIError)
	if !ok {
		t.Fatalf("expected *client.APIError, got %T", msg.err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("StatusCode=%d", apiErr.StatusCode)
	}
	if apiErr.Message == "" {
		t.Fatalf("expected message")
	}
}

func TestModel_Update_CreateURLMsg_ErrorResetsMode(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	m.loading = false
	m.mode = modeCreating
	m.createInput.SetValue("https://example.com")

	m2, _ := m.Update(createURLMsg{err: &client.APIError{StatusCode: 401, Message: "Unauthorized"}})
	mm := m2.(model)
	if mm.mode != modeBrowsing {
		t.Fatalf("mode=%v", mm.mode)
	}
	if mm.createInput.Value() != "" {
		t.Fatalf("expected createInput cleared")
	}
	if !strings.Contains(mm.status, "Create failed") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_Update_CreateURLMsg_SuccessCopiesAndRefreshes(t *testing.T) {
	old := clipboardWriteAll
	defer func() { clipboardWriteAll = old }()

	var copied string
	clipboardWriteAll = func(s string) error {
		copied = s
		return nil
	}

	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "abcdef"}, nil)
	m.loading = false
	m.mode = modeCreating
	m.offset = 40
	m.createInput.SetValue("https://example.com")

	m2, cmd := m.Update(createURLMsg{resp: &client.CreateURLResponse{ShortCode: "abc123", ShortURL: "https://mjr.wtf/abc123", OriginalURL: "https://example.com"}})
	mm := m2.(model)
	if mm.mode != modeBrowsing {
		t.Fatalf("mode=%v", mm.mode)
	}
	if mm.createInput.Value() != "" {
		t.Fatalf("expected createInput cleared")
	}
	if copied != "https://mjr.wtf/abc123" {
		t.Fatalf("copied=%q", copied)
	}
	if !strings.Contains(mm.status, "Created") {
		t.Fatalf("status=%q", mm.status)
	}
	if !mm.loading {
		t.Fatalf("expected loading=true")
	}
	if mm.offset != 0 {
		t.Fatalf("offset=%d", mm.offset)
	}
	if cmd == nil {
		t.Fatalf("expected refresh cmd")
	}
}

func TestModel_View_EmptyState_OffsetNonZeroNotMisleading(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.loading = false
	m.offset = 20
	m.urls = nil
	m.filtered = nil

	out := m.View()
	if !strings.Contains(out, "No URLs on this page") {
		t.Fatalf("expected page empty-state, got:\n%s", out)
	}
	if strings.Contains(out, "No URLs yet") {
		t.Fatalf("did not expect global empty-state when offset>0")
	}
}

func TestModel_View_EmptyState_FilterNotMisleading(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.loading = false
	m.filterQuery = "abc"
	m.urls = nil
	m.filtered = nil

	out := m.View()
	if !strings.Contains(out, "No matches for filter") {
		t.Fatalf("expected filter empty-state, got:\n%s", out)
	}
}

func TestListURLsCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/urls" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("limit") != "20" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("offset") != "40" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"urls":[{"id":1,"short_code":"abc123","original_url":"https://example.com","created_at":"2026-01-01T00:00:00Z","created_by":"me","click_count":2}],"total":1,"limit":20,"offset":40}`))
	}))
	t.Cleanup(srv.Close)

	msg := listURLsCmd(tui_config.Config{BaseURL: srv.URL, Token: "t"}, 20, 40)().(listURLsMsg)
	if msg.err != nil {
		t.Fatalf("expected nil err, got %v", msg.err)
	}
	if msg.total != 1 {
		t.Fatalf("total=%d", msg.total)
	}
	if len(msg.urls) != 1 || msg.urls[0].ShortCode != "abc123" {
		t.Fatalf("urls=%v", msg.urls)
	}
	if msg.urls[0].CreatedAt == nil {
		t.Fatalf("expected created_at")
	}

	msg = listURLsCmd(tui_config.Config{BaseURL: ""}, 20, 0)().(listURLsMsg)
	if msg.err == nil {
		t.Fatalf("expected err for empty baseURL")
	}
}

func TestModel_Update_OpenAnalytics(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.loading = false
	m.urls = []tuiURL{{ShortCode: "abc123", OriginalURL: "https://example.com"}}
	m.filtered = m.urls
	m.cursor = 0

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	mm := m2.(model)
	if mm.mode != modeViewingAnalytics {
		t.Fatalf("mode=%v", mm.mode)
	}
	if !mm.analyticsLoading {
		t.Fatalf("expected analyticsLoading=true")
	}
	if mm.analyticsShortCode != "abc123" {
		t.Fatalf("analyticsShortCode=%q", mm.analyticsShortCode)
	}
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
}

func TestModel_Update_AnalyticsTimeRange_RequiresBoth(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.mode = modeAnalyticsTimeRange
	m.analyticsShortCode = "abc123"
	m.analyticsRangeFocus = 1
	m.analyticsStartInput.SetValue("2025-11-20T00:00:00Z")
	m.analyticsEndInput.SetValue("")

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := m2.(model)
	if mm.mode != modeAnalyticsTimeRange {
		t.Fatalf("mode=%v", mm.mode)
	}
	if !strings.Contains(mm.status, "provided together") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_Update_AnalyticsTimeRange_EndAfterStart(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.mode = modeAnalyticsTimeRange
	m.analyticsShortCode = "abc123"
	m.analyticsRangeFocus = 1
	m.analyticsStartInput.SetValue("2025-11-22T23:59:59Z")
	m.analyticsEndInput.SetValue("2025-11-20T00:00:00Z")

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := m2.(model)
	if mm.mode != modeAnalyticsTimeRange {
		t.Fatalf("mode=%v", mm.mode)
	}
	if !strings.Contains(mm.status, "after") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_Update_AnalyticsTimeRange_SuccessFetches(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.mode = modeAnalyticsTimeRange
	m.analyticsShortCode = "abc123"
	m.analyticsRangeFocus = 1
	m.analyticsStartInput.SetValue("2025-11-20T00:00:00Z")
	m.analyticsEndInput.SetValue("2025-11-22T23:59:59Z")

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := m2.(model)
	if mm.mode != modeViewingAnalytics {
		t.Fatalf("mode=%v", mm.mode)
	}
	if !mm.analyticsLoading {
		t.Fatalf("expected analyticsLoading=true")
	}
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if mm.analyticsStartTime == nil || mm.analyticsEndTime == nil {
		t.Fatalf("expected times set")
	}
	if !mm.analyticsStartTime.Before(*mm.analyticsEndTime) {
		t.Fatalf("expected start < end")
	}
}

func TestDeleteURLCmd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/api/urls/abc123" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(srv.Close)

	msg := deleteURLCmd(tui_config.Config{BaseURL: srv.URL, Token: "t"}, "abc123")().(deleteURLMsg)
	if msg.err != nil {
		t.Fatalf("expected nil err, got %v", msg.err)
	}
	if msg.shortCode != "abc123" {
		t.Fatalf("shortCode=%q", msg.shortCode)
	}
}

func TestModel_Update_OpenDeleteConfirm(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.loading = false
	m.total = 1
	m.urls = []tuiURL{{ShortCode: "abc123", OriginalURL: "https://example.com"}}
	m.filtered = m.urls
	m.cursor = 0

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	mm := m2.(model)
	if mm.mode != modeDeleteConfirm {
		t.Fatalf("mode=%v", mm.mode)
	}
	if mm.deleteConfirmShortCode != "abc123" {
		t.Fatalf("deleteConfirmShortCode=%q", mm.deleteConfirmShortCode)
	}
	if !strings.Contains(mm.status, "Confirm") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_Update_DeleteConfirm_Cancel(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.mode = modeDeleteConfirm
	m.deleteConfirmShortCode = "abc123"
	m.deleteConfirmOriginalURL = "https://example.com"

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	mm := m2.(model)
	if mm.mode != modeBrowsing {
		t.Fatalf("mode=%v", mm.mode)
	}
	if !strings.Contains(mm.status, "cancel") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_Update_DeleteConfirm_ConfirmStartsDelete(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.mode = modeDeleteConfirm
	m.deleteConfirmShortCode = "abc123"

	m2, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	mm := m2.(model)
	if !mm.deleteLoading {
		t.Fatalf("expected deleteLoading=true")
	}
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
}

func TestModel_Update_DeleteURLMsg_SuccessRemovesItem(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.loading = false
	m.total = 2
	m.urls = []tuiURL{{ShortCode: "abc123"}, {ShortCode: "def456"}}
	m.filtered = m.urls
	m.cursor = 0

	m2, _ := m.Update(deleteURLMsg{shortCode: "abc123"})
	mm := m2.(model)
	if len(mm.urls) != 1 || mm.urls[0].ShortCode != "def456" {
		t.Fatalf("urls=%v", mm.urls)
	}
	if mm.total != 1 {
		t.Fatalf("total=%d", mm.total)
	}
	if !strings.Contains(mm.status, "Deleted") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_Update_DeleteURLMsg_NotFoundRefreshes(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.loading = false

	m2, cmd := m.Update(deleteURLMsg{shortCode: "abc123", err: &client.APIError{StatusCode: 404, Message: "Not found"}})
	mm := m2.(model)
	if !mm.loading {
		t.Fatalf("expected loading=true")
	}
	if cmd == nil {
		t.Fatalf("expected cmd")
	}
	if !strings.Contains(mm.status, "not found") {
		t.Fatalf("status=%q", mm.status)
	}
}

func TestModel_Update_DeleteURLMsg_APIErrorDoesNotRefresh(t *testing.T) {
	m := newModel(tui_config.Config{BaseURL: "http://example", Token: "t"}, nil)
	m.loading = false
	m.mode = modeDeleteConfirm
	m.deleteLoading = true
	m.deleteConfirmShortCode = "abc123"
	m.deleteConfirmOriginalURL = "https://example.com"

	m2, cmd := m.Update(deleteURLMsg{shortCode: "abc123", err: &client.APIError{StatusCode: 401, Message: "Unauthorized"}})
	mm := m2.(model)
	if mm.deleteLoading {
		t.Fatalf("expected deleteLoading=false")
	}
	if mm.mode != modeBrowsing {
		t.Fatalf("mode=%v", mm.mode)
	}
	if cmd != nil {
		t.Fatalf("expected no cmd")
	}
	if mm.loading {
		t.Fatalf("expected loading=false")
	}
	if !strings.Contains(mm.status, "Delete failed (401)") {
		t.Fatalf("status=%q", mm.status)
	}
	if mm.deleteConfirmShortCode != "" || mm.deleteConfirmOriginalURL != "" {
		t.Fatalf("expected confirm fields cleared")
	}
}
