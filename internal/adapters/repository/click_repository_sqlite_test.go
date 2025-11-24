package repository

import (
	"context"
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
	if err := urlRepo.Create(context.Background(), u); err != nil {
		t.Fatalf("failed to create URL: %v", err)
	}

	t.Run("successfully record click", func(t *testing.T) {
		c, err := click.NewClick(u.ID, "https://google.com", "US", "Mozilla/5.0")
		if err != nil {
			t.Fatalf("failed to create click: %v", err)
		}

		err = clickRepo.Record(context.Background(), c)
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

		err = clickRepo.Record(context.Background(), c)
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
	urlRepo.Create(context.Background(), u)

	t.Run("no clicks returns 0", func(t *testing.T) {
		count, err := clickRepo.GetTotalClickCount(context.Background(), u.ID)
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
			clickRepo.Record(context.Background(), c)
		}

		count, err := clickRepo.GetTotalClickCount(context.Background(), u.ID)
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
	urlRepo.Create(context.Background(), u)

	// Record clicks from different countries
	countries := []string{"US", "US", "GB", "US", "CA", "GB"}
	for _, country := range countries {
		c, _ := click.NewClick(u.ID, "", country, "")
		clickRepo.Record(context.Background(), c)
	}

	// Record a click with no country
	c, _ := click.NewClick(u.ID, "", "", "")
	clickRepo.Record(context.Background(), c)

	t.Run("get clicks by country", func(t *testing.T) {
		results, err := clickRepo.GetClicksByCountry(context.Background(), u.ID)
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
	urlRepo.Create(context.Background(), u)

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
		clickRepo.Record(context.Background(), click)
	}

	t.Run("get aggregate stats", func(t *testing.T) {
		stats, err := clickRepo.GetStatsByURL(context.Background(), u.ID)
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
	urlRepo.Create(context.Background(), u)

	now := time.Now()

	// Record clicks at different times
	c1, _ := click.NewClick(u.ID, "https://google.com", "US", "")
	c1.ClickedAt = now.Add(-2 * time.Hour)
	clickRepo.Record(context.Background(), c1)

	c2, _ := click.NewClick(u.ID, "https://google.com", "GB", "")
	c2.ClickedAt = now.Add(-1 * time.Hour)
	clickRepo.Record(context.Background(), c2)

	c3, _ := click.NewClick(u.ID, "https://facebook.com", "US", "")
	c3.ClickedAt = now.Add(-30 * time.Minute)
	clickRepo.Record(context.Background(), c3)

	t.Run("get stats in time range", func(t *testing.T) {
		startTime := now.Add(-90 * time.Minute)
		endTime := now

		stats, err := clickRepo.GetStatsByURLAndTimeRange(context.Background(), u.ID, startTime, endTime)
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

		stats, err := clickRepo.GetStatsByURLAndTimeRange(context.Background(), u.ID, startTime, endTime)
		if err != nil {
			t.Fatalf("GetStatsByURLAndTimeRange() error = %v", err)
		}

		if stats.TotalCount != 0 {
			t.Errorf("GetStatsByURLAndTimeRange() TotalCount = %d, want 0", stats.TotalCount)
		}
	})
}

func TestSQLiteClickRepository_ReferrerDomainExtraction(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	urlRepo := NewSQLiteURLRepository(db)
	clickRepo := NewSQLiteClickRepository(db)

	// Create a URL
	u, _ := url.NewURL("test", "https://example.com", "testuser")
	urlRepo.Create(context.Background(), u)

	t.Run("referrer domain is stored correctly", func(t *testing.T) {
		testCases := []struct {
			referrer       string
			expectedDomain string
		}{
			{"https://google.com/search", "google.com"},
			{"https://www.reddit.com/r/golang", "www.reddit.com"},
			{"http://example.com", "example.com"},
			{"", ""}, // Direct navigation
			{"malformed-url", ""}, // Malformed URL
		}

		for _, tc := range testCases {
			c, err := click.NewClick(u.ID, tc.referrer, "US", "Mozilla/5.0")
			if err != nil {
				t.Fatalf("NewClick() error = %v", err)
			}

			if c.ReferrerDomain != tc.expectedDomain {
				t.Errorf("NewClick() ReferrerDomain = %q, want %q for referrer %q",
					c.ReferrerDomain, tc.expectedDomain, tc.referrer)
			}

			err = clickRepo.Record(context.Background(), c)
			if err != nil {
				t.Fatalf("Record() error = %v", err)
			}

			if c.ID == 0 {
				t.Error("Record() should set ID")
			}
		}
	})

	t.Run("stats aggregate by full referrer URL", func(t *testing.T) {
		// Record multiple clicks with different paths but same domain
		clicks := []string{
			"https://twitter.com/user1/status/123",
			"https://twitter.com/user2/status/456",
			"https://twitter.com/search?q=golang",
		}

		for _, ref := range clicks {
			c, _ := click.NewClick(u.ID, ref, "US", "Mozilla/5.0")
			clickRepo.Record(context.Background(), c)
		}

		stats, err := clickRepo.GetStatsByURL(context.Background(), u.ID)
		if err != nil {
			t.Fatalf("GetStatsByURL() error = %v", err)
		}

		// Verify that we get stats for each unique full referrer URL
		// (not aggregated by domain)
		if stats.ByReferrer["https://twitter.com/user1/status/123"] != 1 {
			t.Errorf("Expected 1 click from twitter.com/user1/status/123")
		}
		if stats.ByReferrer["https://twitter.com/user2/status/456"] != 1 {
			t.Errorf("Expected 1 click from twitter.com/user2/status/456")
		}
		if stats.ByReferrer["https://twitter.com/search?q=golang"] != 1 {
			t.Errorf("Expected 1 click from twitter.com/search?q=golang")
		}
	})
}

func TestSQLiteClickRepository_Top10ReferrersLimit(t *testing.T) {
	db, cleanup := setupSQLiteTestDB(t)
	defer cleanup()

	urlRepo := NewSQLiteURLRepository(db)
	clickRepo := NewSQLiteClickRepository(db)

	// Create a URL
	u, _ := url.NewURL("test", "https://example.com", "testuser")
	urlRepo.Create(context.Background(), u)

	t.Run("returns only top 10 referrers", func(t *testing.T) {
		// Create 15 different referrers with different click counts
		referrers := []struct {
			url   string
			count int
		}{
			{"https://google.com", 15},
			{"https://facebook.com", 14},
			{"https://twitter.com", 13},
			{"https://reddit.com", 12},
			{"https://linkedin.com", 11},
			{"https://instagram.com", 10},
			{"https://tiktok.com", 9},
			{"https://youtube.com", 8},
			{"https://pinterest.com", 7},
			{"https://tumblr.com", 6},
			{"https://snapchat.com", 5}, // Should not appear (rank 11)
			{"https://whatsapp.com", 4},  // Should not appear (rank 12)
			{"https://telegram.com", 3},  // Should not appear (rank 13)
			{"https://discord.com", 2},   // Should not appear (rank 14)
			{"https://slack.com", 1},     // Should not appear (rank 15)
		}

		// Record clicks
		for _, ref := range referrers {
			for i := 0; i < ref.count; i++ {
				c, _ := click.NewClick(u.ID, ref.url, "US", "Mozilla/5.0")
				clickRepo.Record(context.Background(), c)
			}
		}

		stats, err := clickRepo.GetStatsByURL(context.Background(), u.ID)
		if err != nil {
			t.Fatalf("GetStatsByURL() error = %v", err)
		}

		// Should only return top 10 referrers
		if len(stats.ByReferrer) != 10 {
			t.Errorf("GetStatsByURL() returned %d referrers, want 10", len(stats.ByReferrer))
		}

		// Verify top 10 are present
		expectedTop10 := []string{
			"https://google.com",
			"https://facebook.com",
			"https://twitter.com",
			"https://reddit.com",
			"https://linkedin.com",
			"https://instagram.com",
			"https://tiktok.com",
			"https://youtube.com",
			"https://pinterest.com",
			"https://tumblr.com",
		}

		for _, ref := range expectedTop10 {
			if _, exists := stats.ByReferrer[ref]; !exists {
				t.Errorf("Expected referrer %s to be in top 10", ref)
			}
		}

		// Verify bottom 5 are not present
		notExpected := []string{
			"https://snapchat.com",
			"https://whatsapp.com",
			"https://telegram.com",
			"https://discord.com",
			"https://slack.com",
		}

		for _, ref := range notExpected {
			if _, exists := stats.ByReferrer[ref]; exists {
				t.Errorf("Expected referrer %s to NOT be in top 10", ref)
			}
		}
	})
}
