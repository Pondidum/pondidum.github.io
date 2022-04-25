---
date: "2017-01-07T00:00:00Z"
tags: ["c#", "cqs", "cqrs", "mediatr"]
title: MediatR and Magic
---

Having recently watched Greg Young's excellent talk on [8 Lines of Code][8-lines] I was thinking about how this kind of thinking applies to the mediator pattern, and specifically the [MediatR][mediatr] implementation.

I have written about the advantages of [CQRS with MediatR][self-mediatr] before, but having used it for a long time now, there are some parts which cause friction on a regular basis.


## The problems

### Discoverability

The biggest issue first.  You have a controller with the following constructor:

```csharp
public AddressController(IMediator mediator)
{
    _mediator = mediator;
}
```

What messages does it emit? What handlers are used by it?  No idea without grepping for `_mediator.`

### Where is the hander for X?

So you have a controller with a method which sends a `GetAllAddressesQuery`:

```csharp
public class AddressController : ApiController
{
    public IEnumerable<Address> Get()
    {
        return _mediator.Send(new GetAllAddressesQuery(User));
    }
}
```

The fastest way to get to the handler definition is to hit `Ctrl+T` and type in `GetAllAddressesQueryHandler`.  This becomes more problematic on larger codebases when you can end up with many handlers with similar names.

### What calls {command|query}Handler?

Given the following handler, what uses it?

```csharp
public class GetAllAddressesQueryHandler : IRequestHandler<GetAllAddressesQuery, IEnumerable<Address>>
{
    public IEnumerable<Address> Handle(GetAllAddressesQuery message)
    {
        //...
    }
}
```

With this problem you can use `Find Usages` on the `GetAllAddressesQuery` type parameter to find what calls it, so this isn't so bad at all.  The main problem is I am often doing `Find Usages` on the handler itself, not the message.

## Solutions

### Discoverability

The team I am on at work felt this problem a lot before I joined, and had decided to role their own mediation pipeline.  It works much the same as MediatR, but rather than injecting an `IMediator` interface into the constructor, you inject interface(s) representing the handler(s) being used:

```csharp
public AddressController(IGetAllAddressesQueryHandler getHandler, IAddAddressHandler addHandler)
{
    _getHandler = getHandler;
    _addHandler = addHandler;
}
```

The trade-offs made by this method are:

* The controllers are now more tightly coupled to the handlers (Handlers are mostly used by 1 controller anyway)
* We can't easily do multicast messages (We almost never need to do this)
* More types are required (the interface) for your handler (so what?)

On the whole, I think this is a pretty good trade-off to be made, we get all the discoverability we wanted, and our controllers and handlers are still testable.


### What calls/Where is {command|query}Handler?

This is also solved by the switch to our internal library, but we also augment the change by grouping everything into functionality groups:

```
Frontend
  Adddress
    AddressController.cs
    GetAllAddressesQuery.cs
    GetAllAddressesQueryHandler.cs
    IGetAllAddressesQueryHandler.cs
  Contact
    ContactController.cs
    ...
  Startup.cs
  project.json
```

I happen to prefer this structure to a folder for each role (e.g. `controllers`, `messages`, `handlers`), so this is not a hard change to make for me.


## Magic

As Greg noted in his video, the second you take in a 3rd party library, it's code you own (or are responsible for).  The changes we have made have really just traded some 3rd party magic for some internal magic.  How the handler pipeline gets constructed can be a mystery still (unless you go digging through the library), but it's a mystery we control.

The important part of this to note is that we felt a pain/friction with how we are working, and decided to change what trade-offs we were making.

What trade-offs are you making?  Is it worth changing the deal?


[8-lines]: https://www.infoq.com/presentations/8-lines-code-refactoring
[mediatr]: https://github.com/jbogard/MediatR
[self-mediatr]: /2016/03/19/cqs-with-mediatr/
