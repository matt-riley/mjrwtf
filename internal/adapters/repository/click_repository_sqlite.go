package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository/sqlc/sqlite"
	"github.com/matt-riley/mjrwtf/internal/domain/click"
)

// SQLiteClickRepository implements the Click repository for SQLite
type SQLiteClickRepository struct {
	clickRepositoryBase
	queries *sqliterepo.Queries
}

// NewSQLiteClickRepository creates a new SQLite Click repository
func NewSQLiteClickRepository(db *sql.DB) *SQLiteClickRepository {
	return &SQLiteClickRepository{
		clickRepositoryBase: clickRepositoryBase{db: db},
		queries:             sqliterepo.New(db),
	}
}

// Record records a new click event
func (r *SQLiteClickRepository) Record(c *click.Click) error {
	ctx := context.Background()

	result, err := r.queries.RecordClick(ctx, sqliterepo.RecordClickParams{
		UrlID:     c.URLID,
		ClickedAt: c.ClickedAt,
		Referrer:  stringToStringPtr(c.Referrer),
		Country:   stringToStringPtr(c.Country),
		UserAgent: stringToStringPtr(c.UserAgent),
	})

	if err != nil {
		return mapClickSQLError(err)
	}

	c.ID = result.ID
	return nil
}

// GetStatsByURL retrieves aggregate statistics for a specific URL
func (r *SQLiteClickRepository) GetStatsByURL(urlID int64) (*click.Stats, error) {
	ctx := context.Background()

	// Get total count
	totalCount, err := r.queries.GetTotalClickCount(ctx, urlID)
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	// Get clicks by country
	countryRows, err := r.queries.GetClicksByCountry(ctx, urlID)
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byCountry := make(map[string]int64)
	for _, row := range countryRows {
		if row.Country != nil {
			byCountry[*row.Country] = row.Count
		}
	}

	// Get clicks by referrer
	referrerRows, err := r.queries.GetClicksByReferrer(ctx, urlID)
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byReferrer := make(map[string]int64)
	for _, row := range referrerRows {
		if row.Referrer != nil {
			byReferrer[*row.Referrer] = row.Count
		}
	}

	// Get clicks by date
	dateRows, err := r.queries.GetClicksByDate(ctx, urlID)
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byDate := make(map[string]int64)
	for _, row := range dateRows {
		if row.Date != nil {
			// SQLite returns interface{} for DATE() which needs to be converted to string
			if dateStr, ok := row.Date.(string); ok {
				byDate[dateStr] = row.Count
			}
		}
	}

	return &click.Stats{
		URLID:      urlID,
		TotalCount: totalCount,
		ByCountry:  byCountry,
		ByReferrer: byReferrer,
		ByDate:     byDate,
	}, nil
}

// GetStatsByURLAndTimeRange retrieves statistics for a URL within a time range
func (r *SQLiteClickRepository) GetStatsByURLAndTimeRange(urlID int64, startTime, endTime time.Time) (*click.TimeRangeStats, error) {
	ctx := context.Background()

	// Get total count in time range
	totalCount, err := r.queries.GetTotalClickCountInTimeRange(ctx, sqliterepo.GetTotalClickCountInTimeRangeParams{
		UrlID:     urlID,
		ClickedAt: startTime,
		ClickedAt_2: endTime,
	})
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	// Get clicks by country in time range
	countryRows, err := r.queries.GetClicksByCountryInTimeRange(ctx, sqliterepo.GetClicksByCountryInTimeRangeParams{
		UrlID:     urlID,
		ClickedAt: startTime,
		ClickedAt_2: endTime,
	})
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byCountry := make(map[string]int64)
	for _, row := range countryRows {
		if row.Country != nil {
			byCountry[*row.Country] = row.Count
		}
	}

	// Get clicks by referrer in time range
	referrerRows, err := r.queries.GetClicksByReferrerInTimeRange(ctx, sqliterepo.GetClicksByReferrerInTimeRangeParams{
		UrlID:     urlID,
		ClickedAt: startTime,
		ClickedAt_2: endTime,
	})
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byReferrer := make(map[string]int64)
	for _, row := range referrerRows {
		if row.Referrer != nil {
			byReferrer[*row.Referrer] = row.Count
		}
	}

	return &click.TimeRangeStats{
		URLID:      urlID,
		StartTime:  startTime,
		EndTime:    endTime,
		TotalCount: totalCount,
		ByCountry:  byCountry,
		ByReferrer: byReferrer,
	}, nil
}

// GetTotalClickCount returns the total number of clicks for a URL
func (r *SQLiteClickRepository) GetTotalClickCount(urlID int64) (int64, error) {
	ctx := context.Background()

	count, err := r.queries.GetTotalClickCount(ctx, urlID)
	if err != nil {
		return 0, mapClickSQLError(err)
	}

	return count, nil
}

// GetClicksByCountry returns click counts grouped by country for a URL
func (r *SQLiteClickRepository) GetClicksByCountry(urlID int64) (map[string]int64, error) {
	ctx := context.Background()

	rows, err := r.queries.GetClicksByCountry(ctx, urlID)
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	result := make(map[string]int64)
	for _, row := range rows {
		if row.Country != nil {
			result[*row.Country] = row.Count
		}
	}

	return result, nil
}
