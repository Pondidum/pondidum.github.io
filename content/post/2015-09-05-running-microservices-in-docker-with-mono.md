+++
date = '2015-09-05T00:00:00Z'
tags = ['design', 'microservices', 'docker', 'mono']
title = 'Running microservices in Docker with Mono'

+++

Getting a service running under [Docker][docker] is fairly straight forward once you have all the working parts together.  I have an app written (following [my guide][blog-serviceconsole] on service and console in one), which uses Owin to serve a web page as a demo:


```powershell
install-package Microsoft.Owin.SelfHost
```

```csharp
public partial class Service : ServiceBase
{
  //see the service console post for the rest of this

	protected override void OnStart(string[] args)
	{
		_app = WebApp.Start("http://*:12345", app =>
		{
			app.UseWelcomePage("/");
		});
	}

	protected override void OnStop()
	{
		_app.Dispose();
	}
}
```

To run this under docker/mono we just need to add a `Dockerfile` to the root directory of the solution, which is based off the [documentation here][docker-mono].

Using `mono-service` instead of `mono` to run the application caused me a number of headaches to start with, as the container was exiting instantly.  This is because Docker detects the process has exited, and stops the container.  As we will be running the container detached from the console, we just need to supply the `--no-daemon` argument to `mono-service`.

```
FROM mono:3.10-onbuild
RUN apt-get update && apt-get install mono-4.0-service -y
CMD [ "mono-service",  "./MicroServiceDemo.exe", "--no-daemon" ]
EXPOSE 12345
```

You can then go to your solution directory, and run the following two commands to create your image, and start a container of it:

```bash
docker build -t servicedemo .
docker run -d -p 12345:12345 --name demo servicedemo
```

You can now open your browser and go to your Docker host's IP:12345 and see the Owin welcome page.

## Improvements: Speed and lack of internet

Quite often I have no internet access, so having to `apt-get install mono-4.0-service` each time I build the image can be a pain.  This however is also very easily resolved: by making another image with the package already installed.

Create a new directory (outside of your project directory), and create a `Dockerfile`.  This Dockerfile is identical to the [mono:3.10-onbuild][mono-onbuild-dockerfile] image, but with the added apt-get line.

```
FROM mono:3.10.0

MAINTAINER Jo Shields <jo.shields@xamarin.com>

RUN apt-get update && apt-get install mono-4.0-service -y

RUN mkdir -p /usr/src/app/source /usr/src/app/build
WORKDIR /usr/src/app/source

ONBUILD COPY . /usr/src/app/source
ONBUILD RUN nuget restore -NonInteractive
ONBUILD RUN xbuild /property:Configuration=Release /property:OutDir=/usr/src/app/build/
ONBUILD WORKDIR /usr/src/app/build
```

Now run the build command to make your new base image:

```bash
docker build -t mono-service-onbuild .
```

Now you can go back to your project and update the `Dockerfile` to use this image base instead:

```
FROM mono-service-onbuild
CMD [ "mono-service",  "./MicroServiceDemo.exe", "--no-daemon" ]
EXPOSE 12345
```

Now when you run `docker build -t <project name> .` it will only need to do the compile steps.

Much faster :)

[docker]: https://www.docker.com
[docker-mono]: https://hub.docker.com/_/mono
[blog-serviceconsole]: /2015/08/30/single-project-service-and-console.html
[mono-onbuild-dockerfile]: https://github.com/mono/docker/blob/adc7a3ec47f7d590f75a4dec0203a2103daf8db0/3.10.0/onbuild/Dockerfile
