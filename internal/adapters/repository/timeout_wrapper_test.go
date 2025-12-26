package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// mockURLRepository is a mock that can simulate slow operations
type mockURLRepository struct {
	createDelay                      time.Duration
	findByShortCodeDelay             time.Duration
	deleteDelay                      time.Duration
	listDelay                        time.Duration
	listByCreatedByAndTimeRangeDelay time.Duration
	countDelay                       time.Duration
	createErr                        error
	findByShortCodeErr               error
	deleteErr                        error
	listErr                          error
	listByCreatedByAndTimeRangeErr   error
	countErr                         error
	lastCtxCancelled                 bool
}

func (m *mockURLRepository) Create(ctx context.Context, u *url.URL) error {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		m.lastCtxCancelled = true
		return ctx.Err()
	default:
	}

	if m.createDelay > 0 {
		select {
		case <-time.After(m.createDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return ctx.Err()
		}
	}
	return m.createErr
}

func (m *mockURLRepository) FindByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	if m.findByShortCodeDelay > 0 {
		select {
		case <-time.After(m.findByShortCodeDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return nil, ctx.Err()
		}
	}
	if m.findByShortCodeErr != nil {
		return nil, m.findByShortCodeErr
	}
	return &url.URL{ShortCode: shortCode}, nil
}

