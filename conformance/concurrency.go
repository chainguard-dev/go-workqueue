/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package conformance

import (
	"context"
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/mattmoor/go-workqueue"
	"github.com/mattmoor/go-workqueue/dispatcher"
	"golang.org/x/sync/errgroup"
)

func TestConcurrency(t *testing.T, ctor func(uint) workqueue.Interface) {
	wq := ctor(5)
	if wq == nil {
		t.Fatal("NewWorkQueue returned nil")
	}

	var cb dispatcher.Callback = func(ctx context.Context, key string) error {
		t.Logf("Processing %q", key)
		// This is intentionally much longer than the tick below, to ensure that
		// we handle multiple concurrent dispatch invocations.
		time.Sleep(time.Second)
		return nil
	}

	eg := errgroup.Group{}
	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		if err := eg.Wait(); err != nil {
			t.Errorf("Error group failed: %v", err)
		}
	}()
	defer cancel()

	eg.Go(func() error {
		// This is intentionally MUCH lower than the sleep above, to ensure that
		// we see a lot of concurrent dispatch invocations.
		tick := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-tick.C:
				// Do this in a go routine, so it doesn't block the
				// dispatch loop.
				eg.Go(func() error {
					return dispatcher.Handle(context.WithoutCancel(ctx), wq, 5, cb)
				})
			}
		}
	})

	for i := 0; i < 1000; i++ {
		key := fmt.Sprint(rand.IntN(40))
		if err := wq.Queue(ctx, key); err != nil {
			t.Fatalf("Queue failed: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	for {
		wip, qd, err := wq.Enumerate(ctx)
		if err != nil {
			t.Fatalf("Enumerate failed: %v", err)
		}
		if len(wip) == 0 && len(qd) == 0 {
			break
		}
		t.Logf("Waiting for work to complete (wip: %d, qd: %d)", len(wip), len(qd))
		time.Sleep(100 * time.Millisecond)
	}
}
