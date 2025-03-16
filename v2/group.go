// Package rungroup provides a mechanism for managing and coordinating a collection of goroutines.
//
// It allows you to start multiple goroutines and wait for them to finish, with
// options for canceling other goroutines based on the completion or failure of
// a specific task.
//
// Key features include:
//
//   - Starting goroutines with [Group.Go], [Group.GoCancelOnFinish],
//     [Group.GoCancelOnSuccess], and [Group.GoCancelOnError].
//   - Waiting for all goroutines to finish with [Group.Wait].
//   - Canceling all goroutines with [Group.Cancel] or [Group.Close].
//   - Setting a timeout for the group with [Group.SetTimeout].
//
// This package is useful for scenarios where you need to execute multiple
// tasks concurrently and ensure that they are properly managed and
// coordinated.
//
// # Improvement over v1
//
// v2 aimed for the simplest possible implementation, prioritizing minimalism
// over feature richness.
//
// However, this simplicity was achieved without sacrificing essential
// functionality or addressing key limitations present in v1.
//
// The following improvements were made:
//
//   - Non-terminating tasks:
//     v1 lacked the ability to start a task without automatically triggering
//     cancellation of other tasks. v2 allows tasks to run independently.
//   - Error propagation:
//     v1 did not allow specifying an error when calling `Cancel`. v2 allows
//     errors to be passed to `Cancel` for more informative cancellation.
//   - Zero value safety:
//     v1 panicked when used with a zero value. v2 is safe to use with zero
//     values, initializing with `context.Background`.
//   - Nested tasks:
//     v1 prevented starting new tasks within the same group from within a
//     running task. v2 allows nested tasks to be started within the same
//     group.
package rungroup

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/goaux/stacktrace/v2"
	"github.com/goaux/waitgroup"
)

// ErrClosed is used as the argument when [Group.Close] calls [Group.Cancel].
// If, for a [Group], this call to [Group.Cancel] is the first call to [Group.Cancel],
// then [Group.Wait] will return ErrClosed.
var ErrClosed = errors.New("closed")

// A Group waits for a collection of goroutines to finish.
//
// The main goroutine calls [Group.Go], [Group.GoCancelOnFinish],
// [Group.GoCancelOnSuccess], and [Group.GoCancelOnError] to start a new
// goroutine. Then, each of the goroutines runs and can call [Group.Cancel] if
// desired. At the same time, [Group.Wait] can be used to block until all
// goroutines have finished.
//
// It is possible to use zero values.
type Group struct {
	mu sync.Mutex
	g  waitgroup.Sync

	ctx    context.Context
	cancel context.CancelCauseFunc
}

// New returns a Group initialized with parent as its parent context.
//
// The initialized [Group] must have [Group.Close] or [Group.Cancel] called to
// release its resources when it is no longer needed. This is crucial to
// prevent resource leaks.
//
// A zero-valued Group is initialized with [context.Background] as its parent
// context. Importantly, even a zero-valued Group must have [Group.Close] or
// [Group.Cancel] called when it's no longer needed, just like a Group created
// with `New`. Failing to do so will result in a resource leak.
func New(parent context.Context) *Group {
	ctx, cancel := context.WithCancelCause(parent)
	return &Group{ctx: ctx, cancel: cancel}
}

// Close cancels the [Group] by calling [Group.Cancel] with [ErrClosed],
// thereby releasing its associated resources.
func (gr *Group) Close() {
	gr.Cancel(stacktrace.NewError(ErrClosed, stacktrace.Callers(1)))
}

// Cancel cancels the context for a [Group].
//
// The cause of cancellation is recorded in the context on the first Cancel
// call for each [Group] instance, and can be retrieved via [context.Cause].
func (gr *Group) Cancel(cause error) {
	gr.getContext()
	if cause == nil {
		cause = context.Canceled
	}
	gr.cancel(stacktrace.NewError(cause, stacktrace.Callers(1)))
}

// Wait blocks until all goroutines have exited.
// It returns the argument passed to the first [Group.Cancel] call, or nil if
// [Group.Cancel] was never called.
func (gr *Group) Wait() error {
	gr.getContext()
	gr.g.Wait()
	return context.Cause(gr.ctx)
}

