---
date: "2015-03-25T00:00:00Z"
tags: design api
title: Communicating Intent in APIs
---

Recently was trying to work out how to allow custom resources to be specified in [Dashen][github-dashen].  I already know what data is needed/defined for a resource: a name, a MIME type, and a Stream.  We can make this required data known very easily:

```csharp
public class Resource
{
	public string Name { get; private set; }
	public string MimeType { get; private set; }
	public Stream Content { get; private set; }

	public Resource(string name, string mimeType, Stream content)
	{
		Name = name;
		MimeType = mimeType;
		Content = content;
	}
}
```

As all the parameters can only be set through the constructor, you are communicating that they are all required.

However when it comes to adding this `Resource` into our configuration, we are met with 3 possible solutions:

## Resource collection on the config

```csharp

var dashboard = DashboardBuilder.Create(new DashboardConfiguration
{
	ListenOn = new Uri("http://localhost:3030"),
	Resources = new[] { new Resource("test.png", "image/png", new FileStrea(...))}
});
```

As the `DashboardConfiguration` object is only used in this one call, it implies that the contents of it only get read once.
Nothing to stop you holding on to a reference to the `Resources` collection though.

## AddResource method on the config

```csharp

var config = new DashboardConfiguration
config.ListenOn = new Uri("http://localhost:3030");

config.AddResource(new Resource("test.png", "image/png", new FileStrea(...)));
//or
config.AddResource("test.png", "image/png", new FileStrea(...));

var dashboard = DashboardBuilder.Create(config);
```

`Resources` are still added to the `DashboardConfiguration`, but this time via a method.  This hides the internal storage of resources.  Second version also means we can hide the `Resource` class from the public too if we want.
Also implies a level of uniqueness - could throw an exception on duplicate name being added, or rename the method to `AddUniqueResource` or similar.

## AddResource method on the Dashboard

```csharp

var dashboard = DashboardBuilder.Create(new DashboardConfiguration
{
	ListenOn = new Uri("http://localhost:3030"),
});

dashboard.AddResource(new Resource("test.png", "image/png", new FileStrea(...)));
//or
dashboard.AddResource("test.png", "image/png", new FileStrea(...));
```

`Resource` class is still hideable. Being able to add to the dashboard rather than the config implies that resources could be added at anytime, rather than just startup/config time.

# Selected Solution

In the end I decided to expose the `Resources` as an `IEnumerable<Resource>` on the `DashboardConfiguration` object.  I did this as I don't actually mind if the collection gets modified once the dashboard is started, and I can see some use-cases for dynamic resource resolution.


[github-dashen]: https://github.com/pondidum/Dashen
