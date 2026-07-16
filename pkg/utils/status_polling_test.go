package utils

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type pollingTestChecker struct {
	calls  atomic.Int32
	called chan struct{}
	status string
	err    error
}

func (c *pollingTestChecker) GetResource(context.Context, string) (interface{}, error) {
	c.calls.Add(1)
	select {
	case c.called <- struct{}{}:
	default:
	}
	if c.err != nil {
		return nil, c.err
	}
	return struct{}{}, nil
}

func (c *pollingTestChecker) ExtractStatus(interface{}) string { return c.status }
func (c *pollingTestChecker) GetResourceType() ResourceType    { return "test resource" }

func TestWaitForResourceStatusStopsDuringRetrySleepWhenContextCanceled(t *testing.T) {
	checker := &pollingTestChecker{called: make(chan struct{}, 1), status: "BUILDING"}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- WaitForResourceStatus(ctx, "test", checker)
	}()

	<-checker.called
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WaitForResourceStatus() error = %v, want context.Canceled", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("WaitForResourceStatus did not stop promptly after cancellation")
	}
	if got := checker.calls.Load(); got != 1 {
		t.Fatalf("GetResource called %d times after cancellation, want 1", got)
	}
}

func TestWaitForResourceDeletionStopsDuringRetrySleepWhenContextCanceled(t *testing.T) {
	checker := &pollingTestChecker{called: make(chan struct{}, 1), status: "DELETING"}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- WaitForResourceDeletion(ctx, "test", checker)
	}()

	<-checker.called
	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("WaitForResourceDeletion() error = %v, want context.Canceled", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("WaitForResourceDeletion did not stop promptly after cancellation")
	}
	if got := checker.calls.Load(); got != 1 {
		t.Fatalf("GetResource called %d times after cancellation, want 1", got)
	}
}