// getContext returns the context for the [Group].
// The context associated with the [Group] will be cancelled when [Group.Cancel] is invoked.
func (gr *Group) getContext() context.Context {
	gr.mu.Lock()
	defer gr.mu.Unlock()
	if gr.ctx == nil {
		gr.ctx, gr.cancel = context.WithCancelCause(context.Background())
	}
	return gr.ctx
}

// Go allows you to start a task in a new goroutine and synchronize its
// completion with a [Group].
//
// The completion of a task does not automatically cancel the [Group]'s context.
// You must explicitly call [Group.Cancel] to cancel the [Group]'s context.
//
// Or, methods are available to call [Group.Cancel] upon task completion.
//
//   - GoCancelOnFinish: Calls [Group.Cancel] when the task finishes (regardless of success or error).
//   - GoCancelOnSuccess: Calls [Group.Cancel] when the task completes successfully.
//   - GoCancelOnError: Calls [Group.Cancel] when the task completes with an error.
//
// These methods are essentially macros around [Group.Go], internally calling
// both [Group.Go] and [Group.Cancel].
//
// The [Group]'s context is passed to the task.
// When [Group.Cancel] is called, the [Group]'s context is cancelled.
func (gr *Group) Go(task func(context.Context)) {
	ctx := gr.getContext()
	gr.g.Go(func() { task(ctx) })
}

// SetTimeout cancels the group's context after the timeout duration has elapsed.
//
// The cancellation is performed by calling [Group.Cancel] with
// [context.DeadlineExceeded] as the argument. If the [Group]'s context is
// canceled before the timeout, [Group.Cancel] is not called.
func (gr *Group) SetTimeout(timeout time.Duration) {
	ctx := gr.getContext()
	callers := stacktrace.Callers(1)
	go func() {
		t := time.NewTimer(timeout)
		defer t.Stop()
		select {
		case <-t.C:
			gr.cancel(stacktrace.NewError(context.DeadlineExceeded, callers))
		case <-ctx.Done():
		}
	}()
}

// GoCancelOnFinish starts a task using [Group.Go] and, when that task
// finishes, cancels all other running tasks in the group using [Group.Cancel].
//
// Use Cases:
//
// Use this when completing one task makes other tasks unnecessary.
//
// For example, imagine a primary task and several helper tasks. If the primary
// task completes, you might want to stop the helpers immediately.
func (gr *Group) GoCancelOnFinish(task func(context.Context) error) {
	callers := stacktrace.Callers(1)
	gr.Go(func(ctx context.Context) {
		err := task(ctx)
		if err == nil {
			err = context.Canceled
		}
		gr.cancel(stacktrace.NewError(err, callers))
	})
}

// GoCancelOnSuccess starts a task using [Group.Go] and, if the task completes
// successfully (returns a nil error), cancels all other running tasks in the
// group by calling [Group.Cancel] with an argument [context.Canceled].
//
// Use Cases:
//
// This is helpful when finishing one task makes other tasks unnecessary.
//
// For example, imagine you have several tasks doing the same thing in
// different ways. You'd want to use the result from the task that finishes
// first.
func (gr *Group) GoCancelOnSuccess(task func(context.Context) error) {
	callers := stacktrace.Callers(1)
	gr.Go(func(ctx context.Context) {
		if err := task(ctx); err == nil { // if NO error
			gr.cancel(stacktrace.NewError(context.Canceled, callers))
		}
	})
}

// GoCancelOnError calls [Group.Go] to start a task, and if the task returns a
// non-nil error, it calls [Group.Cancel] with that error as the argument.
//
// Use Cases:
//
// Use this when one task failing means other tasks can't finish.
//
// Imagine a big task split into smaller parts done at the same time. If one
// part fails, you can't complete the whole thing.
func (gr *Group) GoCancelOnError(task func(context.Context) error) {
	callers := stacktrace.Callers(1)
	gr.Go(func(ctx context.Context) {
		if err := task(ctx); err != nil {
			gr.cancel(stacktrace.NewError(err, callers))
		}
	})
}
