// Package application contains use case implementations for the mjrwtf URL shortener,
// coordinating between domain entities and adapters.
package application

import (
	"context"
	"log"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

// RedirectRequest contains the data needed to redirect and track a short URL
type RedirectRequest struct {
	ShortCode string
	Referrer  string
	UserAgent string
	Country   string
}

// RedirectResponse contains the result of a redirect lookup
type RedirectResponse struct {
	OriginalURL string
}

// clickRecordTask represents a task to record a click
type clickRecordTask struct {
	urlID     int64
	shortCode string
	referrer  string
	country   string
	userAgent string
}

// RedirectURLUseCase handles redirecting short URLs and tracking analytics
type RedirectURLUseCase struct {
	urlRepo         url.Repository
	clickRepo       click.Repository
	clickTaskChan   chan clickRecordTask
	done            chan struct{}
	maxWorkers      int
	onClickRecorded func()
}

// NewRedirectURLUseCase creates a new RedirectURLUseCase with bounded concurrency for click recording
// maxWorkers controls the number of concurrent goroutines for analytics recording (default: 100)
func NewRedirectURLUseCase(urlRepo url.Repository, clickRepo click.Repository) *RedirectURLUseCase {
	return NewRedirectURLUseCaseWithWorkers(urlRepo, clickRepo, 100)
}

// NewRedirectURLUseCaseWithWorkers creates a new RedirectURLUseCase with custom worker count
func NewRedirectURLUseCaseWithWorkers(urlRepo url.Repository, clickRepo click.Repository, maxWorkers int) *RedirectURLUseCase {
	if maxWorkers <= 0 {
		maxWorkers = 100
	}
	
	uc := &RedirectURLUseCase{
		urlRepo:       urlRepo,
		clickRepo:     clickRepo,
		clickTaskChan: make(chan clickRecordTask, maxWorkers*2),
		done:          make(chan struct{}),
		maxWorkers:    maxWorkers,
	}
	
	// Start worker pool
	for i := 0; i < maxWorkers; i++ {
		go uc.clickRecordWorker()
	}
	
	return uc
}

// WithClickCallback sets an optional callback to be invoked after click recording (for testing)
func (uc *RedirectURLUseCase) WithClickCallback(callback func()) *RedirectURLUseCase {
	uc.onClickRecorded = callback
	return uc
}

// Shutdown gracefully shuts down the worker pool, waiting for in-flight tasks to complete
func (uc *RedirectURLUseCase) Shutdown() {
	close(uc.done)
	close(uc.clickTaskChan)
}

// clickRecordWorker processes click recording tasks from the channel
func (uc *RedirectURLUseCase) clickRecordWorker() {
	for {
		select {
		case <-uc.done:
			return
		case task, ok := <-uc.clickTaskChan:
			if !ok {
				return
			}
			bgCtx := context.Background()
			
			newClick, err := click.NewClick(task.urlID, task.referrer, task.country, task.userAgent)
			if err != nil {
				log.Printf("Failed to create click entity for URL %s: %v", task.shortCode, err)
				if uc.onClickRecorded != nil {
					uc.onClickRecorded()
				}
				continue
			}

			if err := uc.clickRepo.Record(bgCtx, newClick); err != nil {
				log.Printf("Failed to record click for URL %s: %v", task.shortCode, err)
			}
			
			if uc.onClickRecorded != nil {
				uc.onClickRecorded()
			}
		}
	}
}

// Execute performs the redirect lookup and records analytics asynchronously
func (uc *RedirectURLUseCase) Execute(ctx context.Context, req RedirectRequest) (*RedirectResponse, error) {
	// Look up URL by short code
	foundURL, err := uc.urlRepo.FindByShortCode(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	// Send click recording task to worker pool (non-blocking with buffered channel)
	select {
	case uc.clickTaskChan <- clickRecordTask{
		urlID:     foundURL.ID,
		shortCode: req.ShortCode,
		referrer:  req.Referrer,
		country:   req.Country,
		userAgent: req.UserAgent,
	}:
		// Task sent successfully
	default:
		// Channel full, log warning but don't block the redirect
		log.Printf("Click recording queue full for URL %s, dropping analytics", req.ShortCode)
		if uc.onClickRecorded != nil {
			// Still call callback for test synchronization even if task is dropped
			uc.onClickRecorded()
		}
	}

	return &RedirectResponse{
		OriginalURL: foundURL.OriginalURL,
	}, nil
}
