package errmux

import (
	"fmt"
	"testing"
)

type mapConsumer map[error]struct{}

func (m mapConsumer) Consume(err error) bool {
	m[err] = struct{}{}

	return true
}

func (m mapConsumer) Err() error {
	return nil
}

func TestRangeAll(t *testing.T) {
	for i := 0; i < 100; i++ {
		testRangeAll(t)
	}
}

func testRangeAll(t *testing.T) {
	n := 3
	errs := make([]<-chan error, n)
	exp := map[error]bool{}
	for i := range errs {
		ch := make(chan error, 1)
		err := fmt.Errorf("%d", i)
		ch <- err
		close(ch)
		errs[i] = ch
		exp[err] = false
	}

	m := mapConsumer{}

	h := NewHandler(m, errs...)
	h.Wait()

	for k := range m {
		exp[k] = true
	}

	for k := range m {
		if ok := exp[k]; !ok {
			t.Fatalf("missing error %s", k)
		}
	}
}

func TestCancel(t *testing.T) {
	for i := 0; i < 100; i++ {
		testCancel(t)
	}
}

func testCancel(t *testing.T) {

	ch1 := make(chan error)
	ch2 := make(chan error)
	h := NewHandler(&DefaultConsumer{}, ch1, ch2)
	go func() {
		ch1 <- nil
		h.Cancel()
	}()

	go func() {
		h.Wait()
		ch2 <- nil
		ch2 <- nil
		t.Fatalf("unexpected send")
	}()

	h.Wait()
}
