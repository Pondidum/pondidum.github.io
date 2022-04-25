---
date: "2017-10-04T00:00:00Z"
tags: structuremap di ioc
title: Composite Decorators with StructureMap
---

While I was developing my [Crispin](https://github.com/pondidum/crispin) project, I ended up needing to create a bunch of implementations of a single interface, and then use all those implementations at once (for metrics logging).

The interface looks like so:

```csharp
public interface IStatisticsWriter
{
    Task WriteCount(string format, params object[] parameters);
}
```

And we have a few implementations already:

* LoggingStatisticsWriter - writes to an `ILogger` instance
* StatsdStatisticsWriter - pushes metrics to [StatsD](https://github.com/etsy/statsd)
* InternalStatisticsWriter - aggregates metrics for exposing via Crispin's api

To make all of these be used together, I created a fourth implementation, called `CompositeStatisticsWriter` (a name I made up, but apparently matches the [Gang of Four definition](https://en.wikipedia.org/wiki/Composite_pattern) of a composite!)

```csharp
public class CompositeStatisticsWriter : IStatisticsWriter
{
    private readonly IStatisticsWriter[] _writers;

    public CompositeStatisticsWriter(IEnumerable<IStatisticsWriter> writers)
    {
        _writers = writers.ToArray();
    }

    public async Task WriteCount(string format, params object[] parameters)
    {
        await Task.WhenAll(_writers
            .Select(writer => writer.WriteCount(format, parameters))
            .ToArray());
    }
}
```

The problem with doing this is that StructureMap throws an error about a bi-directional dependency:

```csharp
StructureMap.Building.StructureMapBuildException : Bi-directional dependency relationship detected!
Check the StructureMap stacktrace below:
1.) Instance of Crispin.Infrastructure.Statistics.IStatisticsWriter (Crispin.Infrastructure.Statistics.CompositeStatisticsWriter)
2.) All registered children for IEnumerable<IStatisticsWriter>
3.) Instance of IEnumerable<IStatisticsWriter>
4.) new CompositeStatisticsWriter(*Default of IEnumerable<IStatisticsWriter>*)
5.) Crispin.Infrastructure.Statistics.CompositeStatisticsWriter
6.) Instance of Crispin.Infrastructure.Statistics.IStatisticsWriter (Crispin.Infrastructure.Statistics.CompositeStatisticsWriter)
7.) Container.GetInstance<Crispin.Infrastructure.Statistics.IStatisticsWriter>()
```

After attempting to solve this myself in a few different ways (you can even [watch the stream](https://www.youtube.com/watch?v=2N6cgMBN7ZA) of my attempts), I asked in the StructreMap gitter chat room, and received this answer:

> This has come up a couple times, and yeah, you’ll either need a custom convention or a policy that adds the other `ITest`’s to the instance for `CompositeTest` as inline dependencies so it doesn’t try to make Composite a dependency of itself
> -- <cite>Jeremy D. Miller</cite>

Finally, Babu Annamalai provided a simple implementation when I got stuck (again).

The result is the creation of a custom convention for registering the composite, which provides all the implementations I want it to wrap:

```csharp
public class CompositeDecorator<TComposite, TDependents> : IRegistrationConvention
    where TComposite : TDependents
{
    public void ScanTypes(TypeSet types, Registry registry)
    {
        var dependents = types
            .FindTypes(TypeClassification.Concretes)
            .Where(t => t.CanBeCastTo<TDependents>() && t.HasConstructors())
            .Where(t => t != typeof(TComposite))
            .ToList();

        registry
            .For<TDependents>()
            .Use<TComposite>()
            .EnumerableOf<TDependents>()
            .Contains(x => dependents.ForEach(t => x.Type(t)));
    }
}
```

To use this the StructureMap configuration changes from this:

```csharp
public CrispinRestRegistry()
{
    Scan(a =>
    {
        a.AssemblyContainingType<Toggle>();
        a.WithDefaultConventions();
        a.AddAllTypesOf<IStatisticsWriter>();
    });

    var store = BuildStorage();

    For<IStorage>().Use(store);
    For<IStatisticsWriter>().Use<CompositeStatisticsWriter>();
}
```

To this version:

```csharp
public CrispinRestRegistry()
{
    Scan(a =>
    {
        a.AssemblyContainingType<Toggle>();
        a.WithDefaultConventions();
        a.Convention<CompositeDecorator<CompositeStatisticsWriter, IStatisticsWriter>>();
    });

    var store = BuildStorage();
    For<IStorage>().Use(store);
}
```

And now everything works successfully, and I have Pull Request open on StructureMap's repo with an update to the documentation about this.

Hopefully this helps someone else too!