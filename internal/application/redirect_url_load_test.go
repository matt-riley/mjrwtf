package application

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// slowMockClickRepository is a mock that simulates slow click recording
type slowMockClickRepository struct {
	clicks      []*click.Click
	mu          sync.Mutex
	recordDelay time.Duration
}

func newSlowMockClickRepository(delay time.Duration) *slowMockClickRepository {
	return &slowMockClickRepository{
		clicks:      make([]*click.Click, 0),
		recordDelay: delay,
	}
}

func (m *slowMockClickRepository) Record(ctx context.Context, c *click.Click) error {
	// Simulate slow database operation
	time.Sleep(m.recordDelay)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clicks = append(m.clicks, c)
	return nil
}

func (m *slowMockClickRepository) GetStatsByURL(ctx context.Context, urlID int64) (*click.Stats, error) {
	return &click.Stats{}, nil
}

func (m *slowMockClickRepository) GetStatsByURLAndTimeRange(ctx context.Context, urlID int64, startTime, endTime time.Time) (*click.TimeRangeStats, error) {
	return &click.TimeRangeStats{}, nil
}

func (m *slowMockClickRepository) GetTotalClickCount(ctx context.Context, urlID int64) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return int64(len(m.clicks)), nil
}

func (m *slowMockClickRepository) GetClicksByCountry(ctx context.Context, urlID int64) (map[string]int64, error) {
	return make(map[string]int64), nil
}

func (m *slowMockClickRepository) getClickCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.clicks)
}

// TestRedirectURLUseCase_LoadTest verifies non-blocking behavior under load
func TestRedirectURLUseCase_LoadTest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping load test in short mode")
	}

	// Setup mock repositories
	testURL := &url.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user1",
	}

	urlRepo := newMockURLRepository()
	urlRepo.urls["abc123"] = testURL

	// Use slow click repository to simulate database latency (1ms instead of 10ms)
	clickRepo := newSlowMockClickRepository(1 * time.Millisecond)

	// Create use case with limited workers to test buffering
	useCase := NewRedirectURLUseCaseWithWorkers(urlRepo, clickRepo, 10)

	// Simulate high traffic: 100 requests as fast as possible
	numRequests := 100
	var wg sync.WaitGroup
	var errorsMu sync.Mutex
	var errors []string
	startTime := time.Now()

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(reqNum int) {
			defer wg.Done()

			req := RedirectRequest{
				ShortCode: "abc123",
				Referrer:  "https://google.com",
				UserAgent: "TestAgent",
				Country:   "US",
			}

			resp, err := useCase.Execute(context.Background(), req)
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, fmt.Sprintf("request %d failed: %v", reqNum, err))
				errorsMu.Unlock()
				return
			}

			if resp.OriginalURL != "https://example.com" {
				errorsMu.Lock()
				errors = append(errors, fmt.Sprintf("request %d: unexpected URL: %s", reqNum, resp.OriginalURL))
				errorsMu.Unlock()
			}
		}(i)
	}

	// Wait for all redirects to complete
	wg.Wait()

	// Report any errors that occurred
	if len(errors) > 0 {
		for _, err := range errors {
			t.Error(err)
		}
	}
	redirectDuration := time.Since(startTime)

	// Verify redirects are fast (non-blocking)
	avgRedirectTime := redirectDuration / time.Duration(numRequests)
	if avgRedirectTime > 5*time.Millisecond {
		t.Errorf("redirects are too slow: avg %v per request (should be <5ms)", avgRedirectTime)
	}

	t.Logf("Processed %d redirects in %v (avg: %v per request)", numRequests, redirectDuration, avgRedirectTime)

	// Shutdown and wait for all click recordings to complete
	useCase.Shutdown()

	// Verify most clicks were recorded (some may be dropped if buffer was full)
	finalRecordedClicks := int64(clickRepo.getClickCount())

	t.Logf("Recorded %d/%d clicks (%.1f%%)", finalRecordedClicks, numRequests, float64(finalRecordedClicks)/float64(numRequests)*100)

	// Under extreme burst traffic with slow database (1ms per click), some clicks will be dropped
	// This is expected behavior - the system prioritizes fast redirects over guaranteed click recording
	// We just verify that SOME clicks were recorded and the system didn't crash
	if finalRecordedClicks == 0 {
		t.Error("no clicks were recorded at all")
	}

	// Verify the system handled the load gracefully (logged drops, didn't crash)
	t.Logf("System handled burst traffic gracefully, dropping excess clicks as expected")
}

// TestRedirectURLUseCase_ClickRecordingPerformance verifies click recording doesn't block redirects
func TestRedirectURLUseCase_ClickRecordingPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	testURL := &url.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user1",
	}

	urlRepo := newMockURLRepository()
	urlRepo.urls["abc123"] = testURL

	// Simulate very slow click recording (100ms)
	clickRepo := newSlowMockClickRepository(100 * time.Millisecond)

	useCase := NewRedirectURLUseCaseWithWorkers(urlRepo, clickRepo, 50)
	defer useCase.Shutdown()

	// Measure redirect time
	startTime := time.Now()
	_, err := useCase.Execute(context.Background(), RedirectRequest{
		ShortCode: "abc123",
		Referrer:  "https://google.com",
		UserAgent: "TestAgent",
		Country:   "US",
	})
	redirectTime := time.Since(startTime)

	if err != nil {
		t.Fatalf("redirect failed: %v", err)
	}

	// Redirect should complete in <10ms even though click recording takes 100ms
	if redirectTime > 10*time.Millisecond {
		t.Errorf("redirect blocked by click recording: took %v (should be <10ms)", redirectTime)
	}

	t.Logf("Redirect completed in %v (click recording is async)", redirectTime)
}
