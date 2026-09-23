// Package worker implements a concurrent worker pool that pulls tasks off
// a queue.Queue and dispatches them to registered handlers by task type.
package worker

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/YOUR_USERNAME/taskqueue/internal/queue"
)

// Handler processes a single task's payload. Returning an error causes the
// task to be retried (with backoff) or dead-lettered if retries are exhausted.
type Handler func(ctx context.Context, payload []byte) error

// Pool runs N concurrent workers pulling from a shared queue.
type Pool struct {
	q        *queue.Queue
	handlers map[string]Handler
	mu       sync.RWMutex
}

func NewPool(q *queue.Queue) *Pool {
	return &Pool{
		q:        q,
		handlers: make(map[string]Handler),
	}
}

// RegisterHandler associates a task type with a processing function.
// Must be called before Run for that task type to be handled.
func (p *Pool) RegisterHandler(taskType string, h Handler) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handlers[taskType] = h
}

func (p *Pool) handlerFor(taskType string) (Handler, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	h, ok := p.handlers[taskType]
	return h, ok
}

// Run starts `concurrency` worker goroutines and blocks until ctx is
// cancelled, at which point it waits for in-flight tasks to finish.
func (p *Pool) Run(ctx context.Context, concurrency int) {
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		workerID := fmt.Sprintf("worker-%d", i)
		go func() {
			defer wg.Done()
			p.workerLoop(ctx, workerID)
		}()
	}
	wg.Wait()
}

func (p *Pool) workerLoop(ctx context.Context, workerID string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		task, err := p.q.Dequeue(ctx, workerID, 2*time.Second)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("[%s] dequeue error: %v", workerID, err)
			time.Sleep(time.Second)
			continue
		}
		if task == nil {
			continue // timed out waiting, nothing ready — loop again
		}

		handler, ok := p.handlerFor(task.Type)
		if !ok {
			log.Printf("[%s] no handler registered for task type %q, dead-lettering", workerID, task.Type)
			_ = p.q.Nack(ctx, workerID, task, fmt.Errorf("no handler for type %q", task.Type))
			continue
		}

		if err := handler(ctx, task.Payload); err != nil {
			log.Printf("[%s] task %s failed (attempt %d): %v", workerID, task.ID, task.Attempts+1, err)
			if nackErr := p.q.Nack(ctx, workerID, task, err); nackErr != nil {
				log.Printf("[%s] nack error: %v", workerID, nackErr)
			}
			continue
		}

		if err := p.q.Ack(ctx, workerID, task); err != nil {
			log.Printf("[%s] ack error: %v", workerID, err)
		}
	}
}
