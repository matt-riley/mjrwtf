package application

import (
	"context"
	"sync"

	"github.com/matt-riley/mjrwtf/internal/domain/click"
	"github.com/matt-riley/mjrwtf/internal/domain/url"
	"github.com/matt-riley/mjrwtf/internal/infrastructure/metrics"
	"github.com/rs/zerolog"
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
	urlRepo       url.Repository
	clickRepo     click.Repository
	clickTaskChan chan clickRecordTask
	done          chan struct{}

	workersWg    sync.WaitGroup
	callbackMu   sync.RWMutex
	submitMu     sync.RWMutex
	maxWorkers   int
	queueSize    int
	logger       zerolog.Logger
	metrics      *metrics.Metrics
	shutdownOnce sync.Once

	onClickRecorded func()
}

// RedirectURLOptions configures RedirectURLUseCase.
type RedirectURLOptions struct {
	MaxWorkers int
	QueueSize  int
	Logger     *zerolog.Logger
	Metrics    *metrics.Metrics
}

// NewRedirectURLUseCase creates a new RedirectURLUseCase with bounded concurrency for click recording.
func NewRedirectURLUseCase(urlRepo url.Repository, clickRepo click.Repository) *RedirectURLUseCase {
	return NewRedirectURLUseCaseWithWorkers(urlRepo, clickRepo, DefaultMaxWorkers)
}

// NewRedirectURLUseCaseWithWorkers creates a new RedirectURLUseCase with custom worker count.
func NewRedirectURLUseCaseWithWorkers(urlRepo url.Repository, clickRepo click.Repository, maxWorkers int) *RedirectURLUseCase {
	return NewRedirectURLUseCaseWithOptions(urlRepo, clickRepo, RedirectURLOptions{MaxWorkers: maxWorkers})
}

// NewRedirectURLUseCaseWithOptions creates a new RedirectURLUseCase with custom configuration.
func NewRedirectURLUseCaseWithOptions(urlRepo url.Repository, clickRepo click.Repository, opts RedirectURLOptions) *RedirectURLUseCase {
	maxWorkers := opts.MaxWorkers
	if maxWorkers <= 0 {
		maxWorkers = DefaultMaxWorkers
	}

	queueSize := opts.QueueSize
	if queueSize <= 0 {
		queueSize = maxWorkers * bufferSizeMultiplier
	}

	logger := zerolog.Nop()
	if opts.Logger != nil {
		logger = *opts.Logger
	}

	uc := &RedirectURLUseCase{
		urlRepo:       urlRepo,
		clickRepo:     clickRepo,
		clickTaskChan: make(chan clickRecordTask, queueSize),
		done:          make(chan struct{}),
		maxWorkers:    maxWorkers,
		queueSize:     queueSize,
		logger:        logger,
		metrics:       opts.Metrics,
	}

	uc.updateQueueDepth()

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
		uc.submitMu.Lock()
		close(uc.done)
		close(uc.clickTaskChan)
		uc.submitMu.Unlock()

		uc.workersWg.Wait()
		uc.updateQueueDepth()
	})
}

// clickRecordWorker processes click recording tasks from the channel.
// It drains all tasks from clickTaskChan before exiting, ensuring no queued tasks are lost during shutdown.
func (uc *RedirectURLUseCase) clickRecordWorker() {
	defer uc.workersWg.Done()

	for task := range uc.clickTaskChan {
		uc.callbackMu.RLock()
		cb := uc.onClickRecorded
		uc.callbackMu.RUnlock()

		bgCtx := context.Background()

		uc.updateQueueDepth()

		newClick, err := click.NewClick(task.urlID, task.referrer, task.country, task.userAgent)
		if err != nil {
			uc.recordFailure(err, "failed to create click entity")
			if cb != nil {
				cb()
			}
			continue
		}

		if err := uc.clickRepo.Record(bgCtx, newClick); err != nil {
			uc.recordFailure(err, "failed to record click")
		}

		if cb != nil {
			cb()
		}
	}
}

// Execute performs the redirect lookup and records analytics asynchronously
func (uc *RedirectURLUseCase) Execute(ctx context.Context, req RedirectRequest) (*RedirectResponse, error) {
	foundURL, err := uc.urlRepo.FindByShortCode(ctx, req.ShortCode)
	if err != nil {
		return nil, err
	}

	uc.enqueueClick(clickRecordTask{
		urlID:     foundURL.ID,
		shortCode: req.ShortCode,
		referrer:  req.Referrer,
		country:   req.Country,
		userAgent: req.UserAgent,
	})

	return &RedirectResponse{OriginalURL: foundURL.OriginalURL}, nil
}

func (uc *RedirectURLUseCase) enqueueClick(task clickRecordTask) {
	uc.submitMu.RLock()
	defer uc.submitMu.RUnlock()

	select {
	case <-uc.done:
		uc.dropTask("shutdown in progress")
		return
	default:
	}

	select {
	case uc.clickTaskChan <- task:
		uc.updateQueueDepth()
	default:
		uc.dropTask("queue full")
	}
}

func (uc *RedirectURLUseCase) updateQueueDepth() {
	if uc.metrics == nil || uc.metrics.RedirectClickQueueDepth == nil {
		return
	}
	uc.metrics.RedirectClickQueueDepth.Set(float64(len(uc.clickTaskChan)))
}

func (uc *RedirectURLUseCase) dropTask(reason string) {
	if uc.metrics != nil && uc.metrics.RedirectClickDroppedTotal != nil {
		uc.metrics.RedirectClickDroppedTotal.Inc()
	}
	uc.updateQueueDepth()
	uc.logger.Warn().
		Str("reason", reason).
		Int("queue_size", uc.queueSize).
		Int("queue_depth", len(uc.clickTaskChan)).
		Msg("dropping redirect click analytics")
}

func (uc *RedirectURLUseCase) recordFailure(err error, msg string) {
	if uc.metrics != nil && uc.metrics.RedirectClickRecordFailuresTotal != nil {
		uc.metrics.RedirectClickRecordFailuresTotal.Inc()
	}
	uc.logger.Error().Err(err).Msg(msg)
}
