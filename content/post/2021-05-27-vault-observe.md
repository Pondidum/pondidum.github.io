---
date: "2021-05-27T00:00:00Z"
tags: ["observability", "honeycomb", "opentelemetry", "infrastructure", "vault"]
title: Adding Observability to Vault
---

One of the things I like to do when setting up a Vault cluster is to visualise all the operations Vault is performing, which helps see usage patterns changing, whether there are lots of failed requests coming in, and what endpoints are receiving the most traffic.

While Vault has a lot of data available in Prometheus telemetry, the kind of information I am after is best taken from the Audit backend.  Setting up an audit backend for Vault is reasonably easy - it supports three methods of communication: file, socket and syslog.  For this application, I use a Unix socket and a small daemon running on the same machine as the Vault instance to send the data to a tracing system.

## The Goal

Write a small application that receives audit events and writes traces (spans) to an observability tool.   In this case, I am implementing both Honeycomb and Zipkin via OpenTelemetry.

The [code is available on Github](https://github.com/Pondidum/vault-observe), and the most interesting parts are covered in the rest of this blog post.

## Receiving and Processing Messages

```go
ln, _ := net.Listen("unix", "/tmp/observe.sock")
conn, _ := ln.Accept()

for {
  message, _ := bufio.NewReader(conn).ReadBytes('\n')

  // do something with the message
}
```

We only need to do minimal processing of the data for this application before sending it on to Honeycomb or Zipkin.  As the messages contain nested objects, we need to flatten the object hierarchy for easier viewing in spans.  So instead of this:


```json
{
  "request": {
    "operation": "update",
    "namespace": { "id": "root" },
    "path": "sys/audit/socket",
    "data": {
      "local": false
    }
  }
}
```

We want to send this:

```json
{
  "request.operation": "update",
  "request.namespace.id": "root",
  "request.path": "sys/audit/socket",
  "request.data.local": false
}
```

We also want to get a few strongly typed pieces of data out of the message, too, such as the `type` (`request` or `response`) and the request's `id`, which is in both messages and can be used to group the spans.

To save us from deserialising the json twice, we can do the following:

1. deserialize into a `map[string]interface{}`
2. create a flattened version of the event using the [flatten](https://pkg.go.dev/github.com/jeremywohl/flatten) library
3. turn the map into a typed struct using the [mapstructure](https://pkg.go.dev/github.com/mitchellh/mapstructure) library



```go
// 1 deserialize
event := map[string]interface{}{}
if err := json.Unmarshal(message, &event); err != nil {
  return err
}

// 2 flatten
flat, err := flatten.Flatten(event, "", flatten.DotStyle)
if err != nil {
  return err
}

// 3 type
typed := Event{}
if err := mapstructure.Decode(event, &typed); err != nil {
  return err
}
```

Now that we have our flattened version and our typed version of the message, we can forward it to our span processors.  There are two implementations (3 if you count `stdout`), so let's look at them one at a time.


## Honeycomb

To send the spans to Honeycomb, I am using their lower-level library [libhoney-go](https://pkg.go.dev/github.com/honeycombio/libhoney-go), rather than the more usual [beeline](https://pkg.go.dev/github.com/honeycombio/beeline-go) as I don't need all the `context` propagation or automatic ID generation.

For the first version of this application, just sending the two events to Honeycomb linked together is enough; however, both spans will show  0ms durations.  We'll fix this problem for both Honeycomb and OpenTelemetry later.

To link our spans together properly, I use the `.Request.ID` property from the event as the `trace.trace_id`; it's already a guid and is the same for both the request and response events.  Then, for a `request` event, I make it the parent span by using the `.Request.ID` again, but this time as the `trace.span_id`.  Finally, for the `response` event, I set the `trace.parent_id` to the `.Request.ID`, and generate a random value for the `trace.span_id` field.

Lastly, I loop through the flattened version of the event, adding each key-value pair to the event's attributes and finally send the event.

```go
ev := libhoney.NewEvent()
ev.AddField("trace.trace_id", typed.Request.ID)

if typed.Type == "request" {
  ev.AddField("trace.span_id", typed.Request.ID)
} else {
  ev.AddField("trace.parent_id", typed.Request.ID)
  ev.AddField("trace.span_id", generateSpanID())
}

ev.AddField("service_name", "vault")
ev.AddField("name", typed.Type)

for key, val := range event {
  ev.AddField(key, val)
}

ev.Send()
```

## Zipkin / OpenTelemetry

The process for sending via OpenTelemetry is reasonably similar; we start a new span, copy the flattened structure into the span's attributed and call `End()`, making the TracerProvider send the span to our configured backends (Zipkin in this case.)


```go
id, _ := uuid.Parse(typed.Request.ID)
ctx := context.WithValue(context.Background(), "request_id", id)

tr := otel.GetTracerProvider().Tracer("main")
ctx, span := tr.Start(ctx, typed.Type, trace.WithSpanKind(trace.SpanKindServer))

for key, value := range event {
  span.SetAttributes(attribute.KeyValue{
    Key:   attribute.Key(key),
    Value: attribute.StringValue(fmt.Sprintf("%v", value)),
  })
}

if typed.Error != "" {
  span.SetStatus(codes.Error, typed.Error)
}

span.End()
```

The hard part was figuring out how to feed the `.Request.ID` into the Tracer as the TraceID, which was achieved by configuring OpenTelemetry with a custom ID generator that would use the `request_id` property of the current `context`:

```go
type Generator struct{}

func (g *Generator) NewIDs(ctx context.Context) (trace.TraceID, trace.SpanID) {
  val := ctx.Value("request_id").(uuid.UUID)
  tid := trace.TraceID{}
  req, _ := val.MarshalText()
  copy(tid[:], req)

  sid := trace.SpanID{}
  rand.Read(sid[:])

  return tid, sid
}
```

I am sure more copying and allocation is happening in this method than necessary, but it is good enough for now.  Configuring it for use by OpenTelemetry is straightforward; it just needs adding to the `NewTracerProvider` call by wrapping it with `trace.WithIDGenerator()`:

```go
exporter, _ := zipkin.NewRawExporter(
  "http://localhost:9411/api/v2/spans",
  zipkin.WithSDKOptions(sdktrace.WithSampler(sdktrace.AlwaysSample())),
)

processor := sdktrace.NewSimpleSpanProcessor(exporter)

tp := sdktrace.NewTracerProvider(
  sdktrace.WithSpanProcessor(processor),
  sdktrace.WithResource(resource.NewWithAttributes(
    semconv.ServiceNameKey.String("vault-observe"),
  )),
  sdktrace.WithIDGenerator(&Generator{}),
)

otel.SetTracerProvider(tp)
```

## Testing

To verify that it works, I have a single `docker-compose.yml` file which sets up a Vault instance in dev mode, and a Zipkin instance.  It mounts the current working directory into the Vault container as `/sockets` to share the socket file between the host and the container.

```yaml
version: "3.9"

services:
  vault:
    image: vault:latest
    cap_add:
      - IPC_LOCK
    volumes:
      - "./:/sockets:rw"
    ports:
      - "8200:8200"
    environment:
      VAULT_DEV_ROOT_TOKEN_ID: "vault"
  zipkin:
    image: openzipkin/zipkin-slim
    ports:
      - "9411:9411"
```

Running the application along with the docker container is now as follows:

```bash
go build
docker-compose up -d
./vault-observe --zipkin --socket-path observe.sock
```

In another terminal, you can now enable the new audit backend and send some requests so we can look at them in Zipkin:

```bash
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="vault"

vault audit enable socket address=/sockets/observe.sock socket_type=unix

vault secrets enable -version=2 kv
vault kv put /secrets/test name=andy
vault kv get /secrets/test
```

## Running in Production

There are a few things you should be aware of, running this in production:

- This _must not_ be your only audit backend: Vault will fail requests if they are not successfully written to at least one audit backend if any are enabled.
- There is the possibility of losing data if the `vault-observe` process stops

## Improvements

As I am using this for keeping an eye on request durations and patterns in behaviour, capturing the actual time it takes for Vault to handle a request would be pretty valuable.  So instead of processing both events, I will keep just the timestamp from the `request`, and then when the `response` event comes in, look up the timestamp and calculate the duration.

As I don't want an ever-expanding list of timestamps in memory, I use an [automatically expiring cache](https://pkg.go.dev/github.com/patrickmn/go-cache) so keep them for around 10 seconds, as no request to Vault should be that slow!

```go
requests := cache.New(10*time.Second, 1*time.Minute)

for {
  err := processMessage(requests, conn, sender)
  if err != nil && err != io.EOF {
    fmt.Println(err)
  }
}
```

The `processMessage` function now handles the `request` and `response` messages separately.  The `request` just inserts the event's `time` property into the cache, and exists:

```go
if typed.Type == "request" {
  requests.Set(typed.Request.ID, typed.Time, cache.DefaultExpiration)
  return nil
}
```

The `response`  version pulls the time back out of the cache and stores it into the event itself - it's then up to the sender if it wants to use the value or not.

```go
if typed.Type == "response" {

  if x, found := requests.Get(typed.Request.ID); found {
    typed.StartTime = x.(time.Time)
    requests.Delete(typed.Request.ID)
  } else {
    return fmt.Errorf("No request found in the cache for %s", typed.Request.ID)
  }
}
```

In the Honeycomb sender, we can remove all the parenting logic; we only need to set the `Timestamp` and `duration_ms` fields to get the duration showing correctly:

```go
duration := typed.Time.Sub(typed.StartTime).Milliseconds()

ev := libhoney.NewEvent()
ev.Timestamp = typed.StartTime
ev.AddField("duration_ms", duration)

ev.AddField("trace.trace_id", typed.Request.ID)
ev.AddField("trace.span_id", typed.Request.ID)
```


For the OpenTelemetry sender, we can add a `trace.WithTimestamp()` call to both the `Start()` and `End()` calls so use our events' timestamps:

```go
ctx := context.WithValue(context.Background(), "request_id", id)
tr := otel.GetTracerProvider().Tracer("main")
ctx, span := tr.Start(ctx, typed.Type, trace.WithSpanKind(trace.SpanKindServer), trace.WithTimestamp(typed.StartTime))

// ...


span.End(trace.WithTimestamp(typed.Time))
```
