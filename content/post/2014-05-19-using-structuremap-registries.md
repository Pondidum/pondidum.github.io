+++
date = '2014-05-19T00:00:00Z'
tags = ['design', 'c#', 'structuremap', 'separation', 'testing']
title = 'Using StructureMap Registries for better separation'

+++

When it comes to configuring StructureMap, it supports the use of [Registries][structuremap-registries].  Registries support everything that the standard configure method does(`new Container(c => { /* */});`).

There are two main reasons that I use the registries rather then doing all my configuration in the Container's lambda:  separation of concerns (one registry per area of code) and easier testing (which we will go into shortly).

The only down side I can see to using registries is that it can scatter your configuration across your codebase - but if you have ReSharper, doing a 'Find Implementations' on `Registry` will find them all for you, so it really isn't much of a down side.

## Separation of Concerns

Taking [NuCache][github-nucache] as an example, in our app start we have [ConfigureContainer.cs][github-nucache-configurecontainers]:

```csharp
public static void Register(HttpConfiguration config)
{
	var container = new Container(c => c.Scan(a =>
	{
		a.TheCallingAssembly();
		a.WithDefaultConventions();
		a.LookForRegistries();
	}));

	config.DependencyResolver = new StructureMapDependencyResolver(container);
}
```

This snippet of code gets called as part of the AppStart, and tells StructureMap to use the default conventions (eg: `IFileSystem => FileSystem`), and to process any registries it finds.  The app then has multiple Registries with the actual configuration in (usually one per namespace, although not all namespaces have a registry).

For example, we have these two registries:

```csharp
public class InfrastructureRegistry : Registry
{
	public InfrastructureRegistry()
	{
		For<IPackageCache>()
			.Use<FileSystemPackageCache>()
			.OnCreation(c => c.Initialise())
			.Singleton();
	}
}

public class ProxyBehaviourRegistry : Registry
{
	public ProxyBehaviourRegistry ()
	{
		Scan(a =>
		{
			a.TheCallingAssembly();
			a.AddAllTypesOf<IProxyBehaviour>();
		});
	}
}
```

The [InfrastructureRegistry][github-nucache-infrastructureregistry] just specifies how to resolve an `IPackageCache`, as it has requires some extra initialisation and to be treated as a singleton.

The [ProxyBehaviourRegistry][github-nucache-proxyregistry] tells StructureMap to add all implementations of `IProxyBehaviour`, so that when we construct as `ProxyBehaviourSet`, which has a constructor parameter of `IEnumerable<IProxyBehaviour>` all the implementations are passed in for us.

## Easier Testing

We can use the Registry feature of StructureMap to allow us to test parts of code as they would be in production.  This mostly applies to acceptance style testing, for example when I am testing the XmlRewriter, I want it to behave exactly as it would in production, with the same `IXElementTransform`s passed in.

To do this, we can use the `RewriterRegistry`:

```csharp
var container = new Container(new RewriterRegistry());
var rewriter = container.GetInstance<XmlRewriter>();
```

Here we create a new container with the `RewriterRegistry` passed directly into the constructor.  This gives us access to a container completely configured for using the `XmlRewriter`.  We can then fake the inputs and outputs to the method under test, keeping the whole system in a known production-like state.

```csharp
using (var inputStream = GetType().Assembly.GetManifestResourceStream("NuCache.Tests.Packages.xml"))
using (var outputStream = new MemoryStream())
{
	rewriter.Rewrite(targetUri, inputStream, outputStream);
	outputStream.Position = 0;

	_result = XDocument.Load(outputStream);
	_namespace = _result.Root.Name.Namespace;
}
```

Hopefully this shows how useful and powerful feature StructureMap's Registries are.


[github-nucache]: https://github.com/Pondidum/NuCache
[github-nucache-configurecontainers]: https://github.com/Pondidum/NuCache/blob/master/NuCache/App_Start/ConfigureContainer.cs
[github-nucache-infrastructureregistry]: https://github.com/Pondidum/NuCache/blob/master/NuCache/Infrastructure/InfrastructureRegistry.cs
[github-nucache-proxyregistry]: https://github.com/Pondidum/NuCache/blob/master/NuCache/ProxyBehaviour/ProxyBehaviourRegistry.cs
[structuremap-registries]: http://fubuworld.com/structuremap/registration/registry-dsl/
