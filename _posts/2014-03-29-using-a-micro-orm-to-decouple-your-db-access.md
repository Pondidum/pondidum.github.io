---
layout: post
title: Using a Micro ORM to decouple your DB Access
tags: design code net automapper sql memento

---

One of the databases I use on a regular bases has a rather interesting column naming scheme;  all columns have a prefix, based on the table name.  For example, the table containing people would have the prefix `PEO_`, so you would have this:

```sql
Select * from People

PEO_PersonID, PEO_FirstName, PEO_LastName, PEO_DoB
-----------------------------------------------------
1             John           Jones         1984-07-15
```

I believe the idea was so that when querying, you would not have any column name clashes.  This of course breaks down if you have to join on the same table twice.

This structure presents a problem when it comes to reading the tables into objects in code, as it removes the ability to use an orm - I have yet to see one which allows you to specify a prefix to be used on all columns in a table.

The existing entities are all manually read, and follow the same pattern:

```csharp

public abstract class Entity
{
	public void Load()
	{
		using (var reader = SqlHelper.ExecuteReader("connectionstring", ReadProcedureName))
		{
			if (reader.Read())
			{
				Read(reader);
			}
		}
	}
}

public class Person : Entity
{
	public int ID { get; set; }
	public string FirstName { get; set; }
	public string LastName { get; set; }
	public DateTime DoB { get; set; }

	protected override String ReadProcedureName { get { return "p_getPerson"; } }

	protected override void Read(IDataReader reader)
	{
		ID = reader.GetInt32(0);
		FirstName = reader.GetString(1);
		LastName = reader.GetString(2);
		DoB = reader.GetDateTime(3);
	}
}
```

Note how columns are read in order, which means two things: you cannot use `select *` as your query, and you cannot change column order etc.

To help split this so we can start using an ORM to do the mapping for us, we can utilise the [Memento Pattern][memento-pattern].  First we create a new object, which will be used to read and write from the database:

```csharp
public class PersonDto
{
	public int PEO_ID { get; set; }
	public string PEO_FirstName { get; set; }
	public string PEO_LastName { get; set; }
	public DateTime PEO_DoB { get; set; }
}
```

Note the property names match the column names of the table in the db, our read method could then get changed to this:

```csharp
public abstract class Entity<TDto>
{
	protected virtual string ReadProcedureName { get { return ""; } }

	public void Load()
	{
		var results = _connection.Query<TDto>(ReadProcedureName).ToList();

		if (results.Any())
		{
			Read(results.First());
		}
	}

	protected virtual void Read(TDto dto)
	{
	}
}

public class Person : Entity<PersonDto>
{
	public int ID { get; set; }
	public string FirstName { get; set; }
	public string LastName { get; set; }
	public DateTime DoB { get; set; }

	protected override void Read(PersonDto dto)
	{
		ID = dto.PEO_ID;
		FirstName = dto.PEO_FirstName;
		LastName = dto.PEO_LastName;
		DoB = dto.PEO_DoB;
	}
}
```

This gives us several benefits, in that we can change column naming and ordering freely without effecting the actual `Person` object, and we have made the class slightly more testable - we can pass it a faked `PersonDto` if we needed to load it with some data for a test.

We can however make another improvement to this - namely in the `Read` method, as this is a prime candidate for [AutoMapper][package-automapper].  To get this to work though, have two choices: the first is to manually specify the mappings of one object to the other, and the second is to write a profile which will do the work for us.  Unsurprisingly, I went with the second option:

```csharp
public class PrefixProfile : Profile
{
	private readonly IDictionary<Type, Type> _typeMap;

	public PrefixProfile(IDictionary<Type, Type> typeMap )
	{
		_typeMap = typeMap;
	}

	public override string ProfileName
	{
		get { return "PrefixProfile"; }
	}

	protected override void Configure()
	{
		foreach (var pair in _typeMap)
		{
			var prefix = GetPrefix(pair.Value.GetProperties());

			RecognizeDestinationPrefixes(prefix);
			RecognizePrefixes(prefix);

			CreateMap(pair.Key, pair.Value);
			CreateMap(pair.Value, pair.Key);
		}
	}

	private string GetPrefix(IEnumerable<PropertyInfo> properties)
	{
		return properties
			.Select(GetPrefixFromProperty)
			.FirstOrDefault(p => String.IsNullOrWhiteSpace(p) == false);
	}

	protected virtual string GetPrefixFromProperty(PropertyInfo property)
	{
		var name = property.Name;

		return name.IndexOf("_", StringComparison.OrdinalIgnoreCase) >= 0
			? name.Substring(0, name.IndexOf("_", StringComparison.OrdinalIgnoreCase) + 1)
			: String.Empty;
	}
}
```

This class takes in a dictionary of types (in this case will be things like `Person` => `PersonDto`).  It goes through each pair in the list and determines the prefix for the destination class (the dto).  The `GetPrefixFromProperty` is virtual so that I can customise it for other uses later.

To use this we just need to initialise AutoMapper with the class once on start up:

```csharp
var map = new Dictionary<Type, Type>();
map.Add(typeof (Person), typeof (PersonDto));

Mapper.Initialize(config => config.AddProfile(new PrefixProfile(map)));
```

This means our `Person` class becomes very small:

```csharp
public class Person : Entity<PersonDto>
{
	public int ID { get; set; }
	public string FirstName { get; set; }
	public string LastName { get; set; }
	public DateTime DoB { get; set; }
}
```

And the `Entity` class can take care of the mapping for us, but utilising AutoMapper's Type based Map method:

```csharp
public abstract class Entity<TDto>
{
	protected virtual string ReadProcedureName { get { return ""; } }

	public void Load()
	{
		var _connection = new SqlConnection("");
		var results = _connection.Query<TDto>(ReadProcedureName).ToList();

		if (results.Any())
		{
			Read(results.First());
		}
	}

	protected void Read(TDto dto)
	{
		Mapper.Map(dto, this, typeof(TDto), GetType());
	}
}
```

While the design of having each entity responsible for saving and loading of itself is not the best design, it is what the existing system has in place (around 400 entities exist at last count).  By taking these steps we can remove a lot of boilerplate code from our codebase, which means when we wish to change to a different architecture (such as session or transaction objects in a similar style to RavenDB's ISession), it will be an easier transition.


[memento-pattern]: http://www.dofactory.com/Patterns/PatternMemento.aspx
[package-automapper]: http://automapper.org/
