package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// Mock URL Repository
type mockURLRepoForAnalytics struct {
	findByShortCodeFunc func(ctx context.Context, shortCode string) (*url.URL, error)
}

func (m *mockURLRepoForAnalytics) Create(ctx context.Context, url *url.URL) error {
	return nil
}

func (m *mockURLRepoForAnalytics) FindByShortCode(ctx context.Context, shortCode string) (*url.URL, error) {
	if m.findByShortCodeFunc != nil {
		return m.findByShortCodeFunc(ctx, shortCode)
	}
	return nil, errors.New("not implemented")
}

func (m *mockURLRepoForAnalytics) Delete(ctx context.Context, shortCode string) error {
	return nil
}

func (m *mockURLRepoForAnalytics) List(ctx context.Context, createdBy string, limit, offset int) ([]*url.URL, error) {
	return nil, nil
}

func (m *mockURLRepoForAnalytics) ListByCreatedByAndTimeRange(ctx context.Context, createdBy string, startTime, endTime time.Time) ([]*url.URL, error) {
	return nil, nil
}

// Mock Click Repository
type mockClickRepoForAnalytics struct {
	getStatsByURLFunc             func(ctx context.Context, urlID int64) (*click.Stats, error)
	getStatsByURLAndTimeRangeFunc func(ctx context.Context, urlID int64, startTime, endTime time.Time) (*click.TimeRangeStats, error)
	getTotalClickCountFunc        func(ctx context.Context, urlID int64) (int64, error)
	getClicksByCountryFunc        func(ctx context.Context, urlID int64) (map[string]int64, error)
}

func (m *mockClickRepoForAnalytics) Record(ctx context.Context, c *click.Click) error {
	return nil
}

