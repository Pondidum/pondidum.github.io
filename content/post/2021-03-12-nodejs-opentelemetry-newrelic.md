---
date: "2021-03-12T00:00:00Z"
tags: ["observability", "opentelemetry", "nodejs", "zipkin"]
title: Getting NodeJS OpenTelemetry data into NewRelic
---

I had the need to get some OpenTelemetry data out of a NodeJS application, and into NewRelic's distributed tracing service, but found that there is no way to do it directly, and in this use case, adding a separate collector is more hassle than it's worth.

Luckily, there is an NodeJS [OpenTelemetry library which can report to Zipkin](https://www.npmjs.com/package/@opentelemetry/exporter-zipkin), and NewRelic can also [ingest Zipkin format data](https://docs.newrelic.com/docs/understand-dependencies/distributed-tracing/trace-api/report-zipkin-format-traces-trace-api/).

To use it was relatively straight forward:

```js
import { context, setSpan, Span, trace } from "@opentelemetry/api";
import { BasicTracerProvider, BatchSpanProcessor } from "@opentelemetry/tracing";
import { ZipkinExporter } from "@opentelemetry/exporter-zipkin";

const exporter = new ZipkinExporter({
  url: "https://trace-api.newrelic.com/trace/v1",
  serviceName: "interesting-service",
  headers: {
    "Api-Key": process.env.NEWRELIC_APIKEY,
    "Data-Format": "zipkin",
    "Data-Format-Version": "2",
  },
});

const provider = new BasicTracerProvider();
provider.addSpanProcessor(new BatchSpanProcessor(exporter));
provider.register();

export const tracer = trace.getTracer("default");


const rootSpan = tracer.startSpan("main");

// do something fantastically interesting

rootSpan.end();
provider.shutdown();
```

This has the added benefit of being able to test with Zipkin locally, using the `openzipkin/zipkin-slim` docker container, by just removing the URL property from the `ZipkinExporter`:

```bash
docker run --rm -d -p 9411:9411 openzipkin/zipkin-slim
```

## Child Spans

Figuring out how to create child spans was actually harder in the end, in part because the OpenTelemetry docs don't quite match the actual function signatures.

In the end, I wrote this little helper function:

```js
import { context, setSpan, Span } from "@opentelemetry/api";

function startSpan(parent: Span, name: string): Span {
  return tracer.startSpan(name, undefined, setSpan(context.active(), parent));
}
```

Which I can use like this:

```js

async function DoInterestingThings(span: Span) {
  span = startSpan(span, "do-interesting-things");

  // interesting things happen here

  span.end();
}
```

Doing both of these means I can now see what my misbehaving cron jobs are actually doing, rather than trying to guess what their problems are.
