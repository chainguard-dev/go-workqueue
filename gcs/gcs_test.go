/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/mattmoor/go-workqueue"
	"github.com/mattmoor/go-workqueue/conformance"
)

func TestWorkQueue(t *testing.T) {
	client, err := storage.NewClient(context.Background())
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	bucket, ok := os.LookupEnv("WORKQUEUE_GCS_TEST_BUCKET")
	if !ok {
		t.Skip("WORKQUEUE_GCS_TEST_BUCKET not set")
	}

	conformance.TestSemantics(t, func(u uint) workqueue.Interface {
		return NewWorkQueue(client.Bucket(bucket), u)
	})

	conformance.TestConcurrency(t, func(u uint) workqueue.Interface {
		return NewWorkQueue(client.Bucket(bucket), u)
	})

	conformance.TestDurability(t, func(u uint) workqueue.Interface {
		return NewWorkQueue(client.Bucket(bucket), u)
	})
}