func (m *mockClickRepoForAnalytics) GetStatsByURL(ctx context.Context, urlID int64) (*click.Stats, error) {
	if m.getStatsByURLFunc != nil {
		return m.getStatsByURLFunc(ctx, urlID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClickRepoForAnalytics) GetStatsByURLAndTimeRange(ctx context.Context, urlID int64, startTime, endTime time.Time) (*click.TimeRangeStats, error) {
	if m.getStatsByURLAndTimeRangeFunc != nil {
		return m.getStatsByURLAndTimeRangeFunc(ctx, urlID, startTime, endTime)
	}
	return nil, errors.New("not implemented")
}

func (m *mockClickRepoForAnalytics) GetTotalClickCount(ctx context.Context, urlID int64) (int64, error) {
	if m.getTotalClickCountFunc != nil {
		return m.getTotalClickCountFunc(ctx, urlID)
	}
	return 0, errors.New("not implemented")
}

func (m *mockClickRepoForAnalytics) GetClicksByCountry(ctx context.Context, urlID int64) (map[string]int64, error) {
	if m.getClicksByCountryFunc != nil {
		return m.getClicksByCountryFunc(ctx, urlID)
	}
	return nil, errors.New("not implemented")
}

func TestGetAnalyticsUseCase_Execute_AllTimeStats(t *testing.T) {
	ctx := context.Background()

	testURL := &url.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user1",
	}

	stats := &click.Stats{
		URLID:      1,
		TotalCount: 150,
		ByCountry: map[string]int64{
			"US": 100,
			"UK": 50,
		},
		ByReferrer: map[string]int64{
			"https://google.com":  80,
			"https://twitter.com": 70,
		},
		ByDate: map[string]int64{
			"2025-11-20": 50,
			"2025-11-21": 60,
			"2025-11-22": 40,
		},
	}

	urlRepo := &mockURLRepoForAnalytics{
		findByShortCodeFunc: func(ctx context.Context, shortCode string) (*url.URL, error) {
			if shortCode == "abc123" {
				return testURL, nil
			}
			return nil, url.ErrURLNotFound
		},
	}

	clickRepo := &mockClickRepoForAnalytics{
		getStatsByURLFunc: func(ctx context.Context, urlID int64) (*click.Stats, error) {
			if urlID == 1 {
				return stats, nil
			}
			return nil, errors.New("URL not found")
		},
	}

	useCase := NewGetAnalyticsUseCase(urlRepo, clickRepo)

	t.Run("successfully get all-time analytics", func(t *testing.T) {
		resp, err := useCase.Execute(ctx, GetAnalyticsRequest{
			ShortCode:   "abc123",
			RequestedBy: "user1",
		})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if resp.ShortCode != "abc123" {
			t.Errorf("expected short_code abc123, got %s", resp.ShortCode)
		}

		if resp.OriginalURL != "https://example.com" {
			t.Errorf("expected original_url https://example.com, got %s", resp.OriginalURL)
		}

		if resp.TotalClicks != 150 {
			t.Errorf("expected 150 total clicks, got %d", resp.TotalClicks)
		}

		if resp.ByCountry["US"] != 100 {
			t.Errorf("expected 100 US clicks, got %d", resp.ByCountry["US"])
		}

		if resp.ByCountry["UK"] != 50 {
			t.Errorf("expected 50 UK clicks, got %d", resp.ByCountry["UK"])
		}

		if resp.ByDate == nil {
			t.Error("expected by_date to be present for all-time stats")
		}

		if resp.StartTime != nil || resp.EndTime != nil {
			t.Error("expected start_time and end_time to be nil for all-time stats")
		}
	})

	t.Run("URL not found", func(t *testing.T) {
		_, err := useCase.Execute(ctx, GetAnalyticsRequest{
			ShortCode:   "notfound",
			RequestedBy: "user1",
		})

		if !errors.Is(err, url.ErrURLNotFound) {
			t.Errorf("expected ErrURLNotFound, got %v", err)
		}
	})

	t.Run("empty short code", func(t *testing.T) {
		_, err := useCase.Execute(ctx, GetAnalyticsRequest{
			ShortCode:   "",
			RequestedBy: "user1",
		})

		if !errors.Is(err, url.ErrEmptyShortCode) {
			t.Errorf("expected ErrEmptyShortCode, got %v", err)
		}
	})

	t.Run("unauthorized access", func(t *testing.T) {
		_, err := useCase.Execute(ctx, GetAnalyticsRequest{
			ShortCode:   "abc123",
			RequestedBy: "user2", // Different user
		})

		if !errors.Is(err, url.ErrUnauthorizedDeletion) {
			t.Errorf("expected ErrUnauthorizedDeletion, got %v", err)
		}
	})

	t.Run("empty requested by", func(t *testing.T) {
		_, err := useCase.Execute(ctx, GetAnalyticsRequest{
			ShortCode:   "abc123",
			RequestedBy: "",
		})

		if !errors.Is(err, url.ErrInvalidCreatedBy) {
			t.Errorf("expected ErrInvalidCreatedBy, got %v", err)
		}
	})
}

func TestGetAnalyticsUseCase_Execute_TimeRangeStats(t *testing.T) {
	ctx := context.Background()

	testURL := &url.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		CreatedAt:   time.Now(),
		CreatedBy:   "user1",
	}

	startTime := time.Date(2025, 11, 20, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 11, 22, 23, 59, 59, 0, time.UTC)

	timeRangeStats := &click.TimeRangeStats{
		URLID:      1,
		StartTime:  startTime,
		EndTime:    endTime,
		TotalCount: 100,
		ByCountry: map[string]int64{
			"US": 70,
			"UK": 30,
		},
		ByReferrer: map[string]int64{
			"https://google.com":  60,
			"https://twitter.com": 40,
		},
	}

	urlRepo := &mockURLRepoForAnalytics{
		findByShortCodeFunc: func(ctx context.Context, shortCode string) (*url.URL, error) {
			if shortCode == "abc123" {
				return testURL, nil
			}
			return nil, url.ErrURLNotFound
		},
	}

	clickRepo := &mockClickRepoForAnalytics{
		getStatsByURLAndTimeRangeFunc: func(ctx context.Context, urlID int64, start, end time.Time) (*click.TimeRangeStats, error) {
			if urlID == 1 {
				return timeRangeStats, nil
			}
			return nil, errors.New("URL not found")
		},
	}

	useCase := NewGetAnalyticsUseCase(urlRepo, clickRepo)

	t.Run("successfully get time range analytics", func(t *testing.T) {
		resp, err := useCase.Execute(ctx, GetAnalyticsRequest{
			ShortCode:   "abc123",
			RequestedBy: "user1",
			StartTime:   &startTime,
			EndTime:     &endTime,
		})

		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if resp.ShortCode != "abc123" {
			t.Errorf("expected short_code abc123, got %s", resp.ShortCode)
		}

		if resp.TotalClicks != 100 {
			t.Errorf("expected 100 total clicks, got %d", resp.TotalClicks)
		}

		if resp.ByCountry["US"] != 70 {
			t.Errorf("expected 70 US clicks, got %d", resp.ByCountry["US"])
		}

		if resp.ByDate != nil {
			t.Error("expected by_date to be nil for time range stats")
		}

		if resp.StartTime == nil || resp.EndTime == nil {
			t.Error("expected start_time and end_time to be present for time range stats")
		}

		if !resp.StartTime.Equal(startTime) {
			t.Errorf("expected start_time %v, got %v", startTime, resp.StartTime)
		}

		if !resp.EndTime.Equal(endTime) {
			t.Errorf("expected end_time %v, got %v", endTime, resp.EndTime)
		}
	})
}
