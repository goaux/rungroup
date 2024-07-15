// Package rungroup provides a way to manage and synchronize concurrent tasks.
//
// Its primary feature is the ability to cancel all goroutines when any one of them completes.
// This package is particularly useful for scenarios where you need to run multiple operations
// concurrently, but want to stop all of them as soon as any one operation completes or fails.
package rungroup

import (
	"context"
	"sync"
	"time"

	"github.com/goaux/waitgroup"
)

// Group represents a collection of goroutines working on subtasks that are part of
// the same overall task. The Group is designed to manage concurrent execution and
// provides automatic cancellation of all running tasks when any single task completes.
//
//   - A Group must be created by New or NewTimeout, zero value must not be used.
//   - A Group must not be copied after first use.
//   - A Group must not be reused after calling Wait.
type Group struct {
	ctx    context.Context
	cancel context.CancelFunc

	wg waitgroup.Sync

	errOnce sync.Once
	err     error

	doNotReuse bool
}

// New creates a new Group with the given context.
// The returned Group's context is canceled when the first task completes (returns),
// regardless of whether it returns an error or nil. This cancellation triggers
// the termination of all other running tasks in the group.
func New(ctx context.Context) *Group {
	ctx, cancel := context.WithCancel(ctx)
	return &Group{
		ctx:    ctx,
		cancel: cancel,
	}
}

// NewTimeout creates a new Group with the given context and timeout duration.
// The returned Group's context is canceled when the timeout expires, or when the first
// task completes (returns), whichever happens first. This cancellation triggers
// the termination of all other running tasks in the group.
//
// The timeout is implemented as a separate task within the group, ensuring
// consistent behavior with other tasks.
func NewTimeout(ctx context.Context, d time.Duration) *Group {
	g := New(ctx)
	g.Go(func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, d)
		defer cancel()
		<-ctx.Done()
		return ctx.Err()
	})
	return g
}

// Cancel explicitly cancels the Group's context, causing all tasks to be interrupted.
// This method can be used to manually trigger the cancellation of all running tasks.
func (g *Group) Cancel() {
	g.checkInitialized()
	g.setResult(context.Canceled)
}

// Go starts a new goroutine in the Group.
// The provided function is executed in its own goroutine. If this function
// completes (either by returning nil or an error), it will trigger the
// cancellation of the Group's context, causing all other running tasks to be terminated.
//
// Go will panic if called on a Group that has already been used (i.e., after Wait has been called).
func (g *Group) Go(task func(context.Context) error) {
	g.checkInitialized()
	g.checkReuse()
	g.wg.Go(func() {
		err := task(g.ctx)
		g.setResult(err)
	})
}

// Wait blocks until all tasks in the Group have completed or been cancelled.
// It returns the first non-nil error (if any) from any of the tasks. If a task
// completes without error, Wait will still trigger the cancellation of all other
// tasks before returning.
//
// If Wait is called without any tasks being started using the Go method,
// it returns nil immediately. Even in this case, the Group cannot be reused.
func (g *Group) Wait() error {
	g.checkInitialized()
	g.doNotReuse = true
	g.wg.Wait()
	// Call setResult to always call cancel even if the Go method has never been called.
	g.setResult(nil)
	return g.err
}

func (g *Group) checkInitialized() {
	if g.cancel == nil {
		panic("A Group must be created by New or NewTimeout, zero value must not be used.")
	}
}

func (g *Group) checkReuse() {
	if g.doNotReuse {
		panic("A Group must must not be reused.")
	}
}

func (g *Group) setResult(err error) {
	g.errOnce.Do(func() {
		g.err = err
		g.cancel()
	})
}
