# rungroup
A Go module for managing concurrent tasks with automatic cancellation when any task completes or fails.

[![Go Reference](https://pkg.go.dev/badge/github.com/goaux/rungroup.svg)](https://pkg.go.dev/github.com/goaux/rungroup)
[![Go Report Card](https://goreportcard.com/badge/github.com/goaux/rungroup)](https://goreportcard.com/report/github.com/goaux/rungroup)

`rungroup` is a Go module that provides a way to manage and synchronize
concurrent tasks. Its primary feature is the ability to cancel all goroutines
when any one of them completes. This package is particularly useful for
scenarios where you need to run multiple operations concurrently but want to
stop all of them as soon as any one operation completes or fails.

## Features

- Manage multiple goroutines as a group
- Automatic cancellation of all tasks when one completes
- Support for timeouts
- Easy-to-use API

## Installation

To install the `rungroup` module, use `go get`:

```
go get github.com/goaux/rungroup
```

## Usage

Here's a basic example of how to use the `rungroup` module:

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/goaux/rungroup"
	"github.com/goaux/timer"
)

func main() {
	// Create a new group with no timeout
	rg := rungroup.New(context.Background())

	// Add tasks to the group
	rg.Go(func(ctx context.Context) error {
		if err := timer.Sleep(ctx, 150*time.Millisecond); err != nil {
			fmt.Println("task1 canceled")
			return err
		}
		fmt.Println("task1 done")
		return nil
	})

	rg.Go(func(ctx context.Context) error {
		if err := timer.Sleep(ctx, 300*time.Millisecond); err != nil {
			fmt.Println("task2 canceled")
			return err
		}
		fmt.Println("task2 done")
		return nil
	})

	// Wait for all tasks to complete or be canceled
	err := rg.Wait()
	if err != nil {
		fmt.Printf("must not error: %v\n", err)
	} else {
		fmt.Println("ok")
	}
}
```

In this example, the first task completes after 150 milliseconds, which
triggers the cancellation of the second task. The `Wait` method returns when
all tasks have either completed or been canceled.

## API

### `New(ctx context.Context) *Group`

Creates a new `Group` with the given context.

### `NewTimeout(ctx context.Context, d time.Duration) *Group`

Creates a new `Group` with the given context and timeout duration.

### `(g *Group) Go(task func(context.Context) error)`

Starts a new goroutine in the `Group`.

### `(g *Group) Wait() error`

Blocks until all tasks in the `Group` have completed or been canceled.

### `(g *Group) Cancel()`

Explicitly cancels the `Group`'s context, causing all tasks to be interrupted.

## Note

- A `Group` must be created by `New` or `NewTimeout`, zero value must not be used.
- A `Group` must not be copied after first use.
- A `Group` must not be reused after calling `Wait`.
