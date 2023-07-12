+++
date = '2016-03-18T00:00:00Z'
tags = ['c#', 'rabbitmq', 'xunit']
title = 'RabbitMQ integration tests in XUnit'

+++

Quite a number of my projects involve talking to [RabbitMQ][rabbitmq], and to help check things work as expected, I often have a number of integration tests which talk to a local RabbitMQ instance.

While this is fine for tests being run locally, it does cause problems with the build servers - we don't want to install RabbitMQ on there, and we don't typically want the build to be dependent on RabbitMQ.

To solve this I created a replacement `FactAttribute` which can check if RabbitMQ is available, and skip tests if it is not.

This attribute works with a single host, and will only check for the host actually being there on its first connection.

```csharp
public class RequiresRabbitFactAttribute : FactAttribute
{
  private static bool? _isAvailable;

  public RequiresRabbitFactAttribute(string host)
  {
    if (_isAvailable.HasValue == false)
      _isAvailable = CheckHost(host);

    if (_isAvailable == false)
      Skip = $"RabbitMQ is not available on {host}.";
  }

  private static bool CheckHost(string host)
  {
    var factory = new ConnectionFactory
    {
      HostName = host,
      RequestedConnectionTimeout = 1000;
    };

    try
    {
      using (var connection = factory.CreateConnection())
      {
        return connection.IsOpen;
      }
    }
    catch (Exception)
    {
      return false;
    }
  }

}
```

I was planning on using a dictionary, keyed by host to store the availability, but realized that I always use the same host throughout a test suite.

The reason for passing the host name in via the ctor rather than using a constant is that this usually resides within a generic "rabbitmq helpers" type assembly, and is used in multiple projects.

[rabbitmq]: https://rabbitmq.com
