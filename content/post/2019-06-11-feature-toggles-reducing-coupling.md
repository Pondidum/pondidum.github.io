---
date: "2019-06-11T00:00:00Z"
tags: ["featuretoggles", "c#", "di", "microservices"]
title: 'Feature Toggles: Reducing Coupling'
---

One of the points I make in my [Feature Toggles talk](https://www.youtube.com/watch?v=r7VI5x2XKXw) is that you shouldn't be querying a toggle's status all over your codebase.  Ideally, each toggle gets checked in as few places as possible - preferably only one place.  The advantage of doing this is that very little of your codebase needs to be coupled to the toggles (either the toggle itself or the library/system for managing toggles itself).

This post will go over several situations when that seems hard to do, namely: multiple services, multiple distinct areas of a codebase, and multiple times in a complex class or method.  As in the [previous post](/2019/06/03/feature-toggles-branch-by-abstraction/) on this, we will be using [Branch By Abstraction](https://www.martinfowler.com/bliki/BranchByAbstraction.html) to do most of the heavy lifting.


## Multiple Services

Multiple services interacting with the same feature toggle is a problematic situation to deal with, especially if multiple teams own the different services.

One of the main issues with this is trying to coordinate the two (or more) services.  For example, if one team needs to switch off their implementation due to a problem, should the other services also get turned off too?  To compound on this problem, what happens if one system can react to the toggle change faster than the other?

Services changing configuration at different speeds can also cause issues with handling in-flight requests too: if the message format is different when the toggle is on, will the receiving system be able to process a message produced when the toggle was in one state but consumed in the other state?

We can solve some of this by using separate toggles for each service (and they are not allowed to query the other service's toggle state), and by writing the services so that they can handle both old format and new format requests at the same time.

For example, if we had a sending system which when the toggle is off will send this DTO:

```csharp
public class PurchaseOptions
{
    public Address Address { get; set; }
}
```

And when the toggle is enabled, it will send the following DTO instead:

```csharp
public class PurchaseOptions
{
    public BillingAddress Address { get; set; }
    public DeliveryAddress Address { get; set; }
}
```

To make the receiving system handle this, we deserialize the request into a DTO which contains all possible versions of the address, and then use the best version based on our own toggle state:

```csharp
public class PurchaseOptionsRequest
{
    public Address Address { get; set; }
    public BillingAddress Address { get; set; }
    public DeliveryAddress Address { get; set; }
}

public class PurchaseController
{
    public async Task<PurchaseOptionsResponse> Post(PurchaseOptionsRequest request)
    {
        if (separateAddresses.Enabled)
        {
            var deliveryAddress = request.DeliveryAddress ?? request.Address;
            var billingAddress = request.BillingAddress ?? request.Address;

            ConfigureDelivery(deliveryAddress);
            CreateInvoice(billingAddress, deliveryAddress);
        }
        else
        {
            var address = request.Address ?? request.DeliveryAddress ?? request.BillingAddress;

            ConfigureDelivery(address)
            CreateInvoice(address, address);
        }
    }
}
```

Note how both sides of the toggle check read all three possible address fields, but try to use different fields first.  This means that no matter whether the sending service has it's toggle on or not, we will use the correct address.


## Multiple Areas of the Codebase

To continue using the address example, we might have a UI, Controller and Handler, which all need to act differently based on the same toggle:

* The UI needs to display either one or two address editors
* The controller needs to have different validation logic for multiple addresses
* The Command Handler will need to dispatch different values

We can solve this all by utilising [Branch By Abstraction](https://www.martinfowler.com/bliki/BranchByAbstraction.html) and Dependency Injection to make most of the codebase unaware that a feature toggle exists.  Even the implementations won't need to know about the toggles.

```csharp
public class Startup
{
    public void ConfigureContainer(ServiceRegistry services)
    {
        if (separateAddresses.Enabled) {
            services.Add<IAddressEditor, MultiAddressEditor>();
            services.Add<IRequestValidator, MultiAddressValidator>();
            services.Add<IDeliveryHandler, MultiAddressDeliveryHandler>();
        }
        else {
            services.Add<IAddressEditor, SingleAddressEditor>();
            services.Add<IRequestValidator, SingleAddressValidator>();
            services.Add<IDeliveryHandler, SingleAddressDeliveryHandler>();
        }
    }
}
```

Let's look at how one of these might work.  The `IRequestValidator` has a definition like so:

```csharp
public interface IRequestValidator<TRequest>
{
    public IEnumerable<string> Validate(TRequest request);
}
```

There is a middleware in the API request pipeline which will pick the right validator out of the container, based on the request type being processed.  We implement two validators, once for the single address, and one for multiaddress:

```csharp
public class SingleAddressValidator : IRequestValidator<SingleAddressRequest>
{
    public IEnumerable<string> Validate(SingleAddressRequest request)
    {
        //complex validation logic..
        if (request.Address == null)
            yield return "No Address specified";

        if (PostCode.Validate(request.Address.PostCode) == false)
            yield return "Invalid Postcode";
    }
}

public class MultiAddressValidator : IRequestValidator<MultiAddressRequest>
{
    public IEnumerable<string> Validate(MultiAddressRequest request)
    {
        var billingMessages = ValidateAddress(request.BillingAddress);

        if (billingMessages.Any())
            return billingMessages;

        if (request.DifferentDeliveryAddress)
            return ValidateAddress(request.DeliveryAddress);
    }
}
```

The implementations themselves don't need to know about the state of the toggle, as the container and middleware take care of picking the right implementation to use.

## Multiple Places in a Class/Method

If you have a single method (or class) which needs to check the toggle state in multiple places, you can also use the same Branch by Abstraction technique as above, by creating a custom interface and pair of implementations, which contain all the functionality which changes.

For example, if we have a method for finding an offer for a customer's basket, which has a few separate checks that the toggle is enabled in it:

```csharp
public SuggestedBasket CreateOffer(CreateOfferCommand command)
{
    if (newFeature.Enabled) {
        ExtraPreValidation(command).Throw();
    } else {
        StandardPreValidation(command).Throw();
    }

    var offer = SelectBestOffer(command.Items);

    if (offer == null && newFeature.Enabled) {
        offer = FindAlternativeOffer(command.Customer, command.Items);
    }

    return SuggestedBasket
        .From(command)
        .With(offer);
}
```

We can extract an interface for this, and replace the toggle specific parts with calls to the interface instead:

```csharp
public interface ICreateOfferStrategy
{
    IThrowable PreValidate(CreateOfferCommand command);
    Offer AlternativeOffer(CreateOfferCommand command, Offer existingOffer);
}

public class DefaultOfferStrategy : ICreateOfferStrategy
{
    public IThrowable PreValidate(CreateOfferCommand command)
    {
        return StandardPreValidation(command);
    }

    public Offer AlternativeOffer(CreateOfferCommand command, Offer existingOffer)
    {
        return existingOffer;
    }
}

public class DefaultOfferStrategy : ICreateOfferStrategy
{
    public IThrowable PreValidate(CreateOfferCommand command)
    {
        return ExtraPreValidation(command);
    }

    public Offer AlternativeOffer(CreateOfferCommand command, Offer existingOffer)
    {
        if (existingOffer != null)
            return existingOffer;

        return TryFindAlternativeOffer(command.Customer, command.Items, offer);
    }
}

public class OfferBuilder
{
    private readonly ICreateOfferStrategy _strategy;

    public OfferBuilder(ICreateOfferStrategy strategy)
    {
        _strategy = strategy;
    }

    public SuggestedBasket CreateOffer(CreateOfferCommand command)
    {
        _strategy.PreValidation(command).Throw();

        var offer = SelectBestOffer(command.Items);

        offer = _strategy.AlternativeOffer(command, offer);

        return SuggestedBasket
            .From(command)
            .With(offer);
    }
}
```

Now that we have done this, our `CreateOffer` method has shrunk dramatically and no longer needs to know about the toggle state, as like the rest of our DI examples, the toggle can be queried once in the startup of the service and the correct `ICreateOfferStrategy` implementation registered into the container.

## End

Hopefully, this post will give a few insights into different ways of reducing the number of calls to your feature toggling library, and prevent you scattering lots of if statements around the codebase!
