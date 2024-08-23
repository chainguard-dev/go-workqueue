// Copyright 2024 Chainguard, Inc.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"chainguard.dev/go-grpc-kit/pkg/duplex"
	"cloud.google.com/go/storage"
	"github.com/chainguard-dev/clog"
	"github.com/chainguard-dev/terraform-infra-common/pkg/httpmetrics"
	"github.com/sethvargo/go-envconfig"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	"github.com/mattmoor/go-workqueue"
	"github.com/mattmoor/go-workqueue/gcs"
)

type envConfig struct {
	Port        int    `env:"PORT, required"`
	Concurrency uint   `env:"WORKQUEUE_CONCURRENCY, required"`
	Mode        string `env:"WORKQUEUE_MODE, required"`
	Bucket      string `env:"WORKQUEUE_BUCKET"`
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx = clog.WithLogger(ctx, clog.New(slog.Default().Handler()))

	var env envConfig
	envconfig.MustProcess(ctx, &env)

	go httpmetrics.ServeMetrics()

	d := duplex.New(
		env.Port,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	var wq workqueue.Interface

	switch env.Mode {
	case "gcs":
		cl, err := storage.NewClient(ctx)
		if err != nil {
			log.Panicf("Failed to create client: %v", err)
		}

		wq = gcs.NewWorkQueue(cl.Bucket(env.Bucket), env.Concurrency)

	default:
		log.Panicf("Unsupported mode: %q", env.Mode)
	}

	workqueue.RegisterWorkqueueServiceServer(d.Server, &enq{wq: wq})
	if err := d.ListenAndServe(ctx); err != nil {
		log.Panicf("ListenAndServe() = %v", err)
	}
}

type enq struct {
	workqueue.UnimplementedWorkqueueServiceServer

	wq workqueue.Interface
}

func (y *enq) Process(ctx context.Context, req *workqueue.ProcessRequest) (*workqueue.ProcessResponse, error) {
	if err := y.wq.Queue(ctx, req.Key); err != nil {
		return nil, status.Errorf(codes.Internal, "Queue() = %v", err)
	}
	return &workqueue.ProcessResponse{}, nil
}
