---
layout: post
title: SOLID Principles - ISP
tags: design code c# solid

---

## Interface Segregation Principle

[Single Responsibility][blog-solid-srp] | [Open Closed][blog-solid-ocp] | [Liskov Substitution][blog-solid-lsp] | [Interface Segregation][blog-solid-isp] | [Dependency Inversion][blog-solid-dip]

Interface Segregation I find is often ignored, or people tend not to see the point in.  Segregating your Interfaces is a very useful way of reducing compexity in your systems, and comes with a number of benefits, such as making mocking inputs easier, and making your objects smaller and simpler.

So as usual, lets start off with an set of types which don't adhere to the principle.  Starting off, we have the following interface, which we are using to write data access classes with:

```csharp
public interface IEntity
{
	Guid ID { get; }
	void Save();
	void Load();
}
```

And a class which implements the interface:

```csharp
public class Entity : IEntity
{
	public Guid ID { get; private set; }

	public void Save()
	{
		Database.Save(this);
	}

	public void Load()
	{
		using (var reader = Database.Load(ID))
		{
			ID = reader.GetGuid(0);
			Read(reader);
		}
	}

	protected virtual void Read(IDataReader reader)
	{
		//nothing in the base
	}
}
```

At first glance, this seems like a pretty reasonable Entity, it doesn't have multiple responsibilities, and it is very simple. However, when we bring the second implementation of `IEntity` into the mix, it becomes more clear that some segregation would be useful:

```csharp
public class ReadOnlyEntity : IEntity
{
	public Guid ID { get; private set; }

	public void Save()
	{
		//do nothing
	}

	public void Load()
	{
		using (var reader = Database.Load(ID))
		{
			ID = reader.GetGuid(0);
			Read(reader);
		}
	}

	protected virtual void Read(IDataReader reader)
	{
		//nothing in the base
	}
}
```

Why would a `ReadOnlyEntity` need a `Save()` method? What happens if you have a collection of data which gets loaded from your database, but never gets saved back (a list of countries and associated data for example.)  Also, consumers of the `IEntity` interface get more access to methods than they need, for example the `Database` class being used here:

```csharp
public class Database
{
	public static void Save(IEntity entity)
	{
		entity.Load();	//?
	}
}
```

From looking at our usages of our entities, we can see there are two specific roles: something that can be loaded, and something that can be saved.  We start our separation by inheriting our existing interface:

```csharp
public interface IEntity : ISaveable, ILoadable
{
}

public interface ISaveable
{
	Guid ID { get; }
	void Save();
}

public interface ILoadable
{
	Guid ID { get; }
	void Load();
}
```

Here we have pulled the method and properties relevant for saving into one interface, and the methods and properties relevant to loading into another.  By making `IEntity` inherit both `ISaveable` and `ILoadable`, we have no need to change any existing code yet.

Our next step is to change usages of `IEntity` to take in the more specific interface that they require:

```csharp
public class Database
{
	public static void Save(ISaveable entity)
	{
	}
}
```

Once this is done, we can remove the `IEntity` interface, and update our implementations to use `ISaveable` and `ILoadable` instead:

```csharp
public class Entity : ISaveable, ILoadable
{
	public Guid ID { get; private set; }

	public void Save()
	{
		Database.Save(this);
	}

	public void Load()
	{
		using (var reader = Database.Load(ID))
		{
			ID = reader.GetGuid(0);
			Read(reader);
		}
	}

	protected virtual void Read(IDataReader reader)
	{
		//nothing in the base
	}
}

public class ReadOnlyEntity : ILoadable
{
	public Guid ID { get; private set; }

	public void Load()
	{
		using (var reader = Original.Database.Load(ID))
		{
			ID = reader.GetGuid(0);
			Read(reader);
		}
	}

	protected virtual void Read(IDataReader reader)
	{
		//nothing in the base
	}
}
```

Now our objects are showing specifically what they are capable of - the `ReadOnlyEntity` doesn't have a `Save()` method which you are not supposed to call!

If you do have a method which requires an object which is both an `ISaveable` and an `ILoadable`, rather than pass in the same object to two parameters, you can achieve it with a generic parameter:

```csharp
public void DoSomething<T>(T entity) where T : ISaveable, ILoadable
{
	entity.Save();
	entity.Load();
}
```

Hopefully this shows the reasoning of segregating your interfaces and the steps to segregate existing interfaces.

All source code is available on my Github: [Solid.Demo Source Code][solid-demo-repo]

[blog-solid-srp]: http://andydote.co.uk/solid-principles-srp
[blog-solid-ocp]: http://andydote.co.uk/solid-principles-ocp
[blog-solid-lsp]: http://andydote.co.uk/solid-principles-lsp
[blog-solid-isp]: http://andydote.co.uk/solid-principles-isp
[blog-solid-dip]: http://andydote.co.uk/solid-principles-dip
[solid-demo-repo]: https://github.com/Pondidum/Solid.Demo
