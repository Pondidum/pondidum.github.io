---
layout: post
title: Integration Testing with Dotnet Core, Docker and RabbitMQ
tags: code dotnetcore rabbitmq docker testing
---

When building libraries, not only is it a good idea to have a large suite of Unit Tests, but also a suite of Integration Tests.

For one of my libraries ([RabbitHarness](https://github.com/pondidum/rabbitharness)) I have a set of tests which check it behaves as expected against a real instance of [RabbitMQ](http://www.rabbitmq.com/).  Ideally these tests will always be run, but sometimes RabbitMQ just isn't available such as when running on [AppVeyor](https://ci.appveyor.com/project/Pondidum/rabbitharness) builds, or if I haven't started my local RabbitMQ Docker container.

## Skipping tests if RabbitMQ is not available

First off, I prevent the tests from running if RabbitMQ is not available by using a custom [XUnit](https://xunit.github.io/) `FactAttribute`:

```csharp
public class RequiresRabbitFactAttribute : FactAttribute
{
	private static readonly Lazy<bool> IsAvailable = new Lazy<bool>(() =>
	{
		var factory = new ConnectionFactory { HostName = "localhost", RequestedConnectionTimeout = 1000 };

		try
		{
			using (var connection = factory.CreateConnection())
				return connection.IsOpen;
		}
		catch (Exception)
		{
			return false;
		}
	});

	public override string Skip
	{
		get { return IsAvailable.Value ? "" : "RabbitMQ is not available";  }
		set { /* nothing */ }
	}
}
```

This attribute will try connecting to a RabbitMQ instance on `localhost` once for all tests per run, and cause any test with this attribute to be skipped if RabbitMQ is not available.

## Build Script & Docker

I decided the build script should start a RabbitMQ container, and use that for the tests, but I didn't want to re-use my standard RabbitMQ instance which I use for all kinds of things, and may well be broken at any given time.

As my build script is just a `bash` script, I can check if the `docker` command is available, and then start a container if it is (relying on the assumption that if `docker` is available, I can start a container).

```bash
if [ -x "$(command -v docker)" ]; then
  CONTAINER=$(docker run -d --rm -p 5672:5672 rabbitmq:3.6.11-alpine)
  echo "Started RabbitMQ container: $CONTAINER"
fi
```

If `docker` is available, we start a new container.  I use `rabbitmq:3.6.11-alpine` as it is a tiny image, with no frills, and also start it with the `-d` and `--rm` flags, which starts the container in a disconnected mode (e.g. the `docker run` command returns instantly), and will delete the container when it is stopped, taking care of clean up for us! I only bother binding the main data connection port (`5672`), as that is all we are going to be using. Finally, the container's ID, which is returned by the `docker run` command, is stored in the `CONTAINER` variable.

I recommend putting this step as the very first part of your build script, as it gives the container time to start up RabbitMQ and be ready for connections while your build is running.  Otherwise I found I was needing to put a `sleep 5` command in afterwards to pause the script for a short time.

The script then continues on with the normal build process:

```bash
dotnet restore "$NAME.sln"
dotnet build "$NAME.sln" --configuration $MODE

find . -iname "*.Tests.csproj" -type f -exec dotnet test "{}" --configuration $MODE \;
dotnet pack ./src/$NAME --configuration $MODE --output ../../.build
```

Once this is all done, I have another check that `docker` exists, and stop the container we started earlier, by using the container ID in `CONTAINER`:

```bash
if [ -x "$(command -v docker)" ]; then
  docker stop $CONTAINER
fi
```

And that's it!  You can see the full [build script for RabbitHarness here](https://github.com/Pondidum/RabbitHarness/blob/master/build.sh).

The only problem with this script is if you try and start a RabbitMQ container while you already have one running, the command will fail, but the build should succeed anyway as the running instance of RabbitMQ will work for the tests, and the `docker stop` command will just output that it can't find a container with a blank ID.

I think I will be using this technique more to help provide isolation for builds - I think that the [Microsoft/mssql-server-linux](https://hub.docker.com/r/microsoft/mssql-server-linux/) containers might be very useful for some of our work codebases (which do work against the Linux instances of MSSQL, even if they weren't designed to!)
