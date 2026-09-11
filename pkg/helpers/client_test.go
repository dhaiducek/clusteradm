// Copyright Contributors to the Open Cluster Management project

package helpers

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func TestWatchUntilSuccessOnFirstEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	calls := 0
	err := WatchUntil(ctx, func() (watch.Interface, error) {
		calls++
		fw := watch.NewFakeWithChanSize(1, false)
		fw.Add(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "ready"}})
		return fw, nil
	}, func(event watch.Event) bool {
		return event.Type == watch.Added
	})
	if err != nil {
		t.Fatalf("WatchUntil() unexpected error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("WatchUntil() watchFunc calls = %d, want 1", calls)
	}
}

func TestWatchUntilRetryAfterClose(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	calls := 0
	err := WatchUntil(ctx, func() (watch.Interface, error) {
		calls++
		fw := watch.NewFakeWithChanSize(1, false)
		if calls == 1 {
			fw.Stop()
			return fw, nil
		}
		fw.Add(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "ready"}})
		return fw, nil
	}, func(event watch.Event) bool {
		return event.Type == watch.Added
	})
	if err != nil {
		t.Fatalf("WatchUntil() unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("WatchUntil() watchFunc calls = %d, want 2", calls)
	}
}

func TestWatchUntilRetryAfterError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	calls := 0
	err := WatchUntil(ctx, func() (watch.Interface, error) {
		calls++
		fw := watch.NewFakeWithChanSize(1, false)
		if calls == 1 {
			fw.Error(&metav1.Status{Status: metav1.StatusFailure, Message: "watch error"})
			return fw, nil
		}
		fw.Add(&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "ready"}})
		return fw, nil
	}, func(event watch.Event) bool {
		return event.Type == watch.Added
	})
	if err != nil {
		t.Fatalf("WatchUntil() unexpected error: %v", err)
	}
	if calls != 2 {
		t.Fatalf("WatchUntil() watchFunc calls = %d, want 2", calls)
	}
}

func TestWatchUntilTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := WatchUntil(ctx, func() (watch.Interface, error) {
		return watch.NewFake(), nil
	}, func(event watch.Event) bool {
		return false
	})
	if err == nil {
		t.Fatal("WatchUntil() error = nil, want timeout")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("WatchUntil() error = %v, want wrapped context.DeadlineExceeded", err)
	}
}

func TestWatchUntilCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := WatchUntil(ctx, func() (watch.Interface, error) {
		return watch.NewFake(), nil
	}, func(event watch.Event) bool {
		return false
	})
	if err == nil {
		t.Fatal("WatchUntil() error = nil, want canceled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("WatchUntil() error = %v, want wrapped context.Canceled", err)
	}
}

func TestWatchUntilWatchFuncError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	want := fmt.Errorf("auth failed")
	calls := 0
	err := WatchUntil(ctx, func() (watch.Interface, error) {
		calls++
		return nil, want
	}, func(event watch.Event) bool {
		return true
	})
	if !errors.Is(err, want) {
		t.Fatalf("WatchUntil() error = %v, want %v", err, want)
	}
	if calls != 1 {
		t.Fatalf("WatchUntil() watchFunc calls = %d, want 1 (no retry)", calls)
	}
}
