+++
title = 'Telemetry and Cancelled Contexts'
tags = [ "opentelemetry", "observability", "tracing", "golang" ]
+++

I use opentelemetry extensively to trace my applications, and one thing I keep running into is when writing a long running process, I want to handle OS signals and still send the telemetry on shutdown.

Typically, my application startup looks something like this:


```go
func main() {
  ctx, cancel := context.WithCancel(context.Background())
  handleSignals(cancel)

  tracerProvider := configureTelemetry(ctx)
  defer tracerProvider.Shutdown(ctx)

  tr = traceProvider.Tracer("cli")

  if err := runMain(ctx, os.Args[:]); err != nil {
    fmt.Fprintf(os.Stderr, err.Error())
    os.Exit(1)
  }
}

func runMain(ctx context.Context, args []string) error {
  ctx, span := tr.Start(ctx, "main")
  defer span.End()

  // some kind of loop
  for _, message := range someProcess(ctx) {
    select {
      case <-ctx.Done:
        return ctx.Err()

      default:
        doMessageThings(ctx, message)
    }
  }

  return nil
}
```

The `handleSignals` method listens to things like `sigint`, and calls the `cancel()` function, and the app stops processing messages and exits gracefully.

When the application exited due to errors, I would see the whole trace from the application, but if the application stopped due to `sigint` or similar, the `main` span would never come through.

After a bit of reading, I realised the bug is here:

```diff
ctx, cancel := context.WithCancel(context.Background())
handleSignals(cancel)

tracerProvider := configureTelemetry(ctx)
- defer tracerProvider.Shutdown(ctx)
+ defer tracerProvider.Shutdown(context.Background())
```

The problem is that when the context has been cancelled, the tracerProvider skips doing any work, so never sends through the last spans!

This issue would have been more noticable if:

- I checked the `err` value from `Shutdown()`, which is easily missed in a `defer` call
- Setting `OTEL_LOG_LEVEL` to `debug` printed something useful!

Hopefully by writing this down I will remember or at least find the answer next time I manage to do this again!
