---
layout: post
title: SOLID Principles - LSP
tags: design c# solid

---

## Liskov Substitution Principle

[Single Responsibility][blog-solid-srp] | [Open Closed][blog-solid-ocp] | [Liskov Substitution][blog-solid-lsp] | [Interface Segregation][blog-solid-isp] | [Dependency Inversion][blog-solid-dip]

The Liskov Substitution Principle is states:

> If **S** is a sub-type of **T**, then objects of type **T** maybe replaced with objects of type **S**

At face value, it means that a small class hierarchy like this:

```csharp
public class FileEntry
{

}

public class DbFileEntry : FileEntry
{

}
```

And a method which takes in a `FileEntry`, can be called like this:

```csharp
ProcessFile(new FileEntry());
```

Or like this:

```csharp
ProcessFile(new DbFileEntry());
```

This however only takes the principle at face value, and would not provide much value.  However, just because a class implements the expected interface does not necessarily mean that it can be a drop in replacement for another implementation.  This can be down to a number of factors, such as side effects of methods (like different kinds of exception being thrown), and external modification of state.

### Side Effects

In this example, you can see that the methods both have a pre-condition on some internal data, but as they throw different kinds of exceptions, they violate the principle:

```csharp
public class FileEntry
{
	public virtual void Process()
	{
		if (File.Exists(Path) == false)
			throw new FileNotFoundException(Path);

		//do work
	}
}

public class DbFileEntry : FileEntry
{
	public override void Process()
	{
		if (Database.Contains(_id) == false)
			throw new KeyNotFoundException(_id.ToString());

		//do work
	}
}
```

The reason for this being a violation is due to what the calling code is expecting to handle:

```csharp
public void RunFiles(IEnumerable<FileEntry> files)
{
	foreach (var file in files)
	{
		try
		{
			file.Process();
		}
		catch (FileNotFoundException ex)
		{
			_fails.Add(file.Name);
		}
	}
}
```

This method when called with a list of `FileEntry` will run every entry, and add the names of any which failed to a collection for later use.  However if it were called with a list of `DbFileEntry`, the first file to fail would cause then entire method to fail, and no more files would be processed.

Fixing the classes so they obey the LSP could be done by changing the `DbFileEntry` to throw the same kind of exception as the `FileEntry`, but the exception type `FileNotFoundException` wouldn't make sense in the context of a database.

The solution is to create a new exception type which the `Process` methods with throw, and that the `RunFiles` method will catch:

```csharp
public class FileEntry
{
	public virtual void Process()
	{
		if (File.Exists(Path) == false)
			throw new FileEntryProcessException(FileNotFoundException(Path));

		//do work
	}
}

public class DbFileEntry : FileEntry
{
	public override void Process()
	{
		if (_database.Contains(_id) == false)
			throw new FileEntryProcessException(KeyNotFoundException(_id));

		//do work
	}
}

public void RunFiles(IEnumerable<FileEntry> files)
{
	foreach ( var file in files)
	{
		try
		{
			file.Process();
		}
		catch (FileEntryProcessException ex)
		{
			_fails.Add(file.Name);
		}
	}
}
```

By keeping the original exceptions we were going to throw as the `.InnerException` property of our new `FileEntryProcessException` we can still preserve the more specific exceptions, while allowing the `RunFiles` method to catch it.

An alternate solution to this would be to have two new specific exception types, which both inherit a single type:

```csharp
public abstract class ProcessException : Exception()
{
}

public class FileNotFoundProcessException : ProcessException
{
	public FileNotFoundProcessException(String path)
	{}
}

public class KeyNotFoundProcessException : ProcessException
{
	public KeyNotFoundProcessException(Guid id)
	{}
}
```

The problem with this approach is that you are hoping that all consumers of `FileEntry` are catching `ProcessException`, rather than one of it's sub-classes.  By using the first solution, you are forcing the consumer to catch your one exception type.

### State Mutation

Extra methods on a sub class can cause a violation of the Liskov Substitution Principle too; by mutating state, and causing calling code to make un-expected transitions.  Take this for example:

```csharp
public class DefaultStateGenerator
{
	private int _state;

	public int GetNextStateID(int currentState)
	{
		return Math.Min(++currentState, 3);
	}
}

public class StateMachine
{
	public StateMachine(IStateGenerator generator)
	{
		_generator = generator;
	}

	public void Transition()
	{
		var newState = _generator.GetNextStateID(_currentState);

		switch (newState)
		{
			case 0:
				break; //do nothing

			case 1:
				break; //do nothing

			case 2:
				PayTheMan();
				break;
		}

		_currentState = newState;
	}
}
```

Using the `DefaultStateGenerator` will cause the state machine to work as expected - it will transition through the states, calling `PayTheMan` one on state 2, and then just sticking at state 3 for subsequent calls.  However, if you were to use the `EvilStateGenerator` things might be a bit different:

```csharp
public class EvilStateGenerator : IStateGenerator
{
	private bool _evil;

	public int GetNextStateID(int currentState)
	{
		return _evil ? 2 : Math.Min(++currentState, 3);
	}

	public void BeEvil()
	{
		_evil = true;
	}
}
```

This `EvilStateGenerator` works as usual, until a call to its `BeEvil` method gets called, at which point it will return state 2 every time, causing the `PayTheMan` method to be called on every `Transition`.

Hopefully these two examples provide sufficient reason for paying attention to the Liskov Substitution Principle.

All source code is available on my Github: [Solid.Demo Source Code][solid-demo-repo]

[blog-solid-srp]: http://andydote.co.uk/solid-principles-srp
[blog-solid-ocp]: http://andydote.co.uk/solid-principles-ocp
[blog-solid-lsp]: http://andydote.co.uk/solid-principles-lsp
[blog-solid-isp]: http://andydote.co.uk/solid-principles-isp
[blog-solid-dip]: http://andydote.co.uk/solid-principles-dip
[solid-demo-repo]: https://github.com/Pondidum/Solid.Demo
