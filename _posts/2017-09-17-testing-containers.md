---
layout: post
title: Testing Containers or Test Behaviour, Not Implementation
tags: design structuremap testing
---

The trouble with testing containers is that usually the test ends up very tightly coupled to the implementation.

Let's see an example.  If we start off with an interface and implementation of a "cache", which in this case is just going to store a single string value.

```csharp
public interface ICache
{
    string Value { get; set; }
}

public class Cache
{
    public string Value { get; set; }
}
```

We then setup our container ([StructureMap](http://structuremap.github.io) in this case) to return the same instance of the cache whenever an `ICache` is requested:
```csharp
var container = new Container(_ =>
{
    _.For<ICache>().Use<Cache>().Singleton();
});
```

The following test is fairly typical of how this behaviour gets verified - it just compares that the same instance was returned by the container:

```csharp
var first = container.GetInstance<ICache>();
var second = container.GetInstance<ICache>();

first.ShouldBe(second);
```

But this is a very brittle test, as it is assuming that `ICache` will actually be the singleton.  However in the future, we might add in a decorator, or make the cache a totally different style of implementation which isn't singleton based.

For example, if we were to include a decorator class, which just logs reads and writes to the console:

```csharp
public class LoggingCache : ICache
{
    private readonly Cache _backingCache;

    public LoggingCache(Cache backingCache)
    {
        _backingCache = backingCache;
    }

    public string Value
    {
        get
        {
            Console.WriteLine("Value fetched");
            return _backingCache.Value;
        }
        set
        {
            Console.Write($"Value changed from {_backingCache.Value} to {value}");
            _backingCache.Value = value;
        }
    }
}
```

Which will change our container registration:

```csharp
var container = new Container(_ => {
    _.ForSingletonOf<Cache>();
    _.For<ICache>().Use<LoggingCache>();
});
```

The test will now fail, or need changing to match the new implementation.  This shows two things:
* Tests are tightly coupled to the implementation
* Tests are testing the implementation, not the intent.

## Testing intent, not implementation

Instead of checking if we get the same class instances back from the container, it would make for more sense to check the classes *behave* as expected.  For my "super stupid cache" example this could take the following form:

```csharp
var first = container.GetInstance<ICache>();
var second = container.GetInstance<ICache>();

first.Value = "testing";
second.Value.ShouldBe("testing");
```

Not only does this test validate the behaviour of the classes, but it is far less brittle - we can change what the container returns entirely for `ICache`, as long as it behaves the same.

But what do you think? How do you go about testing behaviour?
