package repository

import (
	"testing"
	"time"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

func TestSQLiteClickRepository_Record(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	urlRepo := NewSQLiteURLRepository(db)
	clickRepo := NewSQLiteClickRepository(db)

	// Create a URL to associate clicks with
	u, _ := url.NewURL("test", "https://example.com", "testuser")
	if err := urlRepo.Create(u); err != nil {
		t.Fatalf("failed to create URL: %v", err)
	}

	t.Run("successfully record click", func(t *testing.T) {
		c, err := click.NewClick(u.ID, "https://google.com", "US", "Mozilla/5.0")
		if err != nil {
			t.Fatalf("failed to create click: %v", err)
		}

		err = clickRepo.Record(c)
		if err != nil {
			t.Fatalf("Record() error = %v", err)
		}

		if c.ID == 0 {
			t.Error("Record() should set ID")
		}
	})

	t.Run("record click with empty optional fields", func(t *testing.T) {
		c, err := click.NewClick(u.ID, "", "", "")
		if err != nil {
			t.Fatalf("failed to create click: %v", err)
		}

		err = clickRepo.Record(c)
		if err != nil {
			t.Fatalf("Record() error = %v", err)
		}

		if c.ID == 0 {
			t.Error("Record() should set ID")
		}
	})
}

func TestSQLiteClickRepository_GetTotalClickCount(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	urlRepo := NewSQLiteURLRepository(db)
	clickRepo := NewSQLiteClickRepository(db)

	// Create a URL
	u, _ := url.NewURL("test", "https://example.com", "testuser")
	urlRepo.Create(u)

	t.Run("no clicks returns 0", func(t *testing.T) {
		count, err := clickRepo.GetTotalClickCount(u.ID)
		if err != nil {
			t.Fatalf("GetTotalClickCount() error = %v", err)
		}

		if count != 0 {
			t.Errorf("GetTotalClickCount() = %d, want 0", count)
		}
	})

	t.Run("count multiple clicks", func(t *testing.T) {
		// Record 3 clicks
		for i := 0; i < 3; i++ {
			c, _ := click.NewClick(u.ID, "", "", "")
			clickRepo.Record(c)
		}

		count, err := clickRepo.GetTotalClickCount(u.ID)
		if err != nil {
			t.Fatalf("GetTotalClickCount() error = %v", err)
		}

		if count != 3 {
			t.Errorf("GetTotalClickCount() = %d, want 3", count)
		}
	})
}

func TestSQLiteClickRepository_GetClicksByCountry(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	urlRepo := NewSQLiteURLRepository(db)
	clickRepo := NewSQLiteClickRepository(db)

	// Create a URL
	u, _ := url.NewURL("test", "https://example.com", "testuser")
	urlRepo.Create(u)

	// Record clicks from different countries
	countries := []string{"US", "US", "GB", "US", "CA", "GB"}
	for _, country := range countries {
		c, _ := click.NewClick(u.ID, "", country, "")
		clickRepo.Record(c)
	}

	// Record a click with no country
	c, _ := click.NewClick(u.ID, "", "", "")
	clickRepo.Record(c)

	t.Run("get clicks by country", func(t *testing.T) {
		results, err := clickRepo.GetClicksByCountry(u.ID)
		if err != nil {
			t.Fatalf("GetClicksByCountry() error = %v", err)
		}

		if len(results) != 3 {
			t.Errorf("GetClicksByCountry() returned %d countries, want 3", len(results))
		}

		if results["US"] != 3 {
			t.Errorf("GetClicksByCountry() US count = %d, want 3", results["US"])
		}
		if results["GB"] != 2 {
			t.Errorf("GetClicksByCountry() GB count = %d, want 2", results["GB"])
		}
		if results["CA"] != 1 {
			t.Errorf("GetClicksByCountry() CA count = %d, want 1", results["CA"])
		}
	})
}

func TestSQLiteClickRepository_GetStatsByURL(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	urlRepo := NewSQLiteURLRepository(db)
	clickRepo := NewSQLiteClickRepository(db)

	// Create a URL
	u, _ := url.NewURL("test", "https://example.com", "testuser")
	urlRepo.Create(u)

	// Record clicks with various attributes
	clicks := []struct {
		referrer string
		country  string
	}{
		{"https://google.com", "US"},
		{"https://google.com", "US"},
		{"https://facebook.com", "GB"},
		{"https://google.com", "CA"},
		{"", "US"},
	}

	for _, c := range clicks {
		click, _ := click.NewClick(u.ID, c.referrer, c.country, "Mozilla/5.0")
		clickRepo.Record(click)
	}

	t.Run("get aggregate stats", func(t *testing.T) {
		stats, err := clickRepo.GetStatsByURL(u.ID)
		if err != nil {
			t.Fatalf("GetStatsByURL() error = %v", err)
		}

		if stats.URLID != u.ID {
			t.Errorf("GetStatsByURL() URLID = %d, want %d", stats.URLID, u.ID)
		}

		if stats.TotalCount != 5 {
			t.Errorf("GetStatsByURL() TotalCount = %d, want 5", stats.TotalCount)
		}

		// Check country counts
		if stats.ByCountry["US"] != 3 {
			t.Errorf("GetStatsByURL() ByCountry[US] = %d, want 3", stats.ByCountry["US"])
		}
		if stats.ByCountry["GB"] != 1 {
			t.Errorf("GetStatsByURL() ByCountry[GB] = %d, want 1", stats.ByCountry["GB"])
		}
		if stats.ByCountry["CA"] != 1 {
			t.Errorf("GetStatsByURL() ByCountry[CA] = %d, want 1", stats.ByCountry["CA"])
		}

		// Check referrer counts (empty referrer should not be included)
		if stats.ByReferrer["https://google.com"] != 3 {
			t.Errorf("GetStatsByURL() ByReferrer[google] = %d, want 3", stats.ByReferrer["https://google.com"])
		}
		if stats.ByReferrer["https://facebook.com"] != 1 {
			t.Errorf("GetStatsByURL() ByReferrer[facebook] = %d, want 1", stats.ByReferrer["https://facebook.com"])
		}

		// Check that we have date stats
		if len(stats.ByDate) == 0 {
			t.Error("GetStatsByURL() ByDate should not be empty")
		}
	})
}

func TestSQLiteClickRepository_GetStatsByURLAndTimeRange(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	urlRepo := NewSQLiteURLRepository(db)
	clickRepo := NewSQLiteClickRepository(db)

	// Create a URL
	u, _ := url.NewURL("test", "https://example.com", "testuser")
	urlRepo.Create(u)

	now := time.Now()

	// Record clicks at different times
	c1, _ := click.NewClick(u.ID, "https://google.com", "US", "")
	c1.ClickedAt = now.Add(-2 * time.Hour)
	clickRepo.Record(c1)

	c2, _ := click.NewClick(u.ID, "https://google.com", "GB", "")
	c2.ClickedAt = now.Add(-1 * time.Hour)
	clickRepo.Record(c2)

	c3, _ := click.NewClick(u.ID, "https://facebook.com", "US", "")
	c3.ClickedAt = now.Add(-30 * time.Minute)
	clickRepo.Record(c3)

	t.Run("get stats in time range", func(t *testing.T) {
		startTime := now.Add(-90 * time.Minute)
		endTime := now

		stats, err := clickRepo.GetStatsByURLAndTimeRange(u.ID, startTime, endTime)
		if err != nil {
			t.Fatalf("GetStatsByURLAndTimeRange() error = %v", err)
		}

		if stats.URLID != u.ID {
			t.Errorf("GetStatsByURLAndTimeRange() URLID = %d, want %d", stats.URLID, u.ID)
		}

		// Should only include c2 and c3 (within last 90 minutes)
		if stats.TotalCount != 2 {
			t.Errorf("GetStatsByURLAndTimeRange() TotalCount = %d, want 2", stats.TotalCount)
		}

		if stats.ByCountry["US"] != 1 {
			t.Errorf("GetStatsByURLAndTimeRange() ByCountry[US] = %d, want 1", stats.ByCountry["US"])
		}
		if stats.ByCountry["GB"] != 1 {
			t.Errorf("GetStatsByURLAndTimeRange() ByCountry[GB] = %d, want 1", stats.ByCountry["GB"])
		}

		if stats.ByReferrer["https://google.com"] != 1 {
			t.Errorf("GetStatsByURLAndTimeRange() ByReferrer[google] = %d, want 1", stats.ByReferrer["https://google.com"])
		}
		if stats.ByReferrer["https://facebook.com"] != 1 {
			t.Errorf("GetStatsByURLAndTimeRange() ByReferrer[facebook] = %d, want 1", stats.ByReferrer["https://facebook.com"])
		}
	})

	t.Run("no clicks in time range", func(t *testing.T) {
		startTime := now.Add(1 * time.Hour)
		endTime := now.Add(2 * time.Hour)

		stats, err := clickRepo.GetStatsByURLAndTimeRange(u.ID, startTime, endTime)
		if err != nil {
			t.Fatalf("GetStatsByURLAndTimeRange() error = %v", err)
		}

		if stats.TotalCount != 0 {
			t.Errorf("GetStatsByURLAndTimeRange() TotalCount = %d, want 0", stats.TotalCount)
		}
	})
}
