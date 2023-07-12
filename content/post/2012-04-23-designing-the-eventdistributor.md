+++
date = '2012-04-23T00:00:00Z'
tags = ['design', 'c#']
title = 'Designing the EventDistributor'

+++

When it comes to developing a new class, I don't tend to use TDD (Test Driven Development), I favour something I have named TAD - Test Aided Development.  In other words, while I am for Unit Testing in general, designing something via writing tests sometimes feels too clunky and slow.  I always write classes and methods with testing very much in mind, but I do not generally write the tests until later on in the process.  This post covers roughly how I wrote the EventDistributor, and what points of note there are along the way.

The first phase in designing it, was the use case:

```csharp
events.RegisterFor<PersonSavedEvent>(OnPersonSaved);
events.Publish(new PersonSavedEvent());
events.UnRegisterFor<PersonSavedEvent>(OnPersonSaved);

private void OnPersonSaved(PersonSavedEvent e)
{
	/* ... */
}
```

From this use case, we are able to tell that we will have 0 -> n events, and each event will have 0 -> n subscribers.  This points to some kind of `Dictionary` based backing field:

```csharp
public class EventDistributor
{
	private readonly Dictionary<Type, List<Action<Object>>> _events;

	public EventDistributor()
	{
		_events = new Dictionary<Type, List<Action<Object>>>();
	}

	public void RegisterFor<TEvent>(Action<TEvent> handler)
	{
	}

	public void UnRegisterFor<TEvent>(Action<TEvent> handler)
	{
	}

	public void Publish<TEvent>(TEvent @event)
	{
	}
}
```

For populating the dictionary, we need to add an entry for a `TEvent` if there is not already one (and create a blank list of handlers), and append our new handler:

```csharp
public void RegisterFor<TEvent>(Action<TEvent> handler)
{
	var type = typeof(TEvent);
	List<Action<Object>> handlers;

	if (_events.TryGetValue(type, out handlers) == false)
	{
		handlers = new List<Action<Object>>();
		_events[type] = handlers;
	}

	handlers.Add(handler);
}
```

This gives rise to the first problem: the line `handlers.Add(handler);` gives us a nice error of: `Error Argument '1': cannot convert from 'System.Action<TEvent>' to 'System.Action<Object>'`.  To fix this, we need to create a new `Action<Object>` and inside that, cast the parameter to `TEvent`.

```csharp
handlers.Add(o => handler((TEvent) o));
```

This does however make the UnRegisterFor method a little more tricky, as doing `handlers.Remove(o => handler((TEvent)o));` doesn't work because they refer to different objects.  Thankfully, as the Action's `GetHashCode()` gives the same result for each instance, providing the content is the same.  We can use this to check for equality:

```csharp
public void UnRegisterFor<TEvent>(Action<TEvent> handler)
{
	var type = typeof(TEvent);
	List<Action<Object>> handlers;

	if (_events.TryGetValue(type, out handlers) == false)
	{
		return;
	}

	var hash = new Action<object>(o => handler((TEvent) o)).GetHashCode();
	handlers.RemoveAll(h => h.GetHashCode() == hash);
}
```

The `Publish` method is nice and straight forward; if the event isn't registered, throw an exception, and raise each subscriber's handler.

```csharp
public void Publish<TEvent>(TEvent @event)
{
	var type = typeof(TEvent);
	List<Action<Object>> handlers;

	if (_events.TryGetValue(type, out handlers) == false)
	{
		throw new EventNotRegisteredException(type);
	}

	handlers.ForEach(h => h.Invoke(@event));
}
```

Now that we have a class roughly implemented, we create the first set of tests for it:

```csharp
[Test]
public void When_publishing_an_event_without_a_handler()
{
	var distributor = new Distributor();
	Assert.DoesNotThrow(() => distributor.Publish(new PersonSavedEvent()));
}

[Test]
public void When_publishing_an_event_with_a_handler()
{
	var wasCalled = false;
	var distributor = new Distributor();

	distributor.RegisterFor<TestEvent>(e => wasCalled = true);
	distributor.Publish(new TestEvent());

	Assert.IsTrue(wasCalled, "The target was not invoked.");
}

[Test]
public void When_publishing_an_event_and_un_registering()
{
	var callCount = 0;
	var increment = new Action<TestEvent>(e => callCount++);
	var distributor = new Distributor();

	distributor.RegisterFor<TestEvent>(increment);
	distributor.Publish(new TestEvent());

	distributor.UnRegisterFor<TestEvent>(increment);
	distributor.Publish(new TestEvent());

	Assert.AreEqual(1, callCount);
}
```

Other than the publish method is currently a blocking operation, there is one major floor to this class: it contains a possible memory leak.  If a class forgets to UnRegisterFor a handler, the EventDistributor will still have a reference stored, preventing the calling class from being garbage collected.  We can demonstrate this with a simple unit test:

```csharp
[Test]
public void When_the_handling_class_does_not_call_unregister()
{
	var count = 0;
	var increment = new Action(() => count++);
	var distributor = new Distributor();

	using(var l = new Listener(distributor, increment))
	{
		distributor.Publish(new TestEvent());
	}

	GC.Collect();
	GC.WaitForPendingFinalizers();
	GC.Collect();

	distributor.Publish(new TestEvent());

	Assert.AreEqual(1, count, "OnPersonSaved should have only been called 1 time, was actually {0}", count);
}

public class Listener : IDisposable
{
	private readonly Action _action;

	public Listener(Distributor events, Action action)
	{
		_action = action;
		events.RegisterFor<TestEvent>(OnTestEvent);
	}

	private void OnTestEvent(TestEvent e)
	{
		_action.Invoke();
	}

	public void Dispose()
	{
	}
}
```

