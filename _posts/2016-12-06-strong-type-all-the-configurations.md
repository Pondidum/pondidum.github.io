---
layout: post
title: Strong Type All The Configurations
tags: code c# strongtyping configuration stronk
---

As anyone I work with can attest, I a have been prattling on about strong typing everything for quite a while.
One of the places I feel people don't utilise strong typing enough is in application configuration.  This manifests in a number of problems in a codebase.

## The Problems

The first problem is when nothing at all is done about it, and you end up with code spattered with this:

```csharp
var someUrl = new Uri(ConfigurationManager.AppSettings["RemoteService"]);
```

This itself causes a few problems:

* **Repeated:** You have magic strings throughout your codebase
* **Consistency:** Was it `RemoteService` or `RemoteServiceUri`. Or was it in `ConnectionStrings` or `AppSettings`?
* **Visibility:** Can you tell which classes require on which (if any) configuration values?
* **Typing:** Was it actually a URL? or was it DNS entry?
* **Late errors:** You will only find out once that particular piece of code runs
* **Tight Coupling:** Tests won't help either, as they'll be reading your test's `app.config` instead...

## Solution: Version 1

The first solution involves abstracting the `ConfigurationManager` behind a general interface, which can be injected into classes requiring configuration values.  The interface is usually along the following lines:

```csharp
public interface ISettings
{
    string GetString(string key);
    Uri GetUri(string key);
    // GetInt, GetShort, etc.
}
```

And having an implementation which uses the `ConfigurationManager` directly:

```csharp
public class Settings : ISettings
{
    public string GetString(string key) => ConfigurationManager.AppSettings[key];
    public Uri GetUri(string key) => new Uri(ConfigurationManager.AppSettings[key]);
}
```
This solves one of the problems of direct usage of the `ConfigurationManager`, namely **Tight Coupling**.  By using an interface we can now use [NSubstitute](http://nsubstitute.github.io/) or similar mocking library to disconnect tests from `app.config` and `web.config`.

It doesn't really solve the **Typing** issue however, as the casting is only done on fetching the configuration value, and so errors in casting still only happen when the code is executed.  It also doesn't really solve the **Discoverability** issue either - you can now tell if a class requires configuration values, but you cannot tell which values it requires from outside.

The other issues such as **Repeatablility**, **Late Errors** and **Consistency** are not addressed by this method at all.

## Solution: Version 2

My preferred method of solving all of these problems is to replace direct usage of `ConfigurationManager` with an interface & class pair, but with the abstraction being application specific, rather than general.  For example, at application might have this as the interface:

```csharp
public interface IConfiguration
{
    string ApplicationName { get; }
    Uri RemoteHost { get; }
    int TimeoutSeconds { get; }
}
```

This would then be implemented by a concrete class:

```csharp
public class Configuration : IConfiguration
{
    public string ApplicationName { get; }
    public Uri RemoteHost { get; }
    public int TimeoutSeconds { get; }

    public Configuration()
    {
        ApplicationName = ConfigurationManager.AppSetting[nameof(ApplicationName)];
        RemoteHost = new Uri(ConfigurationManager.AppSetting[nameof(RemoteHost)]);
        TimeoutSeconds = (int)ConfigurationManager.AppSetting[nameof(TimeoutSeconds)];
    }
}
```

This method solves all of the first listed problems:

**Repeated** and **Consistency** are solved, as the only repetition is the usage of configuration properties themselves.  **Visibility** is solved as you can now either use "Find Usages" on a property, or you can split your configuration `interface` to have a specific set of properties for each class which is going to need configuration.

**Typing** and **Late errors** are solved as all properties are populated on the first creation of the class, and exceptions are thrown immediately if there are any type errors.

**Tight Coupling** is also solved, as you can fake the entire `IConfiguration` interface for testing with, or just the properties required for a given test.

The only down side is the amount of writing needed to make the constructor, and having to do the same code in every application you write.

## Solution: Version 3

The third solution works exactly as the 2nd solution, but uses the [Stronk Nuget library](https://www.nuget.org/packages/stronk) to populate the configuration object.  **Stronk** takes all the heavy lifting out of configuration reading, and works for most cases with zero extra configuration required.

```csharp
public interface IConfiguration
{
    string ApplicationName { get; }
    Uri RemoteHost { get; }
    int TimeoutSeconds { get; }
}

public class Configuration : IConfiguration
{
    public string ApplicationName { get; }
    public Uri RemoteHost { get; }
    public int TimeoutSeconds { get; }

    public Configuration()
    {
        this.FromAppConfig(); //this.FromWebConfig() works too
    }
}
```

**Stronk** supports a lot of customisation.  For example, if you wanted to be able to handle populating properties of type `MailAddress`, you can add it like so:

```csharp
public Configuration()
{
    var mailConverter = new LambdaValueConverter<MailAddress>(val => new MailAddress(val));
    var options = new StronkOptions();
    options.Add(mailConverter);

    this.FromAppConfig(options);
}
```

You can also replace (or supplement):

* How it detects which properties to populate
* How to populate a property
* How to pick a value from the configuration source for a given property
* How to convert a value for a property
* Where configuration is read from

A few features to come soon:

* Additional types supported "out of the box" (such as `TimeSpan` and `DateTime`)
* Exception policy controlling:
    * What happens on not being able to find a value in the configuration source
    * What happens on not being able to find a converter
    * What happens on a converter throwing an exception

I hope you find it useful.  [Stronk's Source is available on Github](https://github.com/Pondidum/Stronk/), and contributions are welcome :)
