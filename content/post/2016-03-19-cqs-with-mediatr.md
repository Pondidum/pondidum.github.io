---
layout: post
title: CQS with Mediatr
tags: c# cqs cqrs mediatr
---

This article is some extra thoughts I had on api structure after reading [Derek Comartin][derek-comartin-thin-controllers].

Asides from the benefits that Derek mentions (no fat repositories, thin controllers), there are a number of other advantages that this style of architecture brings.

## Ease of Testing

By using Command and Queries, you end up with some very useful seams for writing tests.

### For controllers
With controllers, you typically use Dependency injection to provide an instance of `IMediator`:

```csharp
public class AddressController : ApiController
{
    private readonly IMediator _mediator;

    public AddressController(IMediator mediator)
    {
        _mediator = mediator;
    }

    public IEnumerable<Address> Get()
    {
        return _mediator.Send(new GetAllAddressesQuery(User));
    }
}
```

You can now test the controller's actions return as you expect:

```csharp
[Fact]
public void When_requesting_all_addresses()
{
  var mediator = Substitute.For<IMediator>();
  var controller = new AddressController(mediator);
  controller.User = Substitute.For<IPrincipal>();

  var result = controller.Get();

  mediator
      .Received(1)
      .Send(Arg.Is<GetAllAddressesQuery>(q => q.User == controller.User));
}
```

This is also useful when doing integration tests, as you can use `Microsoft.Owin.Testing.TestApp` to test that all the serialization, content negotiation etc works correctly, and still use a substituted mediator so you have known values to test with:

```csharp

[Fact]
public async void Addresses_get_should_return_an_empty_json_array()
{
    var mediator = Substitute.For<IMediator>();
    mediator.Send(Arg.Any<GetAllAddressesQuery>()).Returns(Enumerable.Empty<Address>());

    var server = TestServer.Create(app =>
    {
        var api = new Startup(mediator);
        api.Configuration(app);
    });

    var response = await _server
        .CreateRequest("/api/address")
        .AddHeader("content-type", "application/json")
        .GetAsync();

    var json = await response.Content.ReadAsStringAsync();

    json.ShouldBe("[]");
}
```

### For Handlers

Handler are now isolated from the front end of your application, which means testing is a simple matter of creating an instance, passing in a message, and checking the result.  For example the `GetAllAddressesQuery` handler could be implemented like so:

```csharp
public class GetAllAddressesQueryHandler : IRequestHandler<GetAllAddressesQuery, IEnumerable<Address>>
{
    public IEnumerable<Address> Handle(GetAllAddressesQuery message)
    {
        if (message.User == null)
            return Enumerable.Empty<Address>();

        return [] {
            new Address { Line1 = "34 Home Road", PostCode = "BY2 9AX" }
        };
    }
}
```

And a test might look like this:

```csharp

[Fact]
public void When_no_user_is_specified()
{
    var handler = new GetAllAddressesQueryHandler();
    var result = handler.Handle(new GetAllAddressesQuery());

    result.ShouldBeEmpty();
}
```

## Multiple Front Ends

The next advantage of using Commmands and Queries is that you can support multiple frontends without code duplication.  This ties in very nicely with a [Hexagonal architecture][hexagonal-architecture]. For example, one of my current projects has a set of commands and queries, which are used by a WebApi, and WebSocket connector, and a RabbitMQ adaptor.

This sample also makes use of [RabbitHarness][rabbit-harness], which provides a small interface for easy sending, listening and querying of queues and exchanges.

```csharp
public RabbitMqConnector(IMediator mediator, IRabbitConnector connector) {
    _mediator = mediator;
    _connector = connector;

    _connector.ListenTo(new QueueDefinition { Name = "AddressQueries" }, OnMessage);
}

private bool OnMessage(IBasicProperties props, GetAllAddressesQuery message)
{
    //in this case, the message sent to RabbitMQ matches the query structure
    var addresses = _mediator.Send(message);

    _connector.SendTo(
        new QueueDefinition { Name = props.ReplyTo },
        replyProps => replyProps.CorrelationID = props.CorrelationID,
        addresses
    );
}
```

## Vertical Slicing

This a soft-advantage of Commands and Queries I have found - you can have many more developers working in parallel on a project adding commands and queries etc, before you start treading on each others toes...and the only painful part is all the `*.csproj` merges you need to do!  Your mileage may vary on this one!

## Disadvantages

In a large project, you can end up with a lot of extra classes, which can be daunting at first - one of my current projects has around 60 `IRequest` and `IRequestHandler` implementations.  As long as you follow a good naming convention, or sort them in to namespaces, it is not that much of a problem.

## Overall

Overall I like this pattern a lot - especially as it makes transitioning towards EventSourcing and/or full CQRS much easier.

How about you? What are your thoughts and experiences on this?


[derek-comartin-thin-controllers]: http://codeopinion.com/thin-controllers-cqrs-mediatr/
[hexagonal-architecture]: http://alistair.cockburn.us/Hexagonal+architecture
[rabbit-harness]: https://www.nuget.org/packages/rabbitharness
