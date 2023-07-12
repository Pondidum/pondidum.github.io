+++
date = '2017-10-11T00:00:00Z'
tags = ['masstransit', 'rabbitmq', 'testing']
title = 'Testing RabbitMQ Concurrency in MassTransit'

+++

We have a service which consumes messages from a [RabbitMQ](http://www.rabbitmq.com/) queue - for each message, it makes a few http calls, collates the results, does a little processing, and then pushes the results to a 3rd party api.  One of the main benefits to having this behind a queue is our usage pattern - the queue usually only has a few messages in it per second, but periodically it will get a million or so messages within 30 minutes (so from ~5 messages/second to ~560 messages/second.)

Processing this spike of messages takes ages, and while this service is only on a `T2.Medium` machine (2 CPUs, 4GB Memory), it only uses 5-10% CPU while processing the messages, which is clearly pretty inefficient.

We use [MassTransit](http://masstransit-project.com/) when interacting with RabbitMQ as it provides us with a lot of useful features, but by default sets the amount of messages to be processed in parallel to `Environment.ProcessorCount * 2`.  For this project that means 4 messages, and as the process is IO bound, it stands to reason that we could increase that concurrency a bit. Or a lot.

The existing MassTransit setup looks pretty similar to this:

```csharp
_bus = Bus.Factory.CreateUsingRabbitMq(rabbit =>
{
    var host = rabbit.Host(new Uri("rabbitmq://localhost"), h =>
    {
        h.Username("guest");
        h.Password("guest");
    });

    rabbit.ReceiveEndpoint(host, "SpikyQueue", endpoint =>
    {
        endpoint.Consumer(() => new TestConsumer());
    });
});
```

## The Test (Driven Development)

As we like testing things, I wrote a test to validate the degree of concurrency we have.  We use a real instance of RabbitMQ ([Started with Docker, as part of the build](/2017/10/02/dotnet-core-docker-integration-tests/)), and have a test message and consumer.  Due to the speed of RabbitMQ delivery, we make the consumer just take a little bit of time before returning:

```csharp
class TestMessage
{
    public int Value { get; set; }
}

class TestConsumer : IConsumer<TestMessage>
{
    public async Task Consume(ConsumeContext<TestMessage> context)
    {
        await Task.Delay(600);
    }
}
```

The final piece of our puzzle is an `IConsumeObserver`, which will count the number of messages processed in parallel, as well as the total number of messages processed.  We will use the total number of messages to know when our test can stop running, and the parallel number to prove if our concurrency changes worked.

What this observer is doing is the following, but as we are in a multithreaded environment, we need to use the `Interlocked` class, and do a bit more work to make sure we don't lose values:

```
PreConsume:
    currentPendingDeliveryCount++
    maxPendingDeliveryCount = Math.Max(maxPendingDeliveryCount, currentPendingDeliveryCount)
PostConsume:
    currentPendingDeliveryCount--
```

The actual `ConsumeCountObserver` code is as follows:

```csharp
class ConsumeCountObserver : IConsumeObserver
{
    int _deliveryCount;
    int _currentPendingDeliveryCount;
    int _maxPendingDeliveryCount;

    readonly int _messageCount;
    readonly TaskCompletionSource<bool> _complete;

    public ConsumeCountObserver(int messageCount)
    {
        _messageCount = messageCount;
        _complete = new TaskCompletionSource<bool>();
    }

    public int MaxDeliveryCount => _maxPendingDeliveryCount;
    public async Task Wait() => await _complete.Task;

    Task IConsumeObserver.ConsumeFault<T>(ConsumeContext<T> context, Exception exception) => Task.CompletedTask;

    Task IConsumeObserver.PreConsume<T>(ConsumeContext<T> context)
    {
        Interlocked.Increment(ref _deliveryCount);

        var current = Interlocked.Increment(ref _currentPendingDeliveryCount);
        while (current > _maxPendingDeliveryCount)
            Interlocked.CompareExchange(ref _maxPendingDeliveryCount, current, _maxPendingDeliveryCount);

        return Task.CompletedTask;
    }

    Task IConsumeObserver.PostConsume<T>(ConsumeContext<T> context)
    {
        Interlocked.Decrement(ref _currentPendingDeliveryCount);

        if (_deliveryCount == _messageCount)
            _complete.TrySetResult(true);

        return Task.CompletedTask;
    }
}
```

Finally, we can put the actual test together:  We publish some messages, connect the observer, and start processing.  Finally, when the observer indicates we have finished, we assert that the `MaxDeliveryCount` was the same as the `ConcurrencyLimit`:

```csharp
[Test]
public async Task WhenTestingSomething()
{
    for (var i = 0; i < MessageCount; i++)
        await _bus.Publish(new TestMessage { Value = i });

    var observer = new ConsumeCountObserver(MessageCount);
    _bus.ConnectConsumeObserver(observer);

    await _bus.StartAsync();
    await observer.Wait();
    await _bus.StopAsync();

    observer.MaxDeliveryCount.ShouldBe(ConcurrencyLimit);
}
```

## The Problem

The problem we had was actually increasing the concurrency:  There are two things you can change, `.UseConcurrencyLimit(32)` and `.PrefetchCount = 32`, but doing this doesn't work:

```csharp
_bus = Bus.Factory.CreateUsingRabbitMq(rabbit =>
{
    var host = rabbit.Host(new Uri("rabbitmq://localhost"), h =>
    {
        h.Username("guest");
        h.Password("guest");
    });

    rabbit.ReceiveEndpoint(host, "SpikeyQueue", endpoint =>
    {
        endpoint.UseConcurrencyLimit(ConcurrencyLimit);
        endpoint.PrefetchCount = (ushort) ConcurrencyLimit;

        endpoint.Consumer(() => new TestConsumer());
    });
});
```

Or well...it does work, if the `ConcurrencyLimit` is **less** than the default.  After a lot of trial and error, it turns out there are not two things you can change, but four:

* `rabbit.UseConcurrencyLimit(val)`
* `rabbit.PrefetchCount = val`
* `endpoint.UseConcurrencyLimit(val)`
* `endpoint.PrefetchCount = val`

This makes sense (kind of): You can set limits on the factory, and then the endpoints can be any value less than or equal to the factory limits.  My process of trial and error to work out which needed to be set:

1. Set them all to 32
2. Run test
    * if it passes, remove one setting, go to 2.
    * if it fails, add last setting back, remove a different setting, go to 2.

After iterating this set of steps for a while, it turns out for my use case that I need to set `rabbit.UseConcurrencyLimit(val)` and `endpoint.PrefetchCount = val`:

```csharp
_bus = Bus.Factory.CreateUsingRabbitMq(rabbit =>
{
    var host = rabbit.Host(new Uri("rabbitmq://localhost"), h =>
    {
        h.Username("guest");
        h.Password("guest");
    });

    rabbit.UseConcurrencyLimit(ConcurrencyLimit);
    rabbit.ReceiveEndpoint(host, "SpikeyQueue", endpoint =>
    {
        endpoint.PrefetchCount = (ushort) ConcurrencyLimit;
        endpoint.Consumer(() => new TestConsumer());
    });
});
```

Interestingly, no matter which place you set the `PrefetchCount` value, it doesn't show up in the RabbitMQ web dashboard.

Hope this might help someone else struggling with getting higher concurrency with MassTransit.