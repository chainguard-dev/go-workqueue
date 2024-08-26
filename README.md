# go-workqueue

This is a Go library that can be used to queue and process work with a number of
properties reminiscient of Kubernetes Controller's workqueue, but that
support being backed by durable storage (e.g. GCS, S3).

Given a `workqueue.Interface` new keys can be added to the queue with
deduplication by simply calling `Queue(ctx, key)`.  For example:

```go
func foo(ctx context.Context, wq workqueue.Interface) error {
    if err := wq.Queue(ctx, "foo"); err != nil {
        return err
    }
    if err := wq.Queue(ctx, "bar"); err != nil {
        return err
    }
    if err := wq.Queue(ctx, "baz"); err != nil {
        return err
    }
    return nil
}
```

Up to `N` items of concurrent work may be processed from the workqueue by
invoking the `Handle` method in the
`github.com/chainguard-dev/go-workqueue/dispatcher` package.  For example:

```go

if err := dispatcher.Handle(ctx, wq, N, func(ctx context.Context, key string) error {
    // Process the key!
    clog.InfoContextf(ctx, "Got key: %s", key)
    return nil
}); err != nil {
    return err
}
```
