---
layout: post
title: Preventing MicroService Boilerplate
tags: code c# microservices consul structuremap kibana boilerplate
---

One of the downsides to microservices I have found is that I end up repeating the same blocks of code over and over for each service.  Not only that, but the project setup is repetitive, as all the services use the [Single Project Service and Console][single-project] method.

# What do we do in every service?

* Initialise Serilog.
* Add a Serilog sink to ElasticSearch for Kibana (but only in non-local config.)
* Hook/Unhook the `AppDomain.Current.UnhandledException` handler.
* Register/UnRegister with Consul.
* Setup StructureMap, if using an IOC Container.
* Run as a Console if the `Environment.UserInteractive` flag is true.
* Run as a Service otherwise

The only task with potential to have variance each time is the setting up of StructureMap, the rest are almost identical every time.

# How to solve all this repetition?

To rectify this, I created a nuget project which encapsulates all of this logic, and allows us to create a Console project with the following startup:

```c#
static void Main(string[] args)
{
	ServiceHost.Run<Startup>("TestService");
}
```

This requires one class implementing the `IStartup` interface, and there are some optional interfaces which can be implemented too:

```c#
public class Startup : IStartup, IDisposable
{
	public Startup()
	{
		Console.WriteLine("starting up");
	}

	public void Execute(ServiceArgs service)
	{
		File.AppendAllLines(Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "log.txt"), new[] { "boot!" });

		while (service.CancelRequested == false)
			Thread.Sleep(500);
	}

	public void Dispose()
	{
		Console.WriteLine("shutting down");
	}
}
```

Optionally, the project can implement two interfaces to control Consul and ElasticSearch configuration:

```c#
public class Config : ILogConfig, IConsulRegistration
{
	public bool EnableKibana { get; }
	public Uri LoggingEndpoint { get; }

	public CatalogRegistration CreateRegistration()
	{
		return new CatalogRegistration() { Service = new AgentService
		{
			Address = "http://localhost",
			Port = 8005,
			Service = "TestService"
		}};
	}

	public CatalogDeregistration CreateDeregistration()
	{
		return new CatalogDeregistration { ServiceID = "TestService" };
	}
}
```

By implementing these interfaces, the `ServiceHost` class can use StructureMap to find the implementations (if any) at run time.

Talking of StructureMap, if we wish to configure the container in the host application, all we need to do is create a class which inherits `Registry`, and the ServiceHost's StructureMap configuration will find it.

# How do we support other tools?

Well we could implment some kind of stage configuration steps, so your startup might change to look like this:

```c#
static void Main(string[] args)
{
	ServiceHost.Stages(new LoggingStage(), new ConsulStage(), new SuperAwesomeThingStage());
	ServiceHost.Run<Startup>("TestService");
}
```

The reason I haven't done this is that on the whole, we tend to use the same tools for each job in every service; StructureMap for IOC, Serilog for logging, Consul for discovery.  So rather than having to write some boilerplate for every service (e.g. specifying all the stages), I just bake the options in to `ServiceHost` directly.

This means that if you want your own version of this library with different tooling support, you need to write it yourself.  As a starting point, I have the code for the [`ServiceContainer` project up on Github][service-container].

It is not difficult to create new stages for the pipeline - all the different tasks the `ServiceHost` can perform are implemented in a pseudo Russian-Doll model - they inherit `Stage`, which looks like this:

```c#
public abstract class Stage : IDisposable
{
	public IContainer Container { get; set; }

	public abstract void Execute();
	public abstract void Dispose();
}
```

Anything you want to your stage to do before the `IStartup.Execute()` call is made is done in `Execute()`, similarly anything to be done afterwards is in `Dispose()`.  For example, the `ConsulStage` is implemented like so:

```c#
public class ConsulStage : Stage
{
	public override void Execute()
	{
		var registration = Container.TryGetInstance<IConsulRegistration>();

		if (registration != null)
		{
			var client = new ConsulClient();
			client.Catalog.Register(registration.CreateRegistration());
		}
	}

	public override void Dispose()
	{
		var registration = Container.TryGetInstance<IConsulRegistration>();

		if (registration != null)
		{
			var client = new ConsulClient();
			client.Catalog.Deregister(registration.CreateDeregistration());
		}
	}
}
```

Finally you just need to add the stage to the `ServiceWrapper` constructor:

```c#
public ServiceWrapper(string name, Type entryPoint)
{
	// snip...

	_stages = new Stage[]
	{
		new LoggingStage(name),
		new ConsulStage()
	};
}
```

# Get started!

That's all there is to it!  Hopefully this gives you a good starting point for de-boilerplating your microservices :)


[single-project]: /2015/08/30/single-project-service-and-console/
[service-container]: https://github.com/pondidum/ServiceContainer
