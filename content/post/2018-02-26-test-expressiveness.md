---
date: "2018-02-26T00:00:00Z"
tags: c# testing
title: Test Expressiveness
---

We have a test suite at work which tests a retry decorator class works as expected.  One of the tests checks that when the inner implementation throws an exception, it will log the number of times it has failed:

```csharp
[Test]
public async Task ShouldLogRetries()
{
    var mockClient = Substitute.For<IContractProvider>();
    var logger = Subsitute.For<ILogger>();
    var sut = new RetryDecorator(mockClient, logger, maxRetries: 3);

    mockClient
        .GetContractPdf(Arg.Any<string>())
        .Throws(new ContractDownloadException());

    try
    {
        await sut.GetContractPdf("foo");
    }
    catch (Exception e){}

    logger.Received(1).Information(Arg.Any<string>(), 1);
    logger.Received(1).Information(Arg.Any<string>(), 2);
    logger.Received(1).Information(Arg.Any<string>(), 3);
}
```

But looking at this test, I couldn't easily work out what the behaviour of `sut.GetContractPdf("foo")` was supposed to be; should it throw an exception, or should it not?  That fact that there is a `try...catch` indicates that it *might* throw an exception, but doesn't give any indication that it's required or not.

```csharp
try
{
    await sut.GetContractPdf("foo");
}
catch (Exception e)
{
}
```

Since we have the [`Shouldly`](https://www.nuget.org/packages/Shouldly/) library in use, I changed the test to be a little more descriptive:

```csharp
Should.Throw<ContractDownloadException>(() =>
    sut.GetContractPdfForAccount("foo")
);
```

Now we know that when the decorator exceeds the number of retries, it should throw the inner implementation's exception.

This in itself is better, but it also raises another question:  Is the test name correct? Or should this now be two separate tests? One called `ShouldLogRetries`, and one called `ShouldThrowInnerExceptionOnRetriesExceeded`?

Even though I ended up adding the second test, I still left the first test with the `Should.Throw(...)` block, as it is still more descriptive at a glance than the `try...catch`.
