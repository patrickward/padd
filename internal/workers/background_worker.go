package workers

import (
	"context"
	"log"
	"sync"
	"time"
)

// BackgroundTask represents a background task that can be cancelled
type BackgroundTask struct {
	Name     string
	Handler  func(ctx context.Context) error
	Interval time.Duration // For periodic tasks, 0 means run once
}

// BackgroundWorker manages and runs background tasks with graceful shutdown
type BackgroundWorker struct {
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	tasks  []BackgroundTask
	mu     sync.Mutex
}

// NewBackgroundWorker creates a new BackgroundWorker
func NewBackgroundWorker(ctx context.Context) *BackgroundWorker {
	cctx, cancel := context.WithCancel(ctx)
	return &BackgroundWorker{
		ctx:    cctx,
		cancel: cancel,
		tasks:  make([]BackgroundTask, 0),
	}
}

// AddTask adds a new background task to the runner
func (br *BackgroundWorker) AddTask(task BackgroundTask) {
	br.mu.Lock()
	defer br.mu.Unlock()
	br.tasks = append(br.tasks, task)
}

// AddPeriodicTask adds a new periodic background task to the runner
func (br *BackgroundWorker) AddPeriodicTask(name string, interval time.Duration, handler func(ctx context.Context) error) {
	task := BackgroundTask{
		Name:     name,
		Handler:  handler,
		Interval: interval,
	}

	br.startTask(task) // Start immediately
}

// StartOneTimeTask adds a new one-time background task to the runner and starts it immediately
func (br *BackgroundWorker) StartOneTimeTask(name string, handler func(ctx context.Context) error) {
	br.AddTask(BackgroundTask{
		Name:    name,
		Handler: handler,
		// Interval is 0 for one-time tasks
	})
}

// Start begins executing all added background tasks
func (br *BackgroundWorker) Start() {
	br.mu.Lock()
	defer br.mu.Unlock()

	for _, task := range br.tasks {
		br.startTask(task)
	}
}

// startTask starts a single background task
func (br *BackgroundWorker) startTask(task BackgroundTask) {
	br.wg.Add(1)
	go func(t BackgroundTask) {
		defer br.wg.Done()

		defer func() {
			if r := recover(); r != nil {
				log.Printf("Recovered from panic in task %s: %v", t.Name, r)
			}
		}()

		if t.Interval > 0 {
			ticker := time.NewTicker(t.Interval)
			defer ticker.Stop()

			// Run once immediately
			if err := t.Handler(br.ctx); err != nil {
				log.Printf("Error in task %s: %v", t.Name, err)
			}

			for {
				select {
				case <-br.ctx.Done():
					log.Printf("Background task '%s' stopping", t.Name)
					return
				case <-ticker.C:
					if err := t.Handler(br.ctx); err != nil {
						log.Printf("Background task '%s' error: %v", t.Name, err)
					}
				}
			}
		} else {
			if err := t.Handler(br.ctx); err != nil {
				log.Printf("Background task '%s' error: %v", t.Name, err)
			}
		}
	}(task)
}

// Shutdown gracefully stops all background tasks
func (br *BackgroundWorker) Shutdown() {
	log.Println("Shutting down background tasks...")
	br.cancel()
	br.wg.Wait()
	log.Println("All background tasks stopped.")
}
