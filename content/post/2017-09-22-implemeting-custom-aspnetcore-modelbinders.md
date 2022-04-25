---
date: "2017-09-22T00:00:00Z"
tags: design aspnetcore dotnetcore
title: Implementing Custom Aspnet Core ModelBinders
---

This post is a summary of a [stream](https://twitch.tv/pondidum) I did last night where I implemented all of this.  If you want to watch me grumble my way through it, it's [available on YouTube here](https://www.youtube.com/watch?v=hR213Oxj_xI).

In my [Crispin](https://github.com/pondidum/crispin) project, I wanted the ability to support loading Toggles by both name and ID, for all operations.  As I use mediator to send messages from my controllers to the handlers in the domain, this means that I had to either:

* create separate request types for loading by name and loading by id
* have both an `ID` and `Name` property on each method

I didn't like the sound of either of these as both involve more typing than I want to do, and the second variant has the added downside of causing a lot of `if` statements in the handlers, as you have to work out which is set before loading.  Not to mention the duplication of the load toggle logic in every handler.

The solution I came up with was to use some inheritance, a static factory, some method hiding, and a custom `IModelBinder`.

## ToggleLocator

I started off by having an `abstract` base class called `ToggleLocator`.  To start with, it just has two static methods for creating an instance of `ToggleLocator`:

```csharp
public abstract class ToggleLocator
{
	public static ToggleLocator Create(Guid toggleID) => new ToggleLocatorByID(toggleID);
	public static ToggleLocator Create(string toggleName) => new ToggleLocatorByName(toggleName);
}
```

As this is going to be used in both Query handlers and Command handlers, I need to be able to load the Toggle (the EventSourced AggregateRoot), and the ToggleView (the projected current state of the AggregateRoot).  So we add two `abstract` methods to the `ToggleLocator`

```csharp
internal abstract ToggleView LocateView(IStorageSession session);
internal abstract Toggle LocateAggregate(IStorageSession session);
```

Note that not only are these two methods `abstract`, they are also `internal` - we don't want anything outside the domain to know about how a toggle is loaded.  I was considering using an privately implemented interface to do this method hiding, but didn't see the point as I can acomplish the same using the internal methods.

We can now write two implementations of the `ToggleLocator`.  First up is the `ToggleLocatorByID`, which is very straight forward to implement; we use the ID to load the AggregateRoot directly, and the `AllToggles` view can be queried by ID to fetch the view version also.

```csharp
public class ToggleLocatorByID : ToggleLocator
{
	private readonly ToggleID _toggleID;

	public ToggleLocatorByID(ToggleID toggleID)
	{
		_toggleID = toggleID;
	}

	internal override ToggleView LocateView(IStorageSession session) => session
		.LoadProjection<AllToggles>()
		.Toggles
		.SingleOrDefault(view => view.ID == _toggleID);

	internal override Toggle LocateAggregate(IStorageSession session) => session
		.LoadAggregate<Toggle>(_toggleID);
}
```

The more interesting class to implement is `ToggleLocatorByName`, as this needs to be able to load an AggregateRoot by name; something which is not directly supported.  So to do this we fetch the `ToggleView` first, and then use the `ID` property so we can load the `Toggle`:

```csharp
public class ToggleLocatorByName : ToggleLocator
{
	private readonly string _toggleName;

	public ToggleLocatorByName(string toggleName)
	{
		_toggleName = toggleName;
	}

	internal override ToggleView LocateView(IStorageSession session) => session
		.LoadProjection<AllToggles>()
		.Toggles
		.SingleOrDefault(t => t.Name.Equals(_toggleName, StringComparison.OrdinalIgnoreCase));

	internal override Toggle LocateAggregate(IStorageSession session)
	{
		var view = LocateView(session);

		return view != null
			? session.LoadAggregate<Toggle>(view.ID)
			: null;
	}
}
```

All this means that the handlers have no conditionals for loading, they just call the relevant `.Locate` method:

```csharp
private Task<UpdateToggleTagsResponse> ModifyTags(ToggleLocator locator, Action<Toggle> modify)
{
	using (var session = _storage.BeginSession())
	{
		var toggle = locator.LocateAggregate(session);
		//or
		var view  = locator.LocateView(session);
		//...
	}
}
```

And in the controllers, we have separate action methods for each route:

```csharp
[Route("name/{toggleName}/tags/{tagName}")]
[HttpPut]
public async Task<IActionResult> PutTag(string toggleName, string tagName)
{
	var request = new AddToggleTagRequest(ToggleLocator.Create(toggleName), tagName);
	var response = await _mediator.Send(request);

	return new JsonResult(response.Tags);
}

[Route("id/{toggleID}/tags/{tagName}")]
[HttpPut]
public async Task<IActionResult> PutTag(Guid toggleID, string tagName)
{
	var request = new AddToggleTagRequest(ToggleLocator.Create(ToggleID.Parse(toggleID)), tagName);
	var response = await _mediator.Send(request);

	return new JsonResult(response.Tags);
}
```

But that is still more duplication than I would like, so lets see if we can resolve this with a custom `IModelBinder`.

## Custom IModelBinder for ToggleLocator

To make a custom model binder, we need to implement two interfaces: `IModelBinderProvider` and `IModelBinder`.  I am not sure why `IModelBinderProvider` exists to be perfectly honest, but you need it, and as it is doing nothing particularly interesting, I decided to implement both interfaces in the one class, and just return `this` from `IModelBinderProvider.GetBinder`:

```csharp
public class ToggleLocatorBinder : IModelBinderProvider
{
	public IModelBinder GetBinder(ModelBinderProviderContext context)
	{
		if (context.Metadata.ModelType == typeof(ToggleLocator))
			return this;

		return null;
	}
}
```

We can then implement the second interface, `IModelBinder`.  Here we check (again) that the parameter is a `ToggleLocator`, fetch the value which came from the route (or querystring, thanks to the `.ValueProvider` property).

All I need to do here is try and parse the value as a `Guid`.  If it parses successfully, we create a `ToggleLocatorByID` instance, otherwise create a `ToggleLocatorByName` instance.

```csharp
public class ToggleLocatorBinder : IModelBinderProvider, IModelBinder
{
	public Task BindModelAsync(ModelBindingContext bindingContext)
	{
		if (bindingContext.ModelType != typeof(ToggleLocator))
			return Task.CompletedTask;

		var value = bindingContext.ValueProvider.GetValue(bindingContext.FieldName);
		var guid = Guid.Empty;

		var locator = Guid.TryParse(value.FirstValue, out guid)
			? ToggleLocator.Create(ToggleID.Parse(guid))
			: ToggleLocator.Create(value.FirstValue);

		bindingContext.Result = ModelBindingResult.Success(locator);

		return Task.CompletedTask;
	}
}
```

We add this into our MVC registration code at the beginning of the `ModelBinderProviders` collection, as MVC will use the first binder which can support the target type, and there is a binder in the collection somewhere which will handle anything which inherits object...

```csharp
services.AddMvc(options =>
{
	options.ModelBinderProviders.Insert(0, new ToggleLocatorBinder());
});
```

Now we can reduce our action methods down to one which handles both routes:

```csharp
[Route("id/{id}/tags/{tagName}")]
[Route("name/{id}/tags/{tagName}")]
[HttpPut]
public async Task<IActionResult> PutTag(ToggleLocator id, string tagName)
{
	var request = new AddToggleTagRequest(id, tagName);
	var response = await _mediator.Send(request);

	return new JsonResult(response.Tags);
}
```

Much better, no duplication, and no (obvious) if statements!
