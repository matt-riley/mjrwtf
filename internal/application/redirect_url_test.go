package application

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// Mock URL Repository
type mockURLRepository struct {
	urls      map[string]*url.URL
	mu        sync.RWMutex
	findError error
}

func newMockURLRepository() *mockURLRepository {
	return &mockURLRepository{
		urls: make(map[string]*url.URL),
	}
}

func (m *mockURLRepository) Create(ctx context.Context, u *url.URL) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.urls[u.ShortCode]; exists {
		return url.ErrDuplicateShortCode
	}
	m.urls[u.ShortCode] = u
	return nil
}

func (m *mockURLRepository) FindByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.findError != nil {
		return nil, m.findError
	}
	u, exists := m.urls[shortCode]
	if !exists {
		return nil, url.ErrURLNotFound
	}
	return u, nil
}

func (m *mockURLRepository) Delete(ctx context.Context, shortCode string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.urls[shortCode]; !exists {
		return url.ErrURLNotFound
	}
	delete(m.urls, shortCode)
	return nil
}

func (m *mockURLRepository) List(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
	return []*url.URL{}, nil
}

func (m *mockURLRepository) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	return []*url.URL{}, nil
}

// Mock Click Repository
type mockClickRepository struct {
	clicks      []*click.Click
	mu          sync.Mutex
	recordError error
}

func newMockClickRepository() *mockClickRepository {
	return &mockClickRepository{
		clicks: make([]*click.Click, 0),
	}
}

func (m *mockClickRepository) Record(ctx context.Context, c *click.Click) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.recordError != nil {
		return m.recordError
	}
	m.clicks = append(m.clicks, c)
	return nil
}

func (m *mockClickRepository) GetStatsByURL(ctx context.Context, urlID int64) (*click.Stats, error) {
	return nil, nil
}

func (m *mockClickRepository) GetStatsByURLAndTimeRange(ctx context.Context, urlID int64, startTime, endTime time.Time) (*click.TimeRangeStats, error) {
	return nil, nil
}

func (m *mockClickRepository) GetTotalClickCount(ctx context.Context, urlID int64) (int64, error) {
	return 0, nil
}

func (m *mockClickRepository) GetClicksByCountry(ctx context.Context, urlID int64) (map[string]int64, error) {
	return nil, nil
}

func (m *mockClickRepository) getRecordedClicksCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.clicks)
}

func TestRedirectURLUseCase_Execute_Success(t *testing.T) {
	// Setup
	urlRepo := newMockURLRepository()
	clickRepo := newMockClickRepository()

	testURL := &url.URL{
		ID:          1,
		ShortCode:   "test123",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user1",
	}
	urlRepo.urls["test123"] = testURL

	// Use WaitGroup for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	useCase := NewRedirectURLUseCase(urlRepo, clickRepo).WithClickCallback(func() {
		wg.Done()
	})

	// Execute
	req := RedirectRequest{
		ShortCode: "test123",
		Referrer:  "https://google.com",
		UserAgent: "Mozilla/5.0",
		Country:   "US",
	}

	resp, err := useCase.Execute(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.OriginalURL != "https://example.com" {
		t.Errorf("Expected OriginalURL 'https://example.com', got '%s'", resp.OriginalURL)
	}

	// Wait for async click recording to complete
	wg.Wait()

	if clickRepo.getRecordedClicksCount() != 1 {
		t.Errorf("Expected 1 click to be recorded, got %d", clickRepo.getRecordedClicksCount())
	}

	// Clean up
	useCase.Shutdown()
}

func TestRedirectURLUseCase_Execute_URLNotFound(t *testing.T) {
	// Setup
	urlRepo := newMockURLRepository()
	clickRepo := newMockClickRepository()

	useCase := NewRedirectURLUseCase(urlRepo, clickRepo)
	defer useCase.Shutdown()

	// Execute
	req := RedirectRequest{
		ShortCode: "nonexistent",
		Referrer:  "https://google.com",
		UserAgent: "Mozilla/5.0",
		Country:   "US",
	}

	resp, err := useCase.Execute(context.Background(), req)

	// Assert
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if !errors.Is(err, url.ErrURLNotFound) {
		t.Errorf("Expected ErrURLNotFound, got %v", err)
	}

	if resp != nil {
		t.Errorf("Expected nil response, got %v", resp)
	}

	// No click should be recorded since URL was not found
	if clickRepo.getRecordedClicksCount() != 0 {
		t.Errorf("Expected 0 clicks to be recorded, got %d", clickRepo.getRecordedClicksCount())
	}
}

