package rungroup_test

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	rungroup "github.com/goaux/rungroup/v2"
)

func Example() {
	n := int32(0)
	var gr rungroup.Group // It is possible to use zero values.
	defer gr.Close()
	gr.Go(func(context.Context) { atomic.AddInt32(&n, 1) })
	gr.Go(func(context.Context) { atomic.AddInt32(&n, 1) })
	err := gr.Wait()
	fmt.Println(n, err)
	// Output:
	// 2 <nil>
}

func ExampleGroup_cancel() {
	n := int32(0)
	var gr rungroup.Group
	defer gr.Close()
	gr.Go(func(ctx context.Context) { gr.Cancel(fmt.Errorf("testing")); atomic.AddInt32(&n, 1) })
	gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
	err := gr.Wait()
	fmt.Println(n, err)
	// Output:
	// 2 testing (group_test.go:30 ExampleGroup_cancel.func1)
}

func ExampleGroup_setTimeout() {
	n := int32(0)
	var gr rungroup.Group
	defer gr.Close()
	gr.SetTimeout(time.Millisecond)
	gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
	gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
	err := gr.Wait()
	fmt.Println(n, err)
	// Output:
	// 2 context deadline exceeded (group_test.go:42 ExampleGroup_setTimeout)
}

func TestNew(t *testing.T) {
	t.Run("", func(t *testing.T) {
		n := int32(0)
		gr := rungroup.New(context.TODO())
		defer gr.Close()
		gr.Go(func(context.Context) { atomic.AddInt32(&n, 1) })
		gr.Go(func(context.Context) { atomic.AddInt32(&n, 1) })
		err := gr.Wait()
		assertNoError(t, err)
		assertEqual(t, n, 2)
	})

	t.Run("cancel parent", func(t *testing.T) {
		parent, cancelParent := context.WithCancel(context.TODO())
		n := int32(0)
		gr := rungroup.New(parent)
		defer gr.Close()
		gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
		gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
		cancelParent()
		err := gr.Wait()
		assertErrorIs(t, err, context.Canceled)
		assertEqual(t, n, 2)
	})
}

func TestGroup(t *testing.T) {
	t.Run("Close", func(t *testing.T) {
		var gr rungroup.Group
		gr.Close()
		err := gr.Wait()
		assertErrorIs(t, err, rungroup.ErrClosed)
	})

	t.Run("Cancel", func(t *testing.T) {
		n := int32(0)
		gr := rungroup.New(context.TODO())
		defer gr.Close()
		gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
		gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
		gr.Cancel(nil)
		err := gr.Wait()
		assertErrorIs(t, err, context.Canceled)
		assertEqual(t, n, 2)
	})

	t.Run("nested start", func(t *testing.T) {
		n := int32(0)
		var gr rungroup.Group
		defer gr.Close()
		gr.Go(func(context.Context) {
			gr.Go(func(context.Context) {
				atomic.AddInt32(&n, 1)
			})
			atomic.AddInt32(&n, 1)
		})
		err := gr.Wait()
		assertNoError(t, err)
		assertEqual(t, n, 2)
	})

	ErrStop := errors.New("stop")

	t.Run("GoCnacelOnFinish", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			n := int32(0)
			var gr rungroup.Group
			defer gr.Close()
			gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
			gr.GoCancelOnFinish(func(ctx context.Context) error { atomic.AddInt32(&n, 1); return nil })
			err := gr.Wait()
			assertErrorIs(t, err, context.Canceled)
			assertEqual(t, n, 2)
		})
		t.Run("error", func(t *testing.T) {
			n := int32(0)
			var gr rungroup.Group
			defer gr.Close()
			gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
			gr.GoCancelOnFinish(func(ctx context.Context) error { atomic.AddInt32(&n, 1); return ErrStop })
			err := gr.Wait()
			assertErrorIs(t, err, ErrStop)
			assertEqual(t, n, 2)
		})
	})

	t.Run("GoCancelOnSuccess", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			n := int32(0)
			var gr rungroup.Group
			defer gr.Close()
			gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
			gr.GoCancelOnSuccess(func(ctx context.Context) error { atomic.AddInt32(&n, 1); return nil })
			err := gr.Wait()
			assertErrorIs(t, err, context.Canceled)
			assertEqual(t, n, 2)
		})
		t.Run("error", func(t *testing.T) {
			n := int32(0)
			var gr rungroup.Group
			defer gr.Close()
			gr.Go(func(ctx context.Context) { atomic.AddInt32(&n, 1) })
			gr.GoCancelOnSuccess(func(ctx context.Context) error { atomic.AddInt32(&n, 1); return ErrStop })
			err := gr.Wait()
			assertNoError(t, err)
			assertEqual(t, n, 2)
		})
	})

	t.Run("GoCancelOnError", func(t *testing.T) {
		t.Run("ok", func(t *testing.T) {
			n := int32(0)
			var gr rungroup.Group
			defer gr.Close()
			gr.Go(func(ctx context.Context) { atomic.AddInt32(&n, 1) })
			gr.GoCancelOnError(func(ctx context.Context) error { atomic.AddInt32(&n, 1); return nil })
			err := gr.Wait()
			assertNoError(t, err)
			assertEqual(t, n, 2)
		})
		t.Run("error", func(t *testing.T) {
			n := int32(0)
			var gr rungroup.Group
			defer gr.Close()
			gr.Go(func(ctx context.Context) { <-ctx.Done(); atomic.AddInt32(&n, 1) })
			gr.GoCancelOnError(func(ctx context.Context) error { atomic.AddInt32(&n, 1); return ErrStop })
			err := gr.Wait()
			assertErrorIs(t, err, ErrStop)
			assertEqual(t, n, 2)
		})
	})
}

// assertEqual calls t.Error if got is not equal to want.
func assertEqual[T comparable](t *testing.T, actual, expect T, msg ...any) {
	t.Helper()
	if actual != expect {
		t.Error(fmt.Sprintf("must equal; actual=%v, expect=%v", actual, expect), fmt.Sprint(msg...))
	}
}

// assertNoError calls t.Error if actual is not nil.
func assertNoError(t *testing.T, actual error, msg ...any) {
	t.Helper()
	if actual != nil {
		t.Error(fmt.Sprintf("must be nil, actual=%#v", actual), fmt.Sprint(msg...))
	}
}

func assertErrorIs(t *testing.T, actual, expect error, msg ...any) {
	t.Helper()
	if !errors.Is(actual, expect) {
		t.Error(fmt.Sprintf("actual=%#v, expect=%#v", actual, expect), fmt.Sprint(msg...))
	}
}
