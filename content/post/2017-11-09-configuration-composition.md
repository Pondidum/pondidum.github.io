---
layout: post
title: Strong Configuration Composition
tags: configuration design architecture c# strongtyping stronk
---

It's no secret I am a fan of strong typing - not only do I talk and blog about it a lot, but I also have a library called [Stronk](https://github.com/pondidum/stronk) which provides strong typed configuration for non dotnet core projects.

The problem I come across often is large configurations.  For example, given the following project structure (3 applications, all reference the Domain project):

```
DemoService
`-- src
    |-- Domain
    |   |-- Domain.csproj
    |   `-- IConfiguration.cs
    |-- QueueConsumer
    |   |-- app.config
    |   |-- QueueConsumerConfiguration.cs
    |   `-- QueueConsumer.csproj
    |-- RestApi
    |   |-- RestConfiguration.cs
    |   |-- RestApi.csproj
    |   `-- web.config
    `-- Worker
        |-- app.config
        |-- WorkerConfiguration.cs
        `-- Worker.csproj
```

The configuration defined in the domain will look something like this:

```csharp
public interface IConfiguration
{
    string ApplicationName { get; }
    string LogPath { get; }
    Uri MetricsEndpoint { get; }

    Uri DocumentsEndpoint { get; }
    Uri ArchivalEndpoint { get; }

    string RabbitMqUsername { get; }
    string RabbitMqPassword { get; }
    string RabbitMqVHost { get; }

    string BulkQueue { get; }
    string DirectQueue { get; }
    string NotificationsQueue { get; }

    Uri RabbitMqConnection { get; }
    string DatabaseConnection { get; }
    string CacheConnection { get; }
}
```

There are a number of problems with this configuration:

First off, it lives in the `Domain` project, which kinda makes sense, as things in there need access to some of the properties - but none of them need to know the name of the Queue being listened to, or where the metrics are being written to.

Next, and also somewhat related to the first point, is that all the entry projects (`RestApi`, `QueueConsumer` and `Worker`) need to supply all the configuration values, and you can't tell at a glance which projects actually need which values.

Finally, classes which use this configuration are less externally discoverable.  For example, which properties does this need: `new DocumentDeduplicator(new Configuration())`? Probably the cache? Maybe the database? or possibly the DocumentsEndpoint?  Who knows without opening the class.

## The Solution

The key to solving this is the Interface Segregation Principal - the I in SOLID.  First we need to split the interface into logical parts, which will allow our consuming classes to only take in the configuration they require, rather than the whole thing:

```csharp
public interface IRabbitConfiguration
{
    Uri RabbitMqConnection { get; }

    string RabbitMqUsername { get; }
    string RabbitMqPassword { get; }
    string RabbitMqVHost { get; }

    string BulkQueue { get; }
    string DirectQueue { get; }
    string NotificationsQueue { get; }
}

public interface IDeduplicationConfiguration
{
    Uri DocumentsEndpoint { get; }
    string CacheConnection { get; }
}

public interface IStorageConfiguration
{
    Uri ArchivalEndpoint { get; }
    string DatabaseConnection { get; }
}
```

We can also move the `IRabbitConfiguration` and `IDeduplicationConfiguration` out of the domain project, and into the `QueueConsumer` and `Worker` projects respectively, as they are only used by types in these projects:

```
DemoService
`-- src
    |-- Domain
    |   |-- Domain.csproj
    |   `-- IStorageConfiguration.cs
    |-- QueueConsumer
    |   |-- app.config
    |   |-- IRabbitConfiguration.cs
    |   |-- QueueConsumerConfiguration.cs
    |   `-- QueueConsumer.csproj
    |-- RestApi
    |   |-- RestConfiguration.cs
    |   |-- RestApi.csproj
    |   `-- web.config
    `-- Worker
        |-- app.config
        |-- IDeduplicationConfiguration.cs
        |-- WorkerConfiguration.cs
        `-- Worker.csproj
```

Next we can create some top-level configuration interfaces, which compose the relevant configuration interfaces for a project (e.g. the `RestApi` doesn't need `IDeduplicationConfiguration` or `IRabbitConfiguration`):

```csharp
public interface IWorkerConfiguration : IStorageConfiguration, IDeduplicationConfiguration
{
    string ApplicationName { get; }
    string LogPath { get; }
    Uri MetricsEndpoint { get; }
}

public interface IRestConfiguration : IStorageConfiguration
{
    string ApplicationName { get; }
    string LogPath { get; }
    Uri MetricsEndpoint { get; }
}

public interface IQueueConsumerConfiguration : IStorageConfiguration, IRabbitConfiguration
{
    string ApplicationName { get; }
    string LogPath { get; }
    Uri MetricsEndpoint { get; }
}
```

Note how we have also not created a central interface for the application configuration - this is because the application configuration is specific to each entry project, and has no need to be passed on to the domain.

Finally, an actual configuration class can be implemented (in this case using [Stronk](https://github.com/pondidum/stronk), but if you are on dotnet core, the inbuilt configuration builder is fine):

```csharp
public class QueueConsumerConfiguration : IQueueConsumerConfiguration
{
    string ApplicationName { get; private set; }
    string LogPath { get; private set; }
    Uri MetricsEndpoint { get; private set; }

    Uri ArchivalEndpoint { get; private set; }
    string DatabaseConnection { get; private set; }
    Uri RabbitMqConnection { get; private set; }

    string RabbitMqUsername { get; private set; }
    string RabbitMqPassword { get; private set; }
    string RabbitMqVHost { get; private set; }

    string BulkQueue { get; private set; }
    string DirectQueue { get; private set; }
    string NotificationsQueue { get; private set; }

    public QueueConsumerConfiguration()
    {
        this.FromAppConfig();
    }
}
```

And our startup class might look something like this (using [StructureMap](http://structuremap.github.io/)):


```csharp
public class Startup : IDisposable
{
    private readonly Container _container;
    private readonly IConsumer _consumer;

    public Startup(IQueueConsumerConfiguration config)
    {
        ConfigureLogging(config);
        ConfigureMetrics(config);

        _container = new Container(_ =>
        {
            _.Scan(a => {
                a.TheCallingAssembly();
                a.LookForRegistries();
            })

            _.For<IQueueConsumerConfiguration>().Use(config);
            _.For<IStorageConfiguration>().Use(config);
            _.For<IRabbitConfiguration>().Use(config);
        });

        _consumer = _container.GetInstance<IConsumer>();
    }

    public async Task Start() => await _consumer.Start();
    public async Task Stop() => await _consumer.Stop();

    private void ConfigureLogging(IQueueConsumerConfiguration config) { /* ... */ }
    private void ConfigureMetrics(IQueueConsumerConfiguration config) { /* ... */ }

    public void Dispose() => _container.Dispose();
}
```

As our `Startup` takes in the top-level configuration interface, if we want to write a test which tests our entire system, it can be done with a single mocked configuration object:

```csharp
[Fact]
public async Task When_the_entire_system_is_run()
{
    var config = Substitute.For<IQueueConsumerConfiguration>();
    config.RabbitMqConnection.Returns(new Uri("localhost:5672"));
    // etc.

    var startup = new Startup(config);
    await startup.Start();
    await startup.Stop();
}
```

## One Final Thing

Even if you have a microservice type project with only the one csproj, I would still recommend splitting your configuration into small interfaces, just due to the discoverability it provides.

How do you do configuration?