func (m *mockURLRepository) Delete(ctx context.Context, shortCode string) error {
	if m.deleteDelay > 0 {
		select {
		case <-time.After(m.deleteDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return ctx.Err()
		}
	}
	return m.deleteErr
}

func (m *mockURLRepository) List(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
	if m.listDelay > 0 {
		select {
		case <-time.After(m.listDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return nil, ctx.Err()
		}
	}
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []*url.URL{}, nil
}

func (m *mockURLRepository) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	if m.listByCreatedByAndTimeRangeDelay > 0 {
		select {
		case <-time.After(m.listByCreatedByAndTimeRangeDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return nil, ctx.Err()
		}
	}
	if m.listByCreatedByAndTimeRangeErr != nil {
		return nil, m.listByCreatedByAndTimeRangeErr
	}
	return []*url.URL{}, nil
}

func (m *mockURLRepository) Count(ctx context.Context, createdBy string) (int, error) {
	if m.countDelay > 0 {
		select {
		case <-time.After(m.countDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return 0, ctx.Err()
		}
	}
	return 0, m.countErr
}

// mockClickRepository is a mock that can simulate slow operations
type mockClickRepository struct {
	recordDelay                    time.Duration
	getStatsByURLDelay             time.Duration
	getStatsByURLAndTimeRangeDelay time.Duration
	getTotalClickCountDelay        time.Duration
	getClicksByCountryDelay        time.Duration
	recordErr                      error
	getStatsByURLErr               error
	getStatsByURLAndTimeRangeErr   error
	getTotalClickCountErr          error
	getClicksByCountryErr          error
	lastCtxCancelled               bool
}

func (m *mockClickRepository) Record(ctx context.Context, c *click.Click) error {
	if m.recordDelay > 0 {
		select {
		case <-time.After(m.recordDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return ctx.Err()
		}
	}
	return m.recordErr
}

func (m *mockClickRepository) GetStatsByURL(ctx context.Context, urlID int64) (*click.Stats, error) {
	if m.getStatsByURLDelay > 0 {
		select {
		case <-time.After(m.getStatsByURLDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return nil, ctx.Err()
		}
	}
	if m.getStatsByURLErr != nil {
		return nil, m.getStatsByURLErr
	}
	return &click.Stats{}, nil
}

func (m *mockClickRepository) GetStatsByURLAndTimeRange(ctx context.Context, urlID int64, startTime, endTime time.Time) (*click.TimeRangeStats, error) {
	if m.getStatsByURLAndTimeRangeDelay > 0 {
		select {
		case <-time.After(m.getStatsByURLAndTimeRangeDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return nil, ctx.Err()
		}
	}
	if m.getStatsByURLAndTimeRangeErr != nil {
		return nil, m.getStatsByURLAndTimeRangeErr
	}
	return &click.TimeRangeStats{}, nil
}

func (m *mockClickRepository) GetTotalClickCount(ctx context.Context, urlID int64) (int64, error) {
	if m.getTotalClickCountDelay > 0 {
		select {
		case <-time.After(m.getTotalClickCountDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return 0, ctx.Err()
		}
	}
	return 0, m.getTotalClickCountErr
}

func (m *mockClickRepository) GetClicksByCountry(ctx context.Context, urlID int64) (map[string]int64, error) {
	if m.getClicksByCountryDelay > 0 {
		select {
		case <-time.After(m.getClicksByCountryDelay):
		case <-ctx.Done():
			m.lastCtxCancelled = true
			return nil, ctx.Err()
		}
	}
	if m.getClicksByCountryErr != nil {
		return nil, m.getClicksByCountryErr
	}
	return make(map[string]int64), nil
}

func TestURLRepositoryWithTimeout_Create_Timeout(t *testing.T) {
	mock := &mockURLRepository{
		createDelay: 200 * time.Millisecond, // Longer than timeout
	}
	repo := NewURLRepositoryWithTimeout(mock, 50*time.Millisecond)

	u, _ := url.NewURL("test", "https://example.com", "user")
	err := repo.Create(context.Background(), u)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Create() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("Create() should have received cancelled context")
	}
}

func TestURLRepositoryWithTimeout_Create_Success(t *testing.T) {
	mock := &mockURLRepository{
		createDelay: 10 * time.Millisecond, // Shorter than timeout
	}
	repo := NewURLRepositoryWithTimeout(mock, 100*time.Millisecond)

	u, _ := url.NewURL("test", "https://example.com", "user")
	err := repo.Create(context.Background(), u)

	if err != nil {
		t.Errorf("Create() without timeout should succeed, got %v", err)
	}
}

func TestURLRepositoryWithTimeout_Create_ContextAlreadyCancelled(t *testing.T) {
	mock := &mockURLRepository{}
	repo := NewURLRepositoryWithTimeout(mock, 100*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	u, _ := url.NewURL("test", "https://example.com", "user")
	err := repo.Create(ctx, u)

	// When parent context is cancelled, the operation should still fail with context error
	if err == nil {
		t.Error("Create() with cancelled context should fail")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Create() with cancelled context expected Canceled or DeadlineExceeded, got %v", err)
	}
}

func TestURLRepositoryWithTimeout_FindByShortCode_Timeout(t *testing.T) {
	mock := &mockURLRepository{
		findByShortCodeDelay: 200 * time.Millisecond,
	}
	repo := NewURLRepositoryWithTimeout(mock, 50*time.Millisecond)

	_, err := repo.FindByShortCode(context.Background(), "test")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("FindByShortCode() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("FindByShortCode() should have received cancelled context")
	}
}

func TestURLRepositoryWithTimeout_Delete_Timeout(t *testing.T) {
	mock := &mockURLRepository{
		deleteDelay: 200 * time.Millisecond,
	}
	repo := NewURLRepositoryWithTimeout(mock, 50*time.Millisecond)

	err := repo.Delete(context.Background(), "test")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Delete() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("Delete() should have received cancelled context")
	}
}

func TestURLRepositoryWithTimeout_List_Timeout(t *testing.T) {
	mock := &mockURLRepository{
		listDelay: 200 * time.Millisecond,
	}
	repo := NewURLRepositoryWithTimeout(mock, 50*time.Millisecond)

	_, err := repo.List(context.Background(), "user", 10, 0)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("List() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("List() should have received cancelled context")
	}
}

func TestURLRepositoryWithTimeout_ListByCreatedByAndTimeRange_Timeout(t *testing.T) {
	mock := &mockURLRepository{
		listByCreatedByAndTimeRangeDelay: 200 * time.Millisecond,
	}
	repo := NewURLRepositoryWithTimeout(mock, 50*time.Millisecond)

	_, err := repo.ListByCreatedByAndTimeRange(context.Background(), "user", time.Now(), time.Now())

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("ListByCreatedByAndTimeRange() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("ListByCreatedByAndTimeRange() should have received cancelled context")
	}
}

func TestURLRepositoryWithTimeout_Count_Timeout(t *testing.T) {
	mock := &mockURLRepository{
		countDelay: 200 * time.Millisecond,
	}
	repo := NewURLRepositoryWithTimeout(mock, 50*time.Millisecond)

	_, err := repo.Count(context.Background(), "user")

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Count() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("Count() should have received cancelled context")
	}
}

func TestClickRepositoryWithTimeout_Record_Timeout(t *testing.T) {
	mock := &mockClickRepository{
		recordDelay: 200 * time.Millisecond,
	}
	repo := NewClickRepositoryWithTimeout(mock, 50*time.Millisecond)

	c, _ := click.NewClick(1, "", "", "")
	err := repo.Record(context.Background(), c)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Record() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("Record() should have received cancelled context")
	}
}

func TestClickRepositoryWithTimeout_Record_Success(t *testing.T) {
	mock := &mockClickRepository{
		recordDelay: 10 * time.Millisecond,
	}
	repo := NewClickRepositoryWithTimeout(mock, 100*time.Millisecond)

	c, _ := click.NewClick(1, "", "", "")
	err := repo.Record(context.Background(), c)

	if err != nil {
		t.Errorf("Record() without timeout should succeed, got %v", err)
	}
}

func TestClickRepositoryWithTimeout_GetStatsByURL_Timeout(t *testing.T) {
	mock := &mockClickRepository{
		getStatsByURLDelay: 200 * time.Millisecond,
	}
	repo := NewClickRepositoryWithTimeout(mock, 50*time.Millisecond)

	_, err := repo.GetStatsByURL(context.Background(), 1)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("GetStatsByURL() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("GetStatsByURL() should have received cancelled context")
	}
}

func TestClickRepositoryWithTimeout_GetStatsByURLAndTimeRange_Timeout(t *testing.T) {
	mock := &mockClickRepository{
		getStatsByURLAndTimeRangeDelay: 200 * time.Millisecond,
	}
	repo := NewClickRepositoryWithTimeout(mock, 50*time.Millisecond)

	_, err := repo.GetStatsByURLAndTimeRange(context.Background(), 1, time.Now(), time.Now())

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("GetStatsByURLAndTimeRange() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("GetStatsByURLAndTimeRange() should have received cancelled context")
	}
}

func TestClickRepositoryWithTimeout_GetTotalClickCount_Timeout(t *testing.T) {
	mock := &mockClickRepository{
		getTotalClickCountDelay: 200 * time.Millisecond,
	}
	repo := NewClickRepositoryWithTimeout(mock, 50*time.Millisecond)

	_, err := repo.GetTotalClickCount(context.Background(), 1)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("GetTotalClickCount() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("GetTotalClickCount() should have received cancelled context")
	}
}

func TestClickRepositoryWithTimeout_GetClicksByCountry_Timeout(t *testing.T) {
	mock := &mockClickRepository{
		getClicksByCountryDelay: 200 * time.Millisecond,
	}
	repo := NewClickRepositoryWithTimeout(mock, 50*time.Millisecond)

	_, err := repo.GetClicksByCountry(context.Background(), 1)

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("GetClicksByCountry() with timeout expected DeadlineExceeded, got %v", err)
	}
	if !mock.lastCtxCancelled {
		t.Error("GetClicksByCountry() should have received cancelled context")
	}
}

// Integration test with real SQLite database to verify timeout behavior
func TestURLRepositoryWithTimeout_Integration_SlowQuery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	baseRepo := NewSQLiteURLRepository(db)
	repo := NewURLRepositoryWithTimeout(baseRepo, 1*time.Millisecond) // Very short timeout

	// Create a URL first
	u, _ := url.NewURL("test", "https://example.com", "user")
	createErr := baseRepo.Create(context.Background(), u) // Use base repo with no timeout
	if createErr != nil {
		t.Fatalf("Failed to create test URL: %v", createErr)
	}

	// Now try to query with timeout - might or might not timeout depending on DB speed
	// This test mainly verifies that the timeout wrapper doesn't break functionality
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	found, err := repo.FindByShortCode(ctx, "test")
	// Either it succeeds or times out, both are acceptable - we just verify no panic
	if err != nil && !errors.Is(err, context.DeadlineExceeded) && err != sql.ErrNoRows {
		t.Errorf("FindByShortCode() unexpected error: %v", err)
	}
	if err == nil && found == nil {
		t.Error("FindByShortCode() returned nil without error")
	}
}
