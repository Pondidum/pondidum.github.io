---
date: "2014-07-17T00:00:00Z"
tags: design c# typing sql database orm
title: Strong Type your entity IDs.
---

## The Database is just an Implementation Detail

A quote from Martin Fowler given during his Architecture talk stated that the Database in your application should just be an implementation detail.  I agree on this wholeheartedly and find that its really not that difficult to achieve if you think about your architecture carefully.

Having said that, I still see parts of the database implementation leaking out into the domain, mainly in the form of IDs.  This might not seem like much of a leak, but it does cause a few problems, especially on larger systems.

The first problem ocours when you have a function taking in an ID of some form, and the parameter name is not really forthcoming on what object's ID it's expecting.  This is especially problematic if your ID columns are int based, rather than uuids, as passing any int to the function will return data - just not necessarily the data you were expecting.

The second problem is that it ties you to using the same ID type as the database is using.  If the database is just an implementation detail, then it definitely should not be dictating what types your domain should be using.

For example, take the following two classes:

```csharp
public class Account
{
	public int ID { get; }
	//...
}

public class User
{
	public int ID { get; }
	public IEnumerable<Account> Accounts { get; }
}
```

The two classes on their own are not unreasonable, but the use of an `int` for the ID is problematic.  Given the following method:

```csharp
public DateTime GetLastActiveDate(int userID)
{
	return ...
}
```

Both of the following calls are valid, and neither the code nor the compiler will tell you which one is correct (if any!):

```csharp
var date1 = GetLastActiveDate(_user.ID);
var date2 = GetLastActiveDate(_user.Accounts.First().ID);
```


## Using the Type System to prevent bad arguments

We can fix this problem by using the Type System to force the correct ID type to be passed in.

First we need to abstract the notion of an ID to be separate from what type its value is.  To do this we create some structs, one for each ID in our system:

```csharp
public struct UserID
{
	private readonly int _value;

	public UserID(int value)
	{
		_value = value;
	}

	public override int GetHashCode()
	{
		return _value;
	}

	public override bool Equals(object obj)
	{
		return (obj is UserID) && (((UserID)obj)._value == _value);
	}
}

public struct AccountID
{
	private readonly int _value;

	public AccountID(int value)
	{
		_value = value;
	}

	public override int GetHashCode()
	{
		return _value;
	}

	public override bool Equals(object obj)
	{
		return obj is AccountID && GetHashCode() == obj.GetHashCode();
	}
}
```

Both of our structs store their values immutably so that they cannot be changed after creation, and we override `GetHashCode` and `Equals` so that separate instances can be compared for equality properly.  Note also that there is no inheritance between the two structs - we do not want the ability for a method to expect a `UserID` and find someone passing in an `AccountID` because it inherits.

We can now update our objects to use these IDs:

```csharp
public class Account
{
	public AccountID ID { get; }
	//...
}

public class User
{
	public UserID ID { get; }
	public IEnumerable<Account> Accounts { get; }
}
```

And update any method which expects an ID now gets the specific type:

```csharp
public DateTime GetLastActiveDate(UserID userID)
{
	return ...
}
```

This means that when someone writes this:

```csharp
var date = GetLastActiveDate(_user.Accounts.First().ID);
```

The compiler will complain with an error: `Unable to cast type 'AccountID` to type 'UserID``.

## Abstracting column type

By doing this work to use custom types instead of native types for our IDs gives us another benefit:  we can hide what type the database is using from the domain, meaning we could change our table's key to be a uuid, and the only place we would need to change in code would be the relevant ID class.

## Extra functionality

One more benefit that comes from this approach is that our IDs are now first class citizens in the type world, and we can imbue them with extra functionality.

A system I use has a table with both a uuid column for the primary key, and an int based refnum column for displaying to users, something like this:

	person:
	id : uuid, forename : varchar(50), surname : varchar(50), dateofbirth : date, refnum : int

As we have a `PersonID` type, we can make that hold both values, and override the `ToString` method so that when called it displays the user friendly ID:

```csharp
public struct PersonID
{
	private readonly Guid _id;
	private readonly int _refnum;

	public PersonID(Guid id, int refnum)
	{
		_id = id;
		_refnum = refnum;
	}

	public override int GetHashCode()
	{
		//http://stackoverflow.com/questions/263400/what-is-the-best-algorithm-for-an-overridden-system-object-gethashcode/263416#263416
		unchecked
		{
			int hash = 17;
			hash = hash * 23 + _id.GetHashCode();
			hash = hash * 23 + _refnum.GetHashCode();
			return hash;
		}
	}

	public override bool Equals(object obj)
	{
		return (obj is PersonID) && (((PersonID)obj)._id == _id) && (((PersonID)obj)._refnum == _refnum);
	}

	public override string ToString()
	{
		return _refnum.ToString()
	}
}
```

This means that if in the future we decided to convert to using the refnum as the primary key, and drop the uuid column, again all we would need to do would be to update the `PersonID` type, and the rest of our code base would be unaffected.
