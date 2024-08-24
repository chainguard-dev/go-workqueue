// Copyright 2024 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	delegate "chainguard.dev/go-grpc-kit/pkg/options"
	"cloud.google.com/go/storage"
	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/sethvargo/go-envconfig"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/oauth2"
	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/oauth"

	"github.com/mattmoor/go-workqueue"
	"github.com/mattmoor/go-workqueue/dispatcher"
	"github.com/mattmoor/go-workqueue/gcs"
)

type envConfig struct {
	Port        int    `env:"PORT, required"`
	Concurrency uint   `env:"WORKQUEUE_CONCURRENCY, required"`
	Mode        string `env:"WORKQUEUE_MODE, required"`
	Bucket      string `env:"WORKQUEUE_BUCKET"`
	Target      string `env:"WORKQUEUE_TARGET, required"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx = clog.WithLogger(ctx, clog.New(slog.Default().Handler()))

	var env envConfig
	envconfig.MustProcess(ctx, &env)

	go httpmetrics.ServeMetrics()

	var wq workqueue.Interface
	switch env.Mode {
	case "gcs":
		cl, err := storage.NewClient(ctx)
		if err != nil {
			log.Panicf("Failed to create client: %v", err)
		}
		wq = gcs.NewWorkQueue(cl.Bucket(env.Bucket), env.Concurrency)

		// Launch a go routine in the background to periodically call Enumerate
		// to ensure that each replica surfaces the latest and greatest metrics
		// even if the worker isn't being invoked for fresh work.
		go func() {
			tick := time.NewTicker(30 * time.Second)
			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					_, _, err := wq.Enumerate(ctx)
					if err != nil {
						log.Printf("Failed to enumerate: %v", err)
					}
				}
			}
		}()

	default:
		log.Panicf("Unsupported mode: %q", env.Mode)
	}

	uri, err := url.Parse(env.Target)
	if err != nil {
		log.Panicf("failed to parse URI: %v", err)
	}
	target, opts := delegate.GRPCOptions(*uri)

	// If the endpoint is TLS terminated (not on K8s), then we are running on
	// Cloud Run and we should authenticate with an ID token.
	if strings.HasPrefix(env.Target, "https://") {
		ts, err := idtoken.NewTokenSource(ctx, env.Target)
		if err != nil {
			log.Panicf("failed to create token source: %v", err)
		}
		opts = append(opts, grpc.WithPerRPCCredentials(oauth.TokenSource{
			TokenSource: oauth2.ReuseTokenSource(nil, ts),
		}))
	}

	conn, err := grpc.NewClient(target, opts...)
	if err != nil {
		log.Panicf("failed to connect to the server: %v", err)
	}
	defer conn.Close()
	client := workqueue.NewWorkqueueServiceClient(conn)

	h := dispatcher.Handler(wq, env.Concurrency, dispatcher.ServiceCallback(client))
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", env.Port),
		Handler:           h2c.NewHandler(h, &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Panicf("failed to start server: %v", err)
	}
}
