package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/matt-riley/mjrwtf/internal/adapters/repository/sqlc/postgres"
	"github.com/matt-riley/mjrwtf/internal/domain/click"
)

// PostgresClickRepository implements the Click repository for PostgreSQL
type PostgresClickRepository struct {
	clickRepositoryBase
	queries *postgresrepo.Queries
}

// NewPostgresClickRepository creates a new PostgreSQL Click repository
func NewPostgresClickRepository(db *sql.DB) *PostgresClickRepository {
	return &PostgresClickRepository{
		clickRepositoryBase: clickRepositoryBase{db: db},
		queries:             postgresrepo.New(db),
	}
}

// Record records a new click event
func (r *PostgresClickRepository) Record(ctx context.Context, c *click.Click) error {
	result, err := r.queries.RecordClick(ctx, postgresrepo.RecordClickParams{
		UrlID:          int32(c.URLID),
		ClickedAt:      c.ClickedAt,
		Referrer:       stringToNullString(c.Referrer),
		ReferrerDomain: stringToNullString(c.ReferrerDomain),
		Country:        stringToNullString(c.Country),
		UserAgent:      stringToNullString(c.UserAgent),
	})

	if err != nil {
		return mapClickSQLError(err)
	}

	c.ID = int64(result.ID)
	return nil
}

// GetStatsByURL retrieves aggregate statistics for a specific URL
func (r *PostgresClickRepository) GetStatsByURL(ctx context.Context, urlID int64) (*click.Stats, error) {
	// Get total count
	totalCount, err := r.queries.GetTotalClickCount(ctx, int32(urlID))
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	// Get clicks by country
	countryRows, err := r.queries.GetClicksByCountry(ctx, int32(urlID))
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byCountry := make(map[string]int64)
	for _, row := range countryRows {
		if row.Country.Valid {
			byCountry[row.Country.String] = row.Count
		}
	}

	// Get clicks by referrer
	referrerRows, err := r.queries.GetClicksByReferrer(ctx, int32(urlID))
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byReferrer := make(map[string]int64)
	for _, row := range referrerRows {
		if row.Referrer.Valid {
			byReferrer[row.Referrer.String] = row.Count
		}
	}

	// Get clicks by date
	dateRows, err := r.queries.GetClicksByDate(ctx, int32(urlID))
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byDate := make(map[string]int64)
	for _, row := range dateRows {
		// PostgreSQL returns time.Time for DATE() which needs to be formatted
		byDate[row.Date.Format("2006-01-02")] = row.Count
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
func (r *PostgresClickRepository) GetStatsByURLAndTimeRange(ctx context.Context, urlID int64, startTime, endTime time.Time) (*click.TimeRangeStats, error) {
	// Get total count in time range
	totalCount, err := r.queries.GetTotalClickCountInTimeRange(ctx, postgresrepo.GetTotalClickCountInTimeRangeParams{
		UrlID:       int32(urlID),
		ClickedAt:   startTime,
		ClickedAt_2: endTime,
	})
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	// Get clicks by country in time range
	countryRows, err := r.queries.GetClicksByCountryInTimeRange(ctx, postgresrepo.GetClicksByCountryInTimeRangeParams{
		UrlID:       int32(urlID),
		ClickedAt:   startTime,
		ClickedAt_2: endTime,
	})
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byCountry := make(map[string]int64)
	for _, row := range countryRows {
		if row.Country.Valid {
			byCountry[row.Country.String] = row.Count
		}
	}

	// Get clicks by referrer in time range
	referrerRows, err := r.queries.GetClicksByReferrerInTimeRange(ctx, postgresrepo.GetClicksByReferrerInTimeRangeParams{
		UrlID:       int32(urlID),
		ClickedAt:   startTime,
		ClickedAt_2: endTime,
	})
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	byReferrer := make(map[string]int64)
	for _, row := range referrerRows {
		if row.Referrer.Valid {
			byReferrer[row.Referrer.String] = row.Count
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
func (r *PostgresClickRepository) GetTotalClickCount(ctx context.Context, urlID int64) (int64, error) {
	count, err := r.queries.GetTotalClickCount(ctx, int32(urlID))
	if err != nil {
		return 0, mapClickSQLError(err)
	}

	return count, nil
}

// GetClicksByCountry returns click counts grouped by country for a URL
func (r *PostgresClickRepository) GetClicksByCountry(ctx context.Context, urlID int64) (map[string]int64, error) {
	rows, err := r.queries.GetClicksByCountry(ctx, int32(urlID))
	if err != nil {
		return nil, mapClickSQLError(err)
	}

	result := make(map[string]int64)
	for _, row := range rows {
		if row.Country.Valid {
			result[row.Country.String] = row.Count
		}
	}

	return result, nil
}
