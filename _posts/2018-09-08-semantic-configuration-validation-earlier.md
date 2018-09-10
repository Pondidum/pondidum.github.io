---
layout: post
title: "Semantic Configuration Validation: Earlier"
tags: configuration c# strongtyping stronk validation
---

After my previous post on [Validating Your Configuration](/2018/08/26/validate-configuration/), one of my colleagues made an interesting point, paraphrasing:

> I want to know if the configuration is valid earlier than that.  At build time preferably.  I don't want my service to not start if part of it is invalid.

There are two points here, namely when to validate, and what to do with the results of validation.

## Handling Validation Results

If your configuration is invalid, you'd think the service should fail to start, as it might be configured in a dangerous manner.  While this makes sense for some service, others might need to work differently.

Say you have an API which supports both writing and reading of a certain type of resource.  The read will return you a resource of some form, and the write side will trigger processing of a resource (and return you [a 202 Accepted, obviously](https://httpstatuses.com/202)).

What happens if your configuration just affects the write side of the API? Should you prevent people from reading too?  Probably not, but again it depends on your domain as to what makes sense.

## Validating at Build Time

This is the far more interesting point (to me).  How can we modify our build to validate that the environment's configuration is valid?  We have the code to do the validation: we have automated tests, and we have a configuration validator class (in this example, implemented using [FluentValidation](https://github.com/JeremySkinner/FluentValidation)).

Depending on where your master configuration is stored, the next step can get much harder.

### Local Configuration

If your configuration is in the current repository ([as it should be](/2018/08/07/managing-consul-appsettings/)) then it will be no problem to read.

```csharp
public class ConfigurationTests
{
    public static IEnumerable<object[]> AvailableEnvironments => Enum
        .GetValues(typeof(Environments))
        .Cast<Environments>()
        .Select(e => new object[] { e });

    [Theory]
    [MemberData(nameof(AvailableEnvironments))]
    public void Environment_specific_configuration_is_valid(Environments environment)
    {
        var config = new ConfigurationBuilder()
            .AddJsonFile("config.json")
            .AddJsonFile($"config.{environment}.json", optional: true)
            .Build()
            .Get<AppConfiguration>();

        var validator = new AppConfigurationValidator();
        validator.ValidateAndThrow(config);
    }
}
```

Given the following two configuration files, we can make it pass and fail:

`config.json:`
```json
{
  "Callback": "https://localhost",
  "Timeout": "00:00:30",
  "MaxRetries": 100
}
```

`config.local.json:`
```json
{
  "MaxRetries": 0
}
```

### Remote Configuration

But what if your configuration is not in the local repository, or at least, not completely there?  For example, have a lot of configuration in Octopus Deploy, and would like to validate that at build time too.

Luckily Octopus has a Rest API (and [acompanying client](https://www.nuget.org/packages/Octopus.Client/))  which you can use to query the values.  All we need to do is replace the `AddJsonFile` calls with an `AddInMemoryCollection()` and populate a dictionary from somewhere:

```csharp
[Theory]
[MemberData(nameof(AvailableEnvironments))]
public async Task Octopus_environment_configuration_is_valid(Environments environment)
{
    var variables = await FetchVariablesFromOctopus(
        "MyDeploymentProjectName",
        environment);

    var config = new ConfigurationBuilder()
        .AddInMemoryCollection(variables)
        .Build()
        .Get<AppConfiguration>();

    var validator = new AppConfigurationValidator();
    validator.ValidateAndThrow(config);
}
```

Reading the variables from Octopus' API requires a bit of work as you don't appear to be able to ask for all variables which would apply if you deployed to a specific environment, which forces you into building the logic yourself.  However, if you are just using Environment scoping, it shouldn't be too hard.

### Time Delays

Verifying the configuration at build time when your state is fetched from a remote store is not going to solve all your problems, as this little diagram illustrates:

![test pass, a user changes value, deployment happens, startup fails](/images/versioning-time.png)

You need to validate in both places: early on in your process, and on startup.  How you handle the configuration being invalid doesn't have to be the same in both places:

* In the build/test phase, fail the build
* On startup, raise an alarm, but start if reasonable

Again, how you handle the configuration errors when your application is starting is down to your domain, and what your application does.