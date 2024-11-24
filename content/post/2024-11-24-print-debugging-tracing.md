+++
title = 'Print debugging: a tool among other tools'
tags = ['opentelemetry', 'testing', 'observability']
+++

This is my thoughts after reading [Don't Look Down on Print Debugging](https://blog.startifact.com/posts/print-debugging/).

## TLDR

Print debugging is a tool, and it has its uses - however, if there are better tools available, maybe use those instead.  For me, a better tool is OpenTelemetry [tracing](https://opentelemetry.io/docs/concepts/signals/traces/); it gives me high granularity, parent-child relationships between operations, timings, and is filterable and searchable.  I can also use it to debug [remote issues](#tracing-for-remote-debugging), as long as the user can send me a file.

## Print Debugging

The good thing about print debugging is that it is always available; it is very rare a process doesn't have a stdout (or stderr) available, or some kind of logging system.  The downside with print debugging is that it is very basic; anything more you want to do with it, such as timing data, has to be implemented.

One of the comments I read was, in essence:

> one place print debugging is best is dealing with threading/timing bugs, it lets you see what is happening when

I can understand this, but often print statements are buffered before hitting the terminal, so you might lose some order; you can solve that with a timestamp (assuming it's accurate enough), but as I mentioned earlier, you need to do that yourself.

## Use a Debugger?

The first debugger I used was in Visual Studio, and it was amazing.  I learned how to use a lot of features in it, such as automatic break-on-exception, conditional breakpoints, automatic counters ("break after hitting this line x times"), watches, and computed values.  This was at the beginning of my career, and neither I, nor the codebase, knew anything of automated testing.

Once I learned about automated testing, and then started writing tests, the amount of time I used the debugger reduced.  It was still a great tool, but tests were sometimes the quicker way to verify some behaviour.  Often times, once I had investigated a problem with the debugger, I would write a test to cover that situation - now the repeated debugger usage to know if the fix worked wasn't needed.

I don't often use a debugger these days.  It still has it's place in my toolkit, but in general, I lean more on OpenTelemetry and tracing in general.

## Tracing

My current go-to tool for debugging and application development is OpenTelemetry tracing.  I've written about why I [prefer tracing to Structured Logging](/2023/09/19/tracing-is-better/) before, but this is a bit different.

By default, tracing lets me see the relationships between function calls, the timing durations of each span (a span is a unit of execution; it could be a function call or a call into a library), and many structured attributes (both automatic and manually added.)

The downside is that you need some way to view that tracing data; using the stdout trace exporter is generally not useful - its far too noisy.  For local development I use [Grafana Tempo all-in-one](https://hub.docker.com/r/grafana/otel-lgtm), or [Otel-TUI](https://github.com/ymtdzzz/otel-tui).  For deployed environments, I still think [honeycomb](https://honeycomb.io) is the best placec for traces.

Most projects I interact with have a docker-compose file for all their local dependencies; adding a tracing viewer to that compose file is pretty trival, so the barrier to entry is really not that high:

```yaml
grafana:
  image: grafana/otel-lgtm
  ports:
  - "3000:3000"
  - "4317:4317"
```

## Tracing for Remote Debugging

I maintain some internal CLI applications, which make extensive use of tracing.  When a user has a problem, they can add a `--store-traces` flag to the CLI, and it will write all the tracing data for their run to a file on disk.  They can then send me that file, I can load it into a trace viewer, and use all the high-cardinality attributes and timing data to figure out where their problem is exactly.

## Do you ever use print debugging?

Not often, but it does happen sometimes.  As I said at the outset; its just a tool, and I can and do use it.  I just happen to prefer a different tool.
