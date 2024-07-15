package rungroup_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/goaux/rungroup"
	"github.com/goaux/timer"
)

func Example() {
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
	// Output:
	// task1 done
	// task2 canceled
	// ok
}

func ExampleNewTimeout() {
	// Create a new group with a 300-milliseconds timeout
	rg := rungroup.NewTimeout(context.Background(), 300*time.Millisecond)

	// Add tasks to the group
	rg.Go(func(ctx context.Context) error {
		if err := timer.Sleep(ctx, 500*time.Millisecond); err != nil {
			fmt.Println("task1 canceled")
			return err
		}
		fmt.Println("task1 done")
		return nil
	})

	rg.Go(func(ctx context.Context) error {
		if err := timer.Sleep(ctx, 150*time.Millisecond); err != nil {
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
	// Output:
	// task2 done
	// task1 canceled
	// ok
}

func TestGroup(t *testing.T) {
	t.Run("All tasks complete successfully", func(t *testing.T) {
		rg := rungroup.New(context.Background())

		rg.Go(func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		})

		rg.Go(func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		err := rg.Wait()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("One task returns error", func(t *testing.T) {
		rg := rungroup.New(context.Background())

		rg.Go(func(ctx context.Context) error {
			return errors.New("task failed")
		})

		rg.Go(func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		err := rg.Wait()
		if err == nil || err.Error() != "task failed" {
			t.Errorf("Expected 'task failed' error, got %v", err)
		}
	})

	t.Run("First completing task cancels others", func(t *testing.T) {
		rg := rungroup.New(context.Background())

		completedChan := make(chan struct{})
		cancelledChan := make(chan struct{})

		rg.Go(func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			close(completedChan)
			return nil
		})

		rg.Go(func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				close(cancelledChan)
				return ctx.Err()
			case <-time.After(200 * time.Millisecond):
				t.Error("Long-running task was not cancelled")
				return nil
			}
		})

		err := rg.Wait()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		select {
		case <-completedChan:
			// This is expected
		case <-time.After(100 * time.Millisecond):
			t.Error("First task did not complete as expected")
		}

		select {
		case <-cancelledChan:
			// This is expected
		case <-time.After(100 * time.Millisecond):
			t.Error("Second task was not cancelled as expected")
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		rg := rungroup.New(ctx)

		rg.Go(func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		})

		cancel()
		err := rg.Wait()
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})

	t.Run("Timeout", func(t *testing.T) {
		rg := rungroup.NewTimeout(context.Background(), 50*time.Millisecond)

		rg.Go(func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			return nil
		})

		err := rg.Wait()
		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got %v", err)
		}
	})

	t.Run("Wait returns immediately with no tasks", func(t *testing.T) {
		rg := rungroup.New(context.Background())

		done := make(chan struct{})
		go func() {
			err := rg.Wait()
			if err != nil {
				t.Errorf("Expected nil error, got %v", err)
			}
			close(done)
		}()

		select {
		case <-done:
			// This is expected
		case <-time.After(100 * time.Millisecond):
			t.Error("Wait did not return immediately")
		}
	})

	t.Run("Group cannot be reused after Wait", func(t *testing.T) {
		rg := rungroup.New(context.Background())

		_ = rg.Wait() // First call to Wait

		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic when reusing Group, but no panic occurred")
			}
		}()

		rg.Go(func(context.Context) error { return nil }) // This should panic
	})
}