While it would be simple to just say that it's the responsibility of the calling code to call `UnRegisterFor`, it would be better to handle that (likely) case ourselves.  Good news is that .net has just the class needed for this built in: [WeakReference][1].  This class allows the target class to become disposed even while we still hold a reference to it.  We can then act on the disposal, and remove our event registration.

Changing the Dispatcher to use this in its dictionary is fairly straight forward, and we even loose some of the casting needed to add items to the list:

```csharp
public class Distributor
{
	private readonly Dictionary<Type, List<WeakReference>> _events;

	public Distributor()
	{
		_events = new Dictionary<Type, List<WeakReference>>();
	}

	public void RegisterFor<TEvent>(Action<TEvent> handler)
	{
		var type = typeof(TEvent);
		List<WeakReference> recipients;

		if (!_events.TryGetValue(type, out recipients))
		{
			recipients = new List<WeakReference>();
			_events[type] = recipients;
		}

		recipients.Add(new WeakReference(handler));
	}

	public void UnRegisterFor<TEvent>(Action<TEvent> handler)
	{
		var type = typeof(TEvent);
		List<WeakReference> recipients;

		if (_events.TryGetValue(type, out recipients))
		{
			recipients.RemoveAll(o => o.Target.GetHashCode() == handler.GetHashCode());
		}
	}

	public void Publish<TEvent>(TEvent @event)
	{
		var type = typeof(TEvent);
		List<WeakReference> recipients;

		if (!_events.TryGetValue(type, out recipients))
		{
			return;
		}

		recipients.RemoveAll(wr => wr.IsAlive == false);
		recipients.ForEach(wr => ((Action<TEvent>)wr.Target).Invoke(@event));
	}
}
```

The main points to note with this change is:

 * We no longer need to create a new `Action<Object>` just to cast the handler in `RegisterFor`.
 * `UnRegisterFor` no longer needs to create a new `Action<Object>` to get the hash code.
 * `Publish` has an extra line to remove all handlers where the target has become disposed.

The next item to work on in this class is making the `Publish` method non-blocking, which can be done in a variety of ways.

The first option is to create a thread that will invoke all the handlers one after the other.  This has the advantage of only one extra thread to deal with, but has the drawback of a single unresponsive handler will block all other handlers.  Ignoring locking and cross-threading issues for the time being, it could be implemented like this:

```csharp
public void PublishAsyncV1<TEvent>(TEvent @event)
{
	var type = typeof(TEvent);
	List<WeakReference> recipients;

	if (!_events.TryGetValue(type, out recipients))
	{
		return;
	}

	var task = new Task(() =>
	{
		recipients.RemoveAll(wr => wr.IsAlive == false);
		recipients.ForEach(wr => ((Action<TEvent>) wr.Target).Invoke(@event));
	});

	task.Start();
}
```

The second option is to have a separate thread/invocation for each handler.  This has the advantage that each of the handlers can take as much time as needed, and will not block any other handlers from being raised, however if you have many handlers to be invoked, it could be slower to return than the first option.  Again, ignoring locking and cross-threading issues, it could be implemented like so:

```csharp
public void PublishAsyncV2<TEvent>(TEvent @event)
{
	var type = typeof(TEvent);
	List<WeakReference> recipients;

	if (!_events.TryGetValue(type, out recipients))
	{
		return;
	}

	recipients.RemoveAll(wr => wr.IsAlive == false);
	recipients.ForEach(wr =>
	{
		var handler = (Action<TEvent>)wr.Target;
		handler.BeginInvoke(@event, handler.EndInvoke, null);
	});
}
```

Personally, I go for the second method, as the number of handlers to be invoked is usually fairly small.

The next part to consider is what we conveniently ignored earlier - the cross-threading issues.  The main issue we have is handlers being added or removed from the list while we are iterating over it.

Now I cannot remember where I read it, it was either from Jon Skeet, or from the [Visual Basic .Net Threading Handbook][2], but the rough idea was "You should lock as smaller area of code as possible".  This is to minimise the chance of a deadlock.  Starting with the Publish methods, we only need to lock the parts that iterate over the list:

```csharp
lock (Padlock)
{
	recipients.RemoveAll(wr => wr.IsAlive == false);
	recipients.ForEach(wr =>
	{
		var handler = (Action<TEvent>)wr.Target;
		handler.BeginInvoke(@event, handler.EndInvoke, null);
	});
}
```

The UnRegisterFor method is also very straight forward, as we again only need to worry about the iteration:

```csharp
if (_events.TryGetValue(type, out recipients))
{
	lock (Padlock)
	{
		recipients.RemoveAll(o => o.Target.GetHashCode() == handler.GetHashCode());
	}
}
```

The RegisterFor method takes a little more locking than the other two, as this will handle the creation of the lists, as well as the addition to the list:

```csharp
lock (Padlock)
{
	if (!_events.TryGetValue(type, out recipients))
	{
		recipients = new List<WeakReference>();
		_events[type] = recipients;
	}

	recipients.Add(new WeakReference(handler));
}
```

The full code listing and unit tests for this can be found here: [EventDistributor Gist][3].

[1]: http://msdn.microsoft.com/en-us/library/system.weakreference.aspx
[2]: http://www.amazon.co.uk/Visual-Basic-NET-Threading-Handbook-Programmer/dp/1861007132
[3]: https://gist.github.com/2467463
