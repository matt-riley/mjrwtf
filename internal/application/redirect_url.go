// Package application contains use case implementations for the mjrwtf URL shortener,
// coordinating between domain entities and adapters.
package application

import (
	"context"
	"log"
	"sync"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
)

const (
	// DefaultMaxWorkers is the default number of worker goroutines for async click recording
	DefaultMaxWorkers = 100
	// bufferSizeMultiplier determines the channel buffer size relative to worker count
	bufferSizeMultiplier = 2
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
	workersWg       sync.WaitGroup
	callbackMu      sync.RWMutex
	maxWorkers      int
	onClickRecorded func()
	shutdownOnce    sync.Once
}

// NewRedirectURLUseCase creates a new RedirectURLUseCase with bounded concurrency for click recording
// maxWorkers controls the number of concurrent goroutines for analytics recording (default: DefaultMaxWorkers)
func NewRedirectURLUseCase(urlRepo url.Repository, clickRepo click.Repository) *RedirectURLUseCase {
	return NewRedirectURLUseCaseWithWorkers(urlRepo, clickRepo, DefaultMaxWorkers)
}

// NewRedirectURLUseCaseWithWorkers creates a new RedirectURLUseCase with custom worker count
func NewRedirectURLUseCaseWithWorkers(urlRepo url.Repository, clickRepo click.Repository, maxWorkers int) *RedirectURLUseCase {
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}

	uc := &RedirectURLUseCase{
		urlRepo:   urlRepo,
		clickRepo: clickRepo,
		// Buffer size is bufferSizeMultiplier times the worker count: each worker can have one pending task,
		// plus an equal amount of headroom to reduce blocking during bursts of submissions.
		clickTaskChan: make(chan clickRecordTask, maxWorkers*bufferSizeMultiplier),
		done:          make(chan struct{}),
		maxWorkers:    maxWorkers,
	}

	// Start worker pool
	uc.workersWg.Add(maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		go uc.clickRecordWorker()
	}

	return uc
}

// WithClickCallback sets an optional callback to be invoked after click recording (for testing)
func (uc *RedirectURLUseCase) WithClickCallback(callback func()) *RedirectURLUseCase {
	uc.callbackMu.Lock()
	uc.onClickRecorded = callback
	uc.callbackMu.Unlock()
	return uc
}

// Shutdown gracefully shuts down the worker pool.
// It prevents new task submissions and waits for all queued tasks to be processed.
// Once Shutdown is called:
// - New task submissions via Execute() are rejected (tasks are dropped)
// - All tasks already in clickTaskChan are processed before workers exit
// - The method blocks until all workers have finished processing
func (uc *RedirectURLUseCase) Shutdown() {
	uc.shutdownOnce.Do(func() {
		// Signal shutdown has started - prevents new submissions in Execute()
		close(uc.done)
		// Close the task channel - workers will drain remaining tasks and then exit
		close(uc.clickTaskChan)
		// Wait for all workers to finish processing
		uc.workersWg.Wait()
	})
}

// clickRecordWorker processes click recording tasks from the channel.
// It drains all tasks from clickTaskChan before exiting, ensuring no queued tasks are lost during shutdown.
func (uc *RedirectURLUseCase) clickRecordWorker() {
	defer uc.workersWg.Done()

	// Process tasks until the channel is closed and drained
	for task := range uc.clickTaskChan {
		// Call callback once per task, regardless of success/failure
		uc.callbackMu.RLock()
		cb := uc.onClickRecorded
		uc.callbackMu.RUnlock()

		// Use background context to prevent cancellation from affecting analytics.
		// This is intentional: we want click recording to complete even if the
		// original request context is cancelled, as analytics should not impact
		// the redirect response.
		bgCtx := context.Background()

		newClick, err := click.NewClick(task.urlID, task.referrer, task.country, task.userAgent)
		if err != nil {
			log.Printf("Failed to create click entity for URL %s: %v", task.shortCode, err)
			if cb != nil {
				cb()
			}
			continue
		}

		if err := uc.clickRepo.Record(bgCtx, newClick); err != nil {
			log.Printf("Failed to record click for URL %s: %v", task.shortCode, err)
		}

		if cb != nil {
			cb()
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

	// Check if shutdown has started - if so, don't accept new tasks
	select {
	case <-uc.done:
		// Shutdown in progress - don't queue new analytics tasks
		log.Printf("Shutdown in progress, dropping analytics for URL %s", req.ShortCode)
		return &RedirectResponse{
			OriginalURL: foundURL.OriginalURL,
		}, nil
	default:
		// Shutdown not started, proceed with task submission
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
		// Channel full, analytics data will be lost. Consider adding metrics/monitoring
		// for dropped analytics to detect capacity issues during traffic spikes.
		// Note: callback is NOT invoked here because no task was actually enqueued.
		log.Printf("Click recording queue full for URL %s, dropping analytics", req.ShortCode)
	}

	return &RedirectResponse{
		OriginalURL: foundURL.OriginalURL,
	}, nil
}
