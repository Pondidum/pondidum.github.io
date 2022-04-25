---
date: "2018-08-26T00:00:00Z"
tags: configuration c# strongtyping stronk validation
title: Validate Your Configuration
---

As I have [written](/2016/12/06/strong-type-all-the-configurations/) many [times](/2017/11/09/configuration-composition/) before, your application's configuration should be strongly typed and validated that it loads correctly at startup.

This means not only that the source values (typically all represented as strings) can be converted to the target types (`int`, `Uri`, `TimeSpan` etc) but that the values are **semantically valid** too.

For example, if you have a `web.config` file with the following `AppSetting`, and a configuration class to go with it:

```xml
<configuration>
  <appSettings>
    <add key="Timeout" value="20" />
  </appSettings>
</configuration>
```

```csharp
public class Configuration
{
    public TimeSpan Timeout { get; private set; }
}
```

We can now load the configuration using [Stronk](https://github.com/pondidum/stronk) (or Microsoft.Extensions.Configuration if you're on dotnet core), and inspect the contents of the `Timeout` property:

```csharp
var config = new StronkConfig().Build<Configuration>();

Console.WriteLine(config.Timeout); // 20 days, 0 hours, 0 minutes, 0 seconds
```

Oops.  **A timeout of 20 days is probably a *little* on the high side!**  The reason this happened is that to parse the string value we use `TimeSpan.Parse(value)`, which will interpret it as days if no other units are specified.

## How to validate?

There are several ways we could go about fixing this, from changing to use `TimeSpan.ParseExact`, but then we need to provide the format string from somewhere, or force people to use Stronk's own decision on format strings.

Instead, we can just write some validation logic ourselves.  If it is a simple configuration, then writing a few statements inline is probably fine:

```csharp
var config = new StronkConfig()
    .Validate.Using<Configuration>(c =>
    {
        if (c.Timeout < TimeSpan.FromSeconds(60) && c.Timeout > TimeSpan.Zero)
            throw new ArgumentOutOfRangeException(nameof(c.Timeout), $"Must be greater than 0, and less than 1 minute");
    });
    .Build<Configuration>();
```

But we can make it much clearer by using a validation library such as [FluentValidation](https://github.com/JeremySkinner/FluentValidation), to do the validation:

```csharp
var config = new StronkConfig()
    .Validate.Using<Configuration>(c => new ConfigurationValidator().ValidateAndThrow(c))
    .Build<Configuration>();
```

```csharp
public class ConfigurationValidator : AbstractValidator<Configuration>
{
    private static readonly HashSet<string> ValidHosts = new HashSet<string>(
        new[] { "localhost", "internal" },
        StringComparer.OrdinalIgnoreCase);

    public ConfigurationValidator()
    {
        RuleFor(x => x.Timeout)
            .GreaterThan(TimeSpan.Zero)
            .LessThan(TimeSpan.FromMinutes(2));

        RuleFor(x => x.Callback)
            .Must(url => url.Scheme == Uri.UriSchemeHttps)
            .Must(url => ValidHosts.Contains(url.Host));
    }
}
```

Here, not only are we checking the `Timeout` is in a valid range, but that our `Callback` is HTTPS and that it is going to a domain on an Allow-List.

## What should I validate?

Everything?  If you have properties controlling the number of threads an application uses, probably checking it's a positive number, and less than `x * Environment.ProcessorCount` (for some value of x) is probably a good idea.

If you are specifying callback URLs in the config file, checking they are in the right domain/scheme would be a good idea (e.g. must be https, must be in a domain allow-list).

How do you check your configuration isn't going to bite you when an assumption turns out to be wrong?