func TestRedirectURLUseCase_Execute_AsyncClickRecording(t *testing.T) {
	// Setup
	urlRepo := newMockURLRepository()
	clickRepo := newMockClickRepository()

	testURL := &url.URL{
		ID:          2,
		ShortCode:   "async123",
		OriginalURL: "https://test.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user2",
	}
	urlRepo.urls["async123"] = testURL

	// Use WaitGroup for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	useCase := NewRedirectURLUseCase(urlRepo, clickRepo).WithClickCallback(func() {
		wg.Done()
	})

	// Execute
	req := RedirectRequest{
		ShortCode: "async123",
		Referrer:  "https://twitter.com",
		UserAgent: "Chrome/96.0",
		Country:   "GB",
	}

	resp, err := useCase.Execute(context.Background(), req)

	// Assert immediate response
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// Initially, click may not be recorded yet (async operation)
	initialCount := clickRepo.getRecordedClicksCount()

	// Wait for async operation to complete
	wg.Wait()

	finalCount := clickRepo.getRecordedClicksCount()

	// The click should be recorded eventually
	if finalCount != 1 {
		t.Errorf("Expected 1 click to be recorded after async operation, got %d (initial: %d)", finalCount, initialCount)
	}

	// Clean up
	useCase.Shutdown()
}

func TestRedirectURLUseCase_Execute_ClickRecordingFailsButRedirectSucceeds(t *testing.T) {
	// Setup
	urlRepo := newMockURLRepository()
	clickRepo := newMockClickRepository()

	// Configure click repo to fail
	clickRepo.recordError = errors.New("database connection failed")

	testURL := &url.URL{
		ID:          3,
		ShortCode:   "resilient",
		OriginalURL: "https://resilient.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user3",
	}
	urlRepo.urls["resilient"] = testURL

	// Use WaitGroup for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	useCase := NewRedirectURLUseCase(urlRepo, clickRepo).WithClickCallback(func() {
		wg.Done()
	})

	// Execute
	req := RedirectRequest{
		ShortCode: "resilient",
		Referrer:  "https://reddit.com",
		UserAgent: "Safari/15.0",
		Country:   "CA",
	}

	resp, err := useCase.Execute(context.Background(), req)

	// Assert - redirect should still succeed even though click recording fails
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.OriginalURL != "https://resilient.com" {
		t.Errorf("Expected OriginalURL 'https://resilient.com', got '%s'", resp.OriginalURL)
	}

	// Wait for async operation to attempt (and fail)
	wg.Wait()

	// No clicks should be recorded due to the error
	if clickRepo.getRecordedClicksCount() != 0 {
		t.Errorf("Expected 0 clicks to be recorded (due to error), got %d", clickRepo.getRecordedClicksCount())
	}

	// Clean up
	useCase.Shutdown()
}

func TestRedirectURLUseCase_Execute_EmptyCountry(t *testing.T) {
	// Setup
	urlRepo := newMockURLRepository()
	clickRepo := newMockClickRepository()

	testURL := &url.URL{
		ID:          4,
		ShortCode:   "nocountry",
		OriginalURL: "https://nocountry.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user4",
	}
	urlRepo.urls["nocountry"] = testURL

	// Use WaitGroup for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	useCase := NewRedirectURLUseCase(urlRepo, clickRepo).WithClickCallback(func() {
		wg.Done()
	})

	// Execute with empty country
	req := RedirectRequest{
		ShortCode: "nocountry",
		Referrer:  "",
		UserAgent: "Mozilla/5.0",
		Country:   "", // Empty country is valid
	}

	resp, err := useCase.Execute(context.Background(), req)

	// Assert
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// Wait for async click recording
	wg.Wait()

	if clickRepo.getRecordedClicksCount() != 1 {
		t.Errorf("Expected 1 click to be recorded, got %d", clickRepo.getRecordedClicksCount())
	}

	// Clean up
	useCase.Shutdown()
}

func TestRedirectURLUseCase_Execute_MultipleClicks(t *testing.T) {
	// Setup
	urlRepo := newMockURLRepository()
	clickRepo := newMockClickRepository()

	testURL := &url.URL{
		ID:          5,
		ShortCode:   "popular",
		OriginalURL: "https://popular.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user5",
	}
	urlRepo.urls["popular"] = testURL

	// Use WaitGroup for synchronization
	var wg sync.WaitGroup
	wg.Add(5)

	useCase := NewRedirectURLUseCase(urlRepo, clickRepo).WithClickCallback(func() {
		wg.Done()
	})

	// Execute multiple redirects
	for i := 0; i < 5; i++ {
		req := RedirectRequest{
			ShortCode: "popular",
			Referrer:  "https://google.com",
			UserAgent: "Mozilla/5.0",
			Country:   "US",
		}

		resp, err := useCase.Execute(context.Background(), req)

		if err != nil {
			t.Fatalf("Redirect %d: Expected no error, got %v", i+1, err)
		}

		if resp == nil {
			t.Fatalf("Redirect %d: Expected response, got nil", i+1)
		}
	}

	// Wait for all async operations to complete
	wg.Wait()

	if clickRepo.getRecordedClicksCount() != 5 {
		t.Errorf("Expected 5 clicks to be recorded, got %d", clickRepo.getRecordedClicksCount())
	}

	// Clean up
	useCase.Shutdown()
}

