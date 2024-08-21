/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package dispatcher

import (
	"context"
	"fmt"

	"github.com/chainguard-dev/clog"
	"golang.org/x/sync/errgroup"

	"github.com/mattmoor/go-workqueue"
)

// Callback is the function that Handle calls to process a particular key.
type Callback func(ctx context.Context, key string) error

// ServiceCallback returns a Callback that invokes the given service.
func ServiceCallback(client workqueue.WorkqueueServiceClient) Callback {
	return func(ctx context.Context, key string) error {
		_, err := client.Process(ctx, &workqueue.ProcessRequest{
			Key: key,
		})
		return err
	}
}

// Handle performs a single iteration of the dispatcher, possibly invoking
// the callback for several different keys.
func Handle(ctx context.Context, wq workqueue.Interface, concurrency uint, f Callback) error {
	// Enumerate the state of the queue.
	wip, next, err := wq.Enumerate(ctx)
	if err != nil {
		return fmt.Errorf("enumerate() = %w", err)
	}

	// Remove any orphaned work by returning it to the queue.
	activeKeys := make(map[string]struct{}, len(wip))
	for _, x := range wip {
		if !x.IsOrphaned() {
			activeKeys[x.Name()] = struct{}{}
			continue
		}
		if err := x.Requeue(ctx); err != nil {
			return fmt.Errorf("requeue() = %w", err)
		}
	}

	// Attempt to launch a new piece of work for each open slot we have available
	// which is: N - active.
	openSlots := concurrency - uint(len(activeKeys))
	idx, launched := 0, uint(0)
	eg := errgroup.Group{}
	for ; idx < len(next) && launched < openSlots; idx++ {
		nextKey := next[idx]

		// If the next key is already in progress, then move to the next candidate.
		if _, ok := activeKeys[nextKey.Name()]; ok {
			continue
		}

		// At this point, we know that nextKey gets launched.  There are two paths below:
		// 1. One is where we lose the race and someone else launches it, and
		// 2. The other is where we launch it.
		// By incrementing the counter here, we ensure we don't overlaunch keys due to a race.
		launched++

		// This is done in a Go routine so that we can process keys concurrently.
		eg.Go(func() error {
			// Start the work, moving it to be in-progress. If we are unsuccessful starting
			// the work, then someone beat us to it, so move on to the next key.
			oip, err := nextKey.Start(ctx)
			if err != nil {
				clog.DebugContextf(ctx, "Failed to start key %q: %v", nextKey.Name(), err)
				return nil
			}

			// Attempt to perform the actual reconciler invocation.
			if err := f(ctx, oip.Name()); err != nil {
				clog.WarnContextf(ctx, "Failed callback for key %q: %v", oip.Name(), err)

				// Requeue if it fails (stops heartbeat).
				if err := oip.Requeue(ctx); err != nil {
					return fmt.Errorf("requeue(after failed callback) = %w", err)
				}
				return nil // This isn't an error in the dispatcher itself.
			}
			// Delete the in-progress key (stops heartbeat).
			if err := oip.Complete(ctx); err != nil {
				return fmt.Errorf("complete() = %w", err)
			}
			return nil
		})
	}
	clog.InfoContextf(ctx, "Launched %d new keys (wip: %d)", launched, len(activeKeys))

	// Wait for all of the in-progress invocations to complete.
	return eg.Wait()
}
