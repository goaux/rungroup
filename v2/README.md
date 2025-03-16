# rungroup/v2

A Go module for managing and coordinating concurrent tasks with fine-grained control over cancellation.

[![Go Reference](https://pkg.go.dev/badge/github.com/goaux/rungroup/v2.svg)](https://pkg.go.dev/github.com/goaux/rungroup/v2)
[![Go Report Card](https://goreportcard.com/badge/github.com/goaux/rungroup/v2)](https://goreportcard.com/report/github.com/goaux/rungroup/v2)

`rungroup/v2` is a Go module that provides a robust mechanism for managing and coordinating a collection of goroutines. It allows you to start multiple tasks concurrently and coordinate their execution with fine-grained control over cancellation behavior.

## Improvements over v1

Version 2 of `rungroup` addresses several limitations present in v1:

- **Non-terminating tasks**: v1 automatically triggered cancellation when any task completed. v2 allows tasks to run independently with explicit control over cancellation behavior.
- **Error propagation**: v2 allows specifying an error when calling `Cancel`, providing more informative cancellation reasons.
- **Zero value safety**: v1 panicked when used with a zero value. v2 is safe to use with zero values, automatically initializing with `context.Background`.
- **Nested tasks**: v1 prevented starting new tasks within the same group from within a running task. v2 allows nested tasks to be started within the same group.

## Installation

To install the `rungroup/v2` module, use `go get`:

```
go get github.com/goaux/rungroup/v2
```

## Usage

Here's a basic example of how to use the `rungroup/v2` module:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/goaux/rungroup/v2"
	"github.com/goaux/stacktrace/v2"
)

func main() {
	// Create a new group
	gr := rungroup.New(context.Background())

	// Ensure we clean up resources
	defer gr.Close()

	// Add a task that will finish after 150ms
	gr.Go(func(ctx context.Context) {
		select {
		case <-time.After(150 * time.Millisecond):
			fmt.Println("Task 1 completed")
		case <-ctx.Done():
			fmt.Println("Task 1 canceled")
		}
	})

	// Add a task that will finish after 300ms
	// This task will only be canceled if explicitly requested
	gr.Go(func(ctx context.Context) {
		select {
		case <-time.After(300 * time.Millisecond):
			fmt.Println("Task 2 completed")
		case <-ctx.Done():
			fmt.Println("Task 2 canceled")
		}
	})

	// Add a task that will cancel the group on completion
	gr.GoCancelOnFinish(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		fmt.Println("Task 3 completed, canceling group")
		return nil
	})

	// Wait for all tasks to complete or be canceled
	err := gr.Wait()
	if err != nil {
		fmt.Printf("Group canceled with error: %v\n", stacktrace.Format(err))
	} else {
		fmt.Println("All tasks completed successfully")
	}
}
```

## API Overview

### Creating a Group

```go
// Create a new Group with a parent context
gr := rungroup.New(ctx)

// A zero value is also valid
var gr rungroup.Group
```

### Starting Tasks

```go
// Start a task without automatic cancellation
gr.Go(func(ctx context.Context) {
    // Your task logic here
})

// Start a task that cancels the group when it completes (success or failure)
gr.GoCancelOnFinish(func(ctx context.Context) error {
    // Your task logic here
    return nil
})

// Start a task that cancels the group only on successful completion
gr.GoCancelOnSuccess(func(ctx context.Context) error {
    // Your task logic here
    return nil
})

// Start a task that cancels the group only on error
gr.GoCancelOnError(func(ctx context.Context) error {
    // Your task logic here
    return errors.New("task failed")
})
```

### Controlling the Group

```go
// Cancel the group with a specific error
gr.Cancel(errors.New("operation aborted"))

// Cancel the group with ErrClosed
gr.Close()

// Set a timeout for the group
gr.SetTimeout(5 * time.Second)

// Wait for all tasks to complete
err := gr.Wait()
```

## Resource Management

It's important to call either `gr.Close()` or `gr.Cancel()` when a Group is no longer needed to prevent resource leaks. This applies to both Groups created with `New()` and zero-value Groups.

## Thread Safety

`rungroup/v2` is designed to be thread-safe. You can start tasks from multiple goroutines, including from within other tasks in the same group.

## License

[Apache License Version 2.0](LICENSE)
