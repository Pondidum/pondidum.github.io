---
layout: post
title: Serilog LogContext with StructureMap and SimpleInjector
tags: net code structuremap simpleinjector di ioc
---

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

Most (if not all) the applications I build use a dependency injection container to build objects.  In my opinion there are only two containers which are worth considering in the .net space:  [StructureMap](http://structuremap.github.io/), and [SimpleInjector](https://simpleinjector.org).  If you like convention based registration, use StructureMap.  If you like typing/specifying everything yourself, use SimpleInjector.

### SimpleInjector

I couldn't work out how to make SimpleInjector do what I wanted for this which is unfortunate, as a large proportion of projects use this container.  Switching to StructureMap which handles this with ease doesn't seem worth the time taken.

Instead I will be asking around to see if this is something SimpleInjector could support in the future, or if it can do it and I am just not seeing how!

### StructureMap

I have the following two classes to test each get a decorated `ILogger`, along with an `ILogOwner` interface to allow me to do more generic testing:

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

And we can verify the behaviour with a parameterised test in XUnit:

```csharp
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
```

## Outcomes

Thanks to this container usage, I no longer have to have the `.ForContext(typeof(Something))` scattered throughout my codebases.

I also really want to do this within codebases which use SimpleInjector, so I'll be asking around, or coming up with another way of doing this.
