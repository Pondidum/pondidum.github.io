---
date: "2019-06-03T00:00:00Z"
tags: featuretoggles c# di microservices
title: 'Feature Toggles: Branch by Abstraction'
---

Recently, I was asked if I could provide an example of Branch By Abstraction when dealing with feature toggles.  As this has come up a few times, I thought a blog post would be a good idea so I can refer others to it later too.

## The Context

As usual, this is some kind of backend (micro)service, and it will send email messages somehow.  We will start with two implementations of message sending: the "current" version; which is synchronous, and a "new" version; which is async.

We'll do a bit of setup to show how feature toggling can be done in three ways for this feature:

1. Static: Configured on startup
1. Dynamic: Check the toggle state on each send
1. Dynamic: Check the toggle for a given message

## Abstractions and Implementations

We have an interface called `IMessageDispatcher` which defines a single `Send` method, which returns a `Task` (or `Promise`, `Future`, etc. depending on your language.)

```csharp
public interface IMessageDispatcher
{
    Task<SendResult> Send(Message message);
}
```

The two message sending implementations don't matter, but we need the types to show the other code examples.  Fill in the blanks if you want!

```csharp
public class HttpMessageDispatcher : IMessageDispatcher
{
    // ...
}

public class QueueMessageDispatcher : IMessageDispatcher
{
    // ...
}
```

## 1. Static Configuration

The word static in this context means that we check the feature toggle's state once on startup and pick an implementation.  We don't recheck the toggle state unless the service is restarted.

For instance, in an ASP.Net core application, you could change which service is registered into the container at startup like so:

```csharp
public void ConfigureServices(IServiceCollection services)
{
    var toggleSource = new ToggleSource(/* ... */);

    if (toggleSource.IsActive(Toggles.AsyncMessageDispatch))
        services.AddTransient<IMessageDispatcher, QueueMessageDispatcher>();
    else
        services.AddTransient<IMessageDispatcher, HttpMessageDispatcher>();
}
```

Which means any class which takes in an instance of `IMessageDispatcher` doesn't need to check the toggle state or worry about which implementation to use.

## 2. Dynamic Configuration

We can build on this abstraction to enable more flexibility, if we want to be able to change the toggle state while the service is running, without needing to restart it.  To do this, we can implement another version of the `IMessageDispatcher` interface which will check the toggle state on each invocation of `Send()`:

```csharp
public class ToggleDispatcher : IMessageDispatcher
{
    private readonly Func<bool> _isToggleActive;
    private readonly IMessageDispatcher _queueSender;
    private readonly IMessageDispatcher _httpSender;

    public ToggleDispatcher(Func<bool> isToggleActive, IMessageDispatcher queueSender, IMessageDispatcher httpSender)
    {
        _isToggleActive = isToggleActive;
        _queueSender = queueSender;
        _httpSender = httpSender;
    }

    public Task<SendResult> Send(Message message)
    {
        var chosen = _isToggleActive()
            ? _queueSender
            : _httpSender;

        return chosen.Send(message);
    }
}
```

And in our startup class, we can change the service registration to use the new version.  Note how we are now registering the two concrete versions into the container so that they can be resolved later by the ToggleDispatcher registration:

```csharp
public void ConfigureServices(IServiceCollection services)
{
    var toggleSource = new ToggleSource(/* ... */);

    services.AddTransient<HttpMessageDispatcher>();
    services.AddTransient<QueueMessageDispatcher>();

    services.AddTransient<IMessageDispatcher>(context => new ToggleDispatcher(
        () => toggleSource.IsActive(Toggles.AsyncMessageDispatch),
        context.GetService<QueueMessageDispatcher>(),
        context.GetService<HttpMessageDispatcher>())
    );
}
```

## 3. Dynamic(er) Configuration

We can take this another step further too, if we want to be able to have a phased rollout of this new `QueueMessageDispatcher`, for example, based on the sender address.  In this case, we can create another decorator which uses the individual message to make the decision.  The only difference to the original `ToggleDispatcher` is that the first argument now also provides a `Message` object:


```csharp
public class MessageBasedToggleDispatcher : IMessageDispatcher
{
    private readonly Func<Message, bool> _isToggleActive;
    private readonly IMessageDispatcher _queueSender;
    private readonly IMessageDispatcher _httpSender;

    public MessageBasedToggleDispatcher(Func<Message, bool> isToggleActive, IMessageDispatcher queueSender, IMessageDispatcher httpSender)
    {
        _isToggleActive = isToggleActive;
        _queueSender = queueSender;
        _httpSender = httpSender;
    }

    public Task<SendResult> Send(Message message)
    {
        var chosen = _isToggleActive(message)
            ? _queueSender
            : _httpSender;

        return chosen.Send(message);
    }
}
```

The startup registration is modified to pass the message property we care about to the `ToggleSource`, with the `toggleSource.IsActive()` call being responsible for what to do with the key we have passed in.  Perhaps it does something like a consistent hash of the address, and if the value is above a certain threshold the toggle is active, or maybe it queries a whitelist of people who the toggle is enabled for.

```csharp
public void ConfigureServices(IServiceCollection services)
{
    var toggleSource = new ToggleSource(/* ... */);

    services.AddTransient<HttpMessageDispatcher>();
    services.AddTransient<QueueMessageDispatcher>();

    services.AddTransient<IMessageDispatcher>(context => new MessageBasedToggleDispatcher(
        message => toggleSource.IsActive(Toggles.AsyncMessageDispatch, message.SenderAddress),
        context.GetService<QueueMessageDispatcher>(),
        context.GetService<HttpMessageDispatcher>())
    );
}
```

## Conclusion

This method of branching is extremly flexible, as it allows us to use toggles to replace feature implementations, but also gives us lots of places where we can add other decorators to add functionality to the pipeline.  For example, we could add an auditing decorator or one which implements the outbox pattern - and the calling code which depends only on `IMessageDispatcher` doesn't need to care.
