// Package errmux provides functions and data for handling errors in a concurrent environment.
package errmux

import (
	"log"
	"sync"
)

// mergeErrors merges a slice of error channels into one, terminating early if q is unblocked.
func mergeErrors(q <-chan struct{}, errs []<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error, len(errs))

	wg.Add(len(errs))

	// For each error channel, start a separate error handling goroutine.
	for _, e := range errs {
		// Make sure to close over our channel.
		go func(ch <-chan error) {
			defer wg.Done()

			/*
				It's tempting to refactor this block to look like so:

					select {
					case <-q:
						return
					case out <- err:
					}

				This will, however, yield only a 50% chance that our quit channel will
				be read. As written below, there is no guarantee out will be available
				for send, but we know by the contract of Handler that all values on out
				will be read. The current implementation ensures our contracts for Wait
				and Err on Handler are valid.
			*/
			for err := range ch {
				select {
				case <-q:
					return
				default:
				}
				out <- err
			}
		}(e)
	}

	// Wait for all error handlers to finish and close.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// Handler represents a handler for multiple concurrent error streams.
type Handler struct {
	c    Consumer
	errs <-chan error
	q    chan struct{}
	t    chan struct{}
	twg  sync.WaitGroup
}

// NewHandler creates a handler with the provided consumer and error channels.
// All values are guaranteed to be read from any given error channel until it is closed
// or the handler is canceled.
func NewHandler(c Consumer, errs ...<-chan error) *Handler {
	q := make(chan struct{})
	errOut := mergeErrors(q, errs)
	t := make(chan struct{}, 1)

	h := &Handler{
		c:    c,
		errs: errOut,
		q:    q,
		t:    t,
	}

	h.twg.Add(1)
	go h.start()
	go h.waitAndClose()

	return h
}

// start begins error processing, terminating early if the consumer's Consume
// method returns false.
func (h *Handler) start() {
	done := false
	// iterate over the entire range of errors
	for err := range h.errs {
		// ensure we only get a false result from Consume one time
		if !done {
			if ok := h.c.Consume(err); !ok {
				h.Cancel()
				done = true
			}
		}
	}

	h.Cancel()
}

// waitAndClose waits for termination and then closes the Handler.
func (h *Handler) waitAndClose() {
	h.twg.Wait()

	close(h.q)
}

// Wait blocks until error processing is finished. After Wait returns, no more
// values will be read from the handler's error channels.
func (h *Handler) Wait() {
	<-h.q
}

// Err blocks until error processing is finished and then returns the consumer's
// final error before termination. All rules of Wait apply to Err.
func (h *Handler) Err() error {
	h.Wait()

	return h.c.Err()
}

// Cancel terminates error handling. If the handler is already canceled or finished
// handling errors, Cancel returns false.
func (h *Handler) Cancel() bool {
	select {
	case h.t <- struct{}{}:
		h.twg.Done()
		return true
	default:
		log.Printf("unable to cancel")
		return false
	}
}
