---
date: "2015-09-21T00:00:00Z"
tags: ["c#", "webapi"]
title: Posting PlainText to Asp WebApi
---

Recently I have been writing a WebApi project which needs to accept plaintext via the body of a PUT request, and did the logical thing of using the `FromBodyAttribute`

```csharp
public HttpStatusCode PutKv([FromBody]string content, string keyGreedy)
{
  return HttpStatusCode.OK;
}
```

Which didn't work, with the useful error message of "Unsupported media type."

It turns out that to bind a value type with the `FromBody` attribute, you have to prefix the body of your request with an `=`.  As I am emulating another Api's interface, this is not an option, so I set about figuring out how to override this requirement.

In the end I discovered that providing a new `MediaTypeFormatter` which handles plaintext is the answer:

```csharp
public class PlainTextMediaTypeFormatter : MediaTypeFormatter
{
  public PlainTextMediaTypeFormatter()
  {
    SupportedMediaTypes.Add(new MediaTypeHeaderValue("text/plain"));
  }

  public override Task<object> ReadFromStreamAsync(Type type, Stream readStream, HttpContent content, IFormatterLogger formatterLogger)
  {
    var source = new TaskCompletionSource<object>();

    try
    {
      using (var memoryStream = new MemoryStream())
      {
        readStream.CopyTo(memoryStream);
        var text = Encoding.UTF8.GetString(memoryStream.ToArray());
        source.SetResult(text);
      }
    }
    catch (Exception e)
    {
      source.SetException(e);
    }

    return source.Task;
  }

  public override Task WriteToStreamAsync(Type type, object value, Stream writeStream, HttpContent content, System.Net.TransportContext transportContext, System.Threading.CancellationToken cancellationToken)
  {
    var bytes = Encoding.UTF8.GetBytes(value.ToString());
    return writeStream.WriteAsync(bytes, 0, bytes.Length, cancellationToken);
  }

  public override bool CanReadType(Type type)
  {
    return type == typeof(string);
  }

  public override bool CanWriteType(Type type)
  {
    return type == typeof(string);
  }
}
```

This can then be added to the `config.Formatters` collection:

```csharp
public static class WebApiConfig
{
  public static void Register(HttpConfiguration http)
  {
    http.Formatters.Add(new PlainTextMediaTypeFormatter());
  }
}
```

It really seems like something which should be supplied out of the box with WebApi to me, but at least it wasn't as complicated to implement as I was expecting it to be :)
