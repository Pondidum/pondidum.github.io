---
layout: post
title: Serilog LogContext with StructureMap and SimpleInjector
tags: c# code structuremap simpleinjector di ioc
---

*This article has been updated after feedback from [.Net Junkie](https://twitter.com/dot_NET_Junkie) (Godfather of SimpleInjector).  I now have a working SimpleInjector implementation of this, and am very appreciative of him for taking the time to help me :)*

Serilog is one of the main set of libraries I use on a regular basis, and while it is great at logging, it does cause something in our codebase that I am less happy about.  Take the following snippet for example:

```csharp
public class Something
{
    private static readonly ILogger Log = Log.ForContext(typeof(Something));
}
```

There are two things I don't like about this.  The first is the static field access:  We have tests which assert on log content for disallowed information, or to include a correlationid etc.  Having a static field means that if tests run in parallel, we end up with flaky tests due to multiple log messages being written.  The second thing I don't like is less about the line itself, but the repetition of this line throughout the codebase.  Nearly every class which does logging has the same line, but with the type parameter changed.

I set out to see if I could remedy both problems at once.

## Fixing the Static Field

The first fix is to inject the logger in via a constructor argument, which will allow tests to use their own version of the logger:

```csharp
public class Something
{
    private readonly ILogger _log;

    public Something(ILogger logger)
    {
        _log = logger.ForContext(typeof(Something));
    }
}
```

That was easy! Now on to the hard part; removing the repeated `.ForContext` call.

## Fixing the ForContext Repetition

Most (if not all) the applications I build use a dependency injection container to build objects.  In my opinion there are only two containers which are worth considering in the .net space:  [StructureMap](http://structuremap.github.io/), and [SimpleInjector](https://simpleinjector.org).  If you like convention based registration, use StructureMap.  If you like to get a safety net that prevents and detects common misconfigurations, use SimpleInjector.

### Tests

We can use the same tests to verify the behaviour both when using StructureMap and SimpleInjector's.  We have a couple of test classes, and an interface to allow for more generic testing:

```csharp
private interface ILogOwner
{
    ILogger Logger { get; }
}

private class Something : ILogOwner
{
    public ILogger Logger { get; }

    public Something(ILogger logger)
    {
        Logger = logger;
    }
}

private class Everything : ILogOwner
{
    public ILogger Logger { get; }

    public Everything(ILogger logger)
    {
        Logger = logger;
    }
}
```

And then a single parameterised test method for verification:

```csharp
public class Tests
{
    private readonly Container _container;

    public Tests()
    {
        Log.Logger = new LoggerConfiguration()
            .MinimumLevel.Debug()
            .WriteTo.Console()
            .CreateLogger();

        // _container = new ...
    }

    [Theory]
    [InlineData(typeof(Something))]
    [InlineData(typeof(Everything))]
    public void Types_get_their_own_context(Type type)
    {
        var instance = (ILogOwner)_container.GetInstance(type);
        var context = GetContextFromLogger(instance);

        context.ShouldBe(type.FullName);
    }

    private static string GetContextFromLogger(ILogOwner owner)
    {
        var logEvent = CreateLogEvent();
        owner.Logger.Write(logEvent);
        return logEvent.Properties["SourceContext"].ToString().Trim('"');
    }

    private static LogEvent CreateLogEvent() => new LogEvent(
        DateTimeOffset.Now,
        LogEventLevel.Debug,
        null,
        new MessageTemplate("", Enumerable.Empty<MessageTemplateToken>()),
        Enumerable.Empty<LogEventProperty>());
}
```


### StructureMap

The StructureMap initialisation just requires a single line change to use the construction context when creating a logger:

```csharp
_container = new Container(_ =>
{
    _.Scan(a =>
    {
        a.TheCallingAssembly();
        a.WithDefaultConventions();
    });

    // original:
    // _.For<ILogger>().Use(context => Log.Logger);

    // contextual
    _.For<ILogger>().Use(context => Log.ForContext(context.ParentType));
});
```


### SimpleInjector

SimpleInjector does a lot of verification of your container configuration, and as such deals mostly with Types, rather than instances, or types which have multiple mappings as we are doing.  This makes it slightly harder to support the behaviour we had with StructureMap, but not impossible.  A huge thanks to .Net Junkie for assisting with this!

First we need to create an implementation of  `IDependencyInjectionBehavior`, which will handle our `ILogger` type requests, and pass all other types requests to the standard implementation:

```csharp
class SerilogContextualLoggerInjectionBehavior : IDependencyInjectionBehavior
{
    private readonly IDependencyInjectionBehavior _original;
    private readonly Container _container;

    public SerilogContextualLoggerInjectionBehavior(ContainerOptions options)
    {
        _original = options.DependencyInjectionBehavior;
        _container = options.Container;
    }

    public void Verify(InjectionConsumerInfo consumer) => _original.Verify(consumer);

    public InstanceProducer GetInstanceProducer(InjectionConsumerInfo i, bool t) =>
        i.Target.TargetType == typeof(ILogger)
            ? GetLoggerInstanceProducer(i.ImplementationType)
            : _original.GetInstanceProducer(i, t);

    private InstanceProducer<ILogger> GetLoggerInstanceProducer(Type type) =>
        Lifestyle.Transient.CreateProducer(() => Log.ForContext(type), _container);
}
```

This can then be set in our container setup:

```csharp
_ontainer = new Container();
_container.Options.DependencyInjectionBehavior = new SerilogContextualLoggerInjectionBehavior(_container.Options);

_container.Register<Something>();
_container.Register<Everything>();
```

And now our tests pass!

## Outcomes

Thanks to this container usage, I no longer have to have the `.ForContext(typeof(Something))` scattered throughout my codebases.

Hopefully this shows how taking away just some of the little tasks makes life easier - I now no longer have to remember to do the `.ForContext` on each class, and don't need to have tests to validate it is done on each class (I have one test in my container configuration tests which validates this behaviour instead).