func TestRedirectURLUseCase_Shutdown(t *testing.T) {
	// Setup
	urlRepo := newMockURLRepository()
	clickRepo := newMockClickRepository()

	testURL := &url.URL{
		ID:          6,
		ShortCode:   "shutdown",
		OriginalURL: "https://shutdown.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user6",
	}
	urlRepo.urls["shutdown"] = testURL

	// Use WaitGroup for synchronization
	var wg sync.WaitGroup
	wg.Add(1)

	useCase := NewRedirectURLUseCase(urlRepo, clickRepo).WithClickCallback(func() {
		wg.Done()
	})

	// Execute a redirect
	req := RedirectRequest{
		ShortCode: "shutdown",
		Referrer:  "https://google.com",
		UserAgent: "Mozilla/5.0",
		Country:   "US",
	}

	resp, err := useCase.Execute(context.Background(), req)

	// Assert redirect succeeded
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	// Wait for click recording
	wg.Wait()

	if clickRepo.getRecordedClicksCount() != 1 {
		t.Errorf("Expected 1 click to be recorded, got %d", clickRepo.getRecordedClicksCount())
	}

	// Shutdown should complete without hanging
	useCase.Shutdown()

	// After shutdown, no more tasks should be processed
	// (This is just to verify Shutdown completes successfully)
}

func TestRedirectURLUseCase_ShutdownDrainsPendingTasks(t *testing.T) {
	// Setup
	urlRepo := newMockURLRepository()
	// Use slow click repository to simulate processing delay
	clickRepo := newSlowMockClickRepository(10 * time.Millisecond)

	testURL := &url.URL{
		ID:          7,
		ShortCode:   "drain",
		OriginalURL: "https://drain.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user7",
	}
	urlRepo.urls["drain"] = testURL

	// Create use case with workers
	numWorkers := 5
	useCase := NewRedirectURLUseCaseWithWorkers(urlRepo, clickRepo, numWorkers)

	// Enqueue tasks that will fit in the buffer (numWorkers * bufferSizeMultiplier)
	// Use fewer tasks than buffer size to ensure they're all queued
	bufferSize := numWorkers * bufferSizeMultiplier
	numTasks := bufferSize // All tasks should fit in buffer

	for i := 0; i < numTasks; i++ {
		req := RedirectRequest{
			ShortCode: "drain",
			Referrer:  "https://test.com",
			UserAgent: "TestAgent",
			Country:   "US",
		}
		_, err := useCase.Execute(context.Background(), req)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}
	}

	// Immediately shutdown - this should drain all pending tasks
	useCase.Shutdown()

	// After shutdown, all tasks should have been processed
	recordedClicks := clickRepo.getClickCount()
	if recordedClicks != numTasks {
		t.Errorf("Expected %d clicks to be recorded after shutdown, got %d", numTasks, recordedClicks)
	}
}

func TestRedirectURLUseCase_ShutdownRejectsNewTasks(t *testing.T) {
	// Setup
	urlRepo := newMockURLRepository()
	clickRepo := newMockClickRepository()

	testURL := &url.URL{
		ID:          8,
		ShortCode:   "reject",
		OriginalURL: "https://reject.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user8",
	}
	urlRepo.urls["reject"] = testURL

	useCase := NewRedirectURLUseCase(urlRepo, clickRepo)

	// Start shutdown in a goroutine
	go useCase.Shutdown()

	// Give shutdown a moment to signal
	time.Sleep(10 * time.Millisecond)

	// Try to execute a redirect after shutdown
	req := RedirectRequest{
		ShortCode: "reject",
		Referrer:  "https://test.com",
		UserAgent: "TestAgent",
		Country:   "US",
	}

	resp, err := useCase.Execute(context.Background(), req)

	// Redirect should still succeed (returns URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response, got nil")
	}

	if resp.OriginalURL != "https://reject.com" {
		t.Errorf("Expected OriginalURL 'https://reject.com', got '%s'", resp.OriginalURL)
	}

	// Wait a bit for any potential async processing
	time.Sleep(50 * time.Millisecond)

	// No clicks should be recorded since shutdown was in progress
	if clickRepo.getRecordedClicksCount() != 0 {
		t.Errorf("Expected 0 clicks to be recorded after shutdown, got %d", clickRepo.getRecordedClicksCount())
	}
}
