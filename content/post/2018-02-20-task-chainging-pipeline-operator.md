---
layout: post
title: Task Chaining and the Pipeline Operator
tags: c#
---

Since I have been trying to learn a functional language (Elixir), I have noticed how grating it is when in C# I need to call a few methods in a row, passing the results of one to the next.

The bit that really grates is that it reads backwards, i.e. the rightmost function call is invoked first, and the left hand one last, like so:

```csharp
await WriteJsonFile(await QueueParts(await ConvertToModel(await ReadBsxFile(record))));
```

In Elixir (or F# etc.) you can write this in the following way:

```csharp
var task = record
    |> await ReadBsxFile
    |> await ConvertToModel
    |> await QueueParts
    |> await WriteJsonFile
```

While there are proposals for the [forward pipe operator](https://github.com/dotnet/csharplang/issues/74) to be added to C# being discussed, it doesn't look like it will happen in the near future.

Something close to this is Linq, and at first, I tried to work out a way to write the pipeline for a single object using the `Select` statement, something like this:

```csharp
await record
    .Select(ReadBsxFile)
    .Select(ConvertToModel)
    .Select(QueueParts)
    .Select(WriteJsonFile);
```

The problem with this is that Linq doesn't play well with async code - you end up needing to call `.Result` on each task selected...which is a [Bad](http://blog.stephencleary.com/2012/07/dont-block-on-async-code.html) [Thing](https://msdn.microsoft.com/en-us/magazine/jj991977.aspx) to do.

I realised that as it's just `Task`s I really care about, I might be able to write some extension methods to accomplish something similar.  I ended up with 3 extensions: one to start a chain from a value, and two to allow either `Task<T>` to be chained, or a `Task`:

```csharp
public static class TaskExtensions
{
    public static async Task<TOut> Start<TIn, TOut>(this TIn value, Func<TIn, Task<TOut>> next)
    {
        return await next(value);
    }

    public static async Task<TOut> Then<TIn, TOut>(this Task<TIn> current, Func<TIn, Task<TOut>> next)
    {
        return await next(await current);
    }

    public static async Task Then<TIn>(this Task<TIn> current, Func<TIn, Task> next)
    {
        await next(await current);
    }
}
```

This can be used to take a single value, and "pipeline" it through a bunch of async methods:

```csharp
var task = record
    .Start(ReadBsxFile)
    .Then(ConvertToModel)
    .Then(QueueParts)
    .Then(WriteJsonFile);
```

One of the nice things about this is that if I want to add another method in the middle of my chain, as long as it's input and output types fit, it can just be inserted or added to the chain:

```csharp
var task = record
    .Start(ReadBsxFile)
    .Then(ConvertToModel)
    .Then(InspectModelForRedundancies)
    .Then(QueueParts)
    .Then(WriteJsonFile)
    .Then(DeleteBsxFile);
```

You can see a real use of this in my [BsxProcessor Lambda](https://github.com/Pondidum/BrickRecon/blob/master/projects/BsxProcessor/src/BsxProcessor/RecordHandler.cs#L24).

This is one of the great things about learning other programming languages: even if you don't use them on a daily basis, they can really give you insight into different ways of doing things, doubly so if they are a different style of language.
