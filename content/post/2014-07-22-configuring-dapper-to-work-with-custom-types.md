---
date: "2014-07-22T00:00:00Z"
tags: ["design", "c#", "typing", "sql", "database", "orm"]
title: Configuring Dapper to work with custom types
---

In the [last post][blog-type-ids] we looked at using custom ID types to help abstract the column type from the domain.

This works well until you start trying to load and save entities using an ORM, as the ORM has not way to know how to map a column to a custom type.  ORMs provide extension points to allow you to create these mappings.  As I tend to favour using [Dapper][orm-dapper], we will go through setting it up to work with our custom ID types.

We need to be able to get the raw value out of the id type, but without exposing this to the outside world.  To do this we internal interface:

```csharp
internal interface IValueID
{
	object Value();
}
```

Then update our id struct with a private implementation of the interface, and also mark the only constructor as internal:

```csharp
public struct PersonID : IValueID
{
	private readonly Guid _id;

	internal PersonID(Guid id)
	{
		_id = id;
	}

	object IValueID.Value()
	{
		return _id;
	}
}
```

We now can define a class which Dapper can use to do the mapping from uuid to id:

```csharp
public class PersonIDHandler : SqlMapper.TypeHandler<PersonID>
{
	public override void SetValue(IDbDataParameter parameter, PersonID value)
	{
		parameter.Value = ((IValueID)value).Value();
	}

	public override PersonID Parse(object value)
	{
		return new PersonID((Guid)value);
	}
}
```

We then need to regiter the command with Dapper once on start up of our application:

```csharp
SqlMapper.AddTypeHandler(new PersonIDHandler());
```

Now when Dapper loads an object with a property type of `PersonID` it will invoke the `Parse` method on `PersonIDHandler`, and populate the resulting object correctly.  It will also work when getting a value from the `PersonID` property, invoking the `SetValue` method on `PersonIDHandler`.

## Extension

While the `PersonIDHandler` works, I really don't want to be creating essentially the same class over and over again for each ID type.  We can fix this by using a generic id handler class, and some reflection magic.

We start off by creating a generic class for id handling:

```csharp
public class CustomHandler<T> : SqlMapper.TypeHandler<T>
{
	private readonly Func<Object, T> _createInstance;

	public CustomHandler()
	{
		var ctor = typeof(T)
			.GetConstructors()
			.Single(c => c.GetParameters().Count() == 1);

		var paramType = ctor
			.GetParameters()
			.First()
			.ParameterType;

		_createInstance = (value) => (T)ctor.Invoke(new[] { Convert.ChangeType(value, paramType) });
	}

	public override void SetValue(IDbDataParameter parameter, T value)
	{
		parameter.Value = ((IValueID)value).Value();
	}

	public override T Parse(object value)
	{
		return _createInstance(value);
	}
}
```

The constructor of this class just finds a single constructor on our ID type with one argument, and creates a Func which will create an instance of the id passing in the value.   We put all this constructor discovery logic into the `CustomHandler`'s constructor as this information only needs to be calculated once, and can then be used for every `Parse` call.

We then need to write something to build an instance of this for each ID type in our system.  As all of our IDs need to implement `IValueID` to work, we can scan for all types in the assembly implementing this interface, and then operate on those.

```csharp
public class InitialiseDapper : IApplicationStart
{
	public void Initialise()
	{
		var interfaceType = typeof(IValueID);

		var idTypes = interfaceType
			.Assembly
			.GetTypes()
			.Where(t => t.IsInterface == false)
			.Where(t => t.IsAbstract == false)
			.Where(t => t.GetInterfaces().Contains(interfaceType));

		var handler = typeof(CustomHandler<>);

		foreach (var idType in idTypes)
		{
			var ctor = handler
				.MakeGenericType(new[] { idType })
				.GetConstructor(Type.EmptyTypes);

			var instance = (SqlMapper.ITypeHandler)ctor.Invoke(new object[] { });

			SqlMapper.AddTypeHandler(idType, instance);
		}
	}
}
```

This class first scans the assembly containing `IValueID` for all types implementing `IValueID` which are not abstract, and not interfaces themselves.  It then goes through each of these types, and builds a new instance of `CustomHandler` for each type, and registers it with Dapper.

You might notice this is in a class which implements `IApplicationStart` - In most of my larger projects, I tend to have an interface like this, which defines a single `void Initialise();` method.  Implementations of the interface get looked for on startup of the application, and their `Initialise` method called once each.

[blog-type-ids]: http://andydote.co.uk/strong-type-your-entity-ids
[orm-dapper]: https://github.com/StackExchange/dapper-dot-net
