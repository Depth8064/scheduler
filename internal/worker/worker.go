package worker

import (
	"context"
	"log"
	"time"
)

// Worker is a minimal execution worker skeleton.
type Worker struct {
	// In a full implementation this would hold stores and configuration.
	stop chan struct{}
}

// NewWorker constructs a worker.
func NewWorker() *Worker {
	return &Worker{stop: make(chan struct{})}
}

// Run starts a simple loop (non-blocking).
func (w *Worker) Run(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("worker: context done")
				return
			case <-w.stop:
				log.Println("worker: stopped")
				return
			case <-ticker.C:
				// placeholder work
			}
		}
	}()
}

// Stop requests the worker to stop.
func (w *Worker) Stop() {
	close(w.stop)
}
