/*
Copyright 2024 Chainguard, Inc.
SPDX-License-Identifier: Apache-2.0
*/

package gcs

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sethvargo/go-envconfig"
)

var env = envconfig.MustProcess(context.Background(), &struct {
	// https://cloud.google.com/run/docs/container-contract#services-env-vars
	KnativeServiceName  string `env:"K_SERVICE, default=unknown"`
	KnativeRevisionName string `env:"K_REVISION, default=unknown"`
}{})

var (
	// TODO(mattmoor): Inspiration:
	// https://pkg.go.dev/k8s.io/client-go/util/workqueue#MetricsProvider

	mInProgressKeys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_in_progress_keys",
			Help: "The number of keys currently being processed by this workqueue.",
		},
		[]string{"service_name", "revision_name"},
	)
	mQueuedKeys = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "workqueue_queued_keys",
			Help: "The number of keys currently in the backlog of this workqueue.",
		},
		[]string{"service_name", "revision_name"},
	)
)