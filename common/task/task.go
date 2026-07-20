package task

import (
	"sync"
	"time"
)

type Task struct {
	Interval time.Duration
	Execute  func() error
	access   sync.Mutex
	running  bool
	stop     chan struct{}
	done     chan struct{} // closed when the worker goroutine exits
}

func (t *Task) Start(first bool) error {
	t.access.Lock()
	if t.running {
		t.access.Unlock()
		return nil
	}
	if t.Interval <= 0 {
		t.access.Unlock()
		return nil
	}
	t.running = true
	t.stop = make(chan struct{})
	t.done = make(chan struct{})
	stop := t.stop
	done := t.done
	t.access.Unlock()

	go func() {
		defer close(done)

		if first {
			if err := t.Execute(); err != nil {
				t.finishFromWorker(stop)
				return
			}
		}

		ticker := time.NewTicker(t.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
			case <-stop:
				return
			}

			if err := t.Execute(); err != nil {
				t.finishFromWorker(stop)
				return
			}
		}
	}()

	return nil
}

func (t *Task) finishFromWorker(stop chan struct{}) {
	t.access.Lock()
	defer t.access.Unlock()
	if !t.running {
		return
	}
	t.running = false
	select {
	case <-stop:
	default:
		close(stop)
	}
}

// Close signals the worker to stop and waits until it has exited.
func (t *Task) Close() {
	t.access.Lock()
	if !t.running {
		done := t.done
		t.access.Unlock()
		if done != nil {
			<-done
		}
		return
	}
	t.running = false
	stop := t.stop
	done := t.done
	select {
	case <-stop:
	default:
		close(stop)
	}
	t.access.Unlock()

	if done != nil {
		<-done
	}
}
