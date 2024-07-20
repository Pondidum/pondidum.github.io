+++
title = 'Multiple errors in an OTEL Span'
tags = ['opentelemetry']
+++


A question at work came up recently about how to handle multiple errors in a single span.  My reaction is that having multiple errors in a span is a design smell, but I had no particular data or spec to back this up, just a gut feeling.  I've since thought about this more and think that in general, you should only have one error per span, but there are circumstances where multiple indeed makes sense.  So, let's dig into them.

## Can I have multiple errors in a Span?

Yes!  An error is stored as an `event` attached to the `span`, so calling `span.recordError(err)` can be called multiple times.  However, a span can only have one status: either `Unset`, `Ok`, or `Error`, so while many errors can be recorded, the status of the span can only be set to one value.

## Multiple unrelated errors

The problem with wanting to store multiple errors in one trace is that while you can have multiple error events, if you are also using the span's attributes for the error details (i.e. setting `err.message` and `err.details` etc.) then whichever was your last error will overwrite previous values.

One solution offered is to combine all errors into one, and then record that in the attributes.  This feels like a bad idea to me due to how you consume the errors in your OTEL service of choice; generally, you filter by attributes without any kind of free text search or, at best, prefix matching.

This, however, means that you lose a lot of filtering ability.  Rather than being able to do direct matching on the `err.message` attribute, you need to do wildcard searching, which is slower, and some providers don't even support it.

Furthermore, you lose the ability to group by error type; you can no longer write `group by err.message` or `group by err.type` as they all have the same type (`composite`), or differing messages (especially if the order of the grouped errors is not deterministic.)  You also loose out on being able to track how often a specific kind of error is occurring, making alerting on changes in error rates much harder to implement.

For these reasons, I would recommend refactoring your code to have a span per operation (or function, whichever provides the granularity you need), such that the main method emits a single span with several child spans, which each can have their own error details, and the parent span can then contain an error, and the overall status of the operation.

```go
func handleMessage(ctx context.Context, m Message) error {
  ctx, span := tr.StartSpan(ctx, "handle_message")
  defer span.End()

  oneSuccess := childOperationOne(ctx, m)
  twoSuccess := childOperationTwo(ctx, m)
  threeSuccess := childOperationThree(ctx, m)

  childOperationFour(ctx, m) //optional

  if oneSuccess && twoSuccess && threeSuccess {
    span.SetStatus(codes.Ok)
  } else {
    span.SetStatus(codes.Error, "mandatory operations failed")
  }
}
```

## Multiple related errors

A good use case for storing multiple errors is when the operation can emit multiple of the same error before succeeding.  For example, an HTTP request retry middleware might emit an error for each attempt it makes but only set the span status to error if there wasn't a successful call after several tries.

```go
func Retry(ctx context.Context, req http.Request) (http.Response, error) {
  ctx, span := tr.StartSpan("retry", ctx)
  defer span.End()

  maxRetries := 5
  span.SetAttributes(attribute.Int("retries.max", maxRetries))

  for i := range maxRetries {
    span.SetAttributes(attribute.Int("retries.current", i))

    res, err := client.Do(req)
    if err == nil {
      span.SetStatus(codes.Ok, "")
      return res, nil
    }

    s.RecordError(err)
    backOffSleep(i)
  }

  retryError := fmt.Errorf("failed after %v tries", maxRetries)
  span.SetStatus(codes.Error, retryError.Error())

  return nil, retryError
}
```

## Tracing Providers

A lot of how you deal with this is still going to come down to which tracing provider/software you use to actually ingest your traces.  While OpenTelemetry has a specification of the protocol and fields, it doesn't specify how a UI has to render that information, so it is possible that you have to adjust your usage to match your provider.

or example, if there are many child operations, then a span per operation with its own possibility of error would make sense; the parent span could then decide based on the child operations whether the whole operation was successful or not.  You could also add attributes such as `operation_count`, `operation_failures`, etc.

## Next Steps

For the codebase in question, based on what little I know of it, I think adding a few more Spans around the problem areas is the right thing to do;  but depending on how many places this occurs, it could be more hassle than its worth.

I'd stick with the same approach for adding tracing to a codebase: add it where it hurts, as that's where you'll see the most value.