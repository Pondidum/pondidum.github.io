---
date: "2015-08-30T00:00:00Z"
tags: ["design", "overseer", "microservices", "console", "cli"]
title: A single project Windows Service and Console
---

I have found that when developing MicroServices, I often want to run them from within Visual Studio, or just as a console application, and not have to bother with the hassle of installing as windows services.

In the past I have seen this achieved by creating a `Class Library` project with all the actual implementation inside it, and then both a `Console Application` and `Windows Service` project referencing the library and doing nothing other than calling a `.Start()` method or similar.

While this works, it has always bugged me as there should be a straight forward way of achieving a single exe to do both roles.  It turns out there is an easy way to do it too...

## Creating the Project

First, create a `WindowsService` project in VisualStudio:
![New Windows Service][vs-new-project]

Then open the project properties, and change the project type to `Console Application` and set the startup object:
![Service Type][vs-project-type]

Next, open `Service1.cs` and add a new method (and rename it to `Service` if you feel the need!):

```csharp
public void StartConsole()
{
	Console.WriteLine("Press any key to exit...");
	OnStart(new string[] { });

	Console.ReadKey();
	OnStop();
}
```

Finally  open `Program.cs` and replace the `Main` method:

```csharp
static void Main()
{
	var service = new Service();

	if (Environment.UserInteractive)
	{
		service.StartConsole();
	}
	else
	{
		ServiceBase.Run(new ServiceBase[] { service });
	}
}
```

## Displaying Output

Calling `Console.Write*` and `Console.Read*` methods when running as a windows service will cause exceptions to be thrown, which suggest that you should redirect the console streams to use them under a windows service.

As a MicroService you shouldn't need to be reading keys from the console (other than the one in our `StartConsole` method), but writing output would be useful...

To do this I like to use my logging library of choice ([Serilog][serilog]), which I have setup to write to files and to a console:

```csharp
private void InitializeLogging()
{
	var baseDirectory = AppDomain.CurrentDomain.BaseDirectory;
	var logs = Path.Combine(baseDirectory, "logs");

	Directory.CreateDirectory(logs);

	Log.Logger = new LoggerConfiguration()
		.MinimumLevel.Debug()
		.WriteTo.ColoredConsole()
		.WriteTo.RollingFile(Path.Combine(logs, "{Date}.log"))
		.CreateLogger();
}
```

And call this method inside the `Service1` constructor:

```csharp
public Service()
{
	InitializeComponent();
	InitializeLogging();
}
```

## The Edge Case

There is one slight edge case which I am aware of, which is that the `Environment.UserInteractive
` property can return true even when running as a windows service if when you install the service you tick `Allow service to interact with desktop` checkbox:

![Service-Logon][service-logon]

My only solution to this is: **Don't tick that box**. I don't think I have ever used that option anyway!

## Wrapping Up

Using this method means less code and projects to maintain, and a very easy path to go from running a service as a desktop application to service.

[vs-new-project]: /images/service-new.png
[vs-project-type]: /images/service-project-type.png
[service-logon]: /images/service-interact.png
[serilog]: http://serilog.net/
