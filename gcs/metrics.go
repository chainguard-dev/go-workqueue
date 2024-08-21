/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	mInProgressKeys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_in_progress_keys",
			Help: "The number of keys currently being processed by this workqueue.",
		},
		[]string{},
	)
	mQueuedKeys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_queued_keys",
			Help: "The number of keys currently in the backlow of this workqueue.",
		},
		[]string{},
	)
)
