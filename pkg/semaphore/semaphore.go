package semaphore

import "time"

// Semaphore implements a classic counting semaphore pattern for limiting concurrency.
type Semaphore struct {
	sem chan struct{}
}

// NewSemaphore creates a new Semaphore with the specified maximum concurrency limit.
// The max parameter determines how many concurrent Acquire operations can succeed.
func NewSemaphore(max int) *Semaphore {
	return &Semaphore{
		sem: make(chan struct{}, max),
	}
}

// Acquire blocks until a semaphore permit is available.
// If the semaphore is at max capacity, it will wait until Release is called.
func (s *Semaphore) Acquire() {
	s.sem <- struct{}{} // Send to channel (takes a slot)
}

// Release frees a semaphore permit, allowing another Acquire to succeed.
// It must be called after Acquire to prevent deadlocks.
func (s *Semaphore) Release() {
	<-s.sem // Receive from channel (frees a slot)
}

// TryAcquire attempts to acquire a permit within the specified timeout.
// Returns true if the permit was acquired, false if the timeout elapsed.
func (s *Semaphore) TryAcquire(timeout time.Duration) bool {
	select {
	case s.sem <- struct{}{}:
		return true
	case <-time.After(timeout):
		return false
	}
}

// Available returns the number of permits currently available (not in use).
// This is calculated as (total capacity) - (currently used permits).
func (s *Semaphore) Available() int {
	return cap(s.sem) - len(s.sem)
}

// Used returns the number of permits currently in use.
// This is equivalent to the number of active Acquire operations not yet Released.
func (s *Semaphore) Used() int {
	return len(s.sem)
}
