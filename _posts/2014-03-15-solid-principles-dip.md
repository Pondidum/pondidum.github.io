---
layout: post
title: SOLID Principles - DIP
tags: design, code, net
permalink: solid-principles-dip
---


[Single Responsibility][blog-solid-srp] | [Open Closed][blog-solid-ocp] | [Liskov Substitution][blog-solid-lsp] | [Interface Segregation][blog-solid-isp] | [Dependency Inversion][blog-solid-dip]

The Dependency Inversion Principle states that "Depend upon Abstractions. Do not depend upon concretions".  A good real world example of this is plug sockets around your house; any device you buy can be plugged into any socket in your house.  You don't have to buy new set of devices when you move house, and you don't have to buy a new house for your devices!

In software terms this means that our higher level classes should not directly depend on lower level classes, but should depend on some intermediary.  The same goes for depending on external resources.  For example, if you have this class which takes a request string, and deserializes it, and does something with the resulting object:

{% highlight c# %}
public class RequestHandler
{
		public void OnRequestReceived(string json)
		{
				var data = NewtonSoftJson.Deserialize<RequestData>(json);

				Console.WriteLine(data.Name + " Received.");
		}
}
{% endhighlight %}

This has two problems - the first is that it is totally dependant on the `NewtonSoftJson` class which means we are in violation of the Dependency Inversion Principle, and also we are tied to a specific provider.  We also are using a static method on the `NewtonSoftJson` class, which makes the method impossible to test, if we didn't want to depend on `NewtonSoftJson` for our test.

We can move towards fixing both of these problems by adding an interface, and depending on that for serialization instead:

{% highlight c# %}
public interface IJsonSerializer
{
		T Deserialize<T>(string json);
}

public class JsonSerializer : IJsonSerializer
{
		public T Deserialize<T>(string json)
		{
				return NewtonSoftJson.Deserialize<T>(json);
		}
}

public class RequestHandler
{
		private readonly IJsonSerializer _serializer;

		public RequestHandler(IJsonSerializer serializer)
		{
				_serializer = serializer;
		}

		public void OnRequestReceived(string json)
		{
				var data = _serializer.Deserialize<RequestData>(json);

				Console.WriteLine(data.Name + " Received.");
		}
}
{% endhighlight %}

By doing this, the `RequestHandler` class is now dependant on an abstraction rather than a concretion.  This nets us many benefits:  We are no longer directly dependant on `NewtonSoftJson`, our `OnRequestReceived` method has become more testable, and we have also centralised our json serialization logic.

This means that if we wish to change to a different library for json serialization (or use the `JavaScriptSerializer` built into the .net framework) we can just create a new class which implements `IJsonSerializer` and pass an instance of the new class to `RequestHandler`.  It also means that anywhere we want to do json serialization can just take an `IJsonSerializer` in as a dependency, and not care what the dependency is actually doing when `Deserialize` is called.

Hopefully this explains a little more on how inverting your dependencies can help make your software more flexible, and more maintainable.

All source code is available on my Github: [Solid.Demo Source Code][solid-demo-repo]

[blog-solid-srp]: http://andydote.co.uk/solid-principles-srp
[blog-solid-ocp]: http://andydote.co.uk/solid-principles-ocp
[blog-solid-lsp]: http://andydote.co.uk/solid-principles-lsp
[blog-solid-isp]: http://andydote.co.uk/solid-principles-isp
[blog-solid-dip]: http://andydote.co.uk/solid-principles-dip
[solid-demo-repo]: https://github.com/Pondidum/Solid.Demo
