---
layout: post
title: Working with XmlTextWriter
tags: code net

---

I was working on some code today that needs a lot of data writing into an XML document.  The documents structure is not repetitive - it is loads of one time data, so templating the document is possible, but not the best route to go.

To that end, it uses an `XmlTextWriter`.  The problem I have with it is the way you must write sub-elements.  If you just need a single value wrapped in a tag, you are catered for already:

{% highlight c# %}
writer.WriteElementString("name", current.Name);
{% endhighlight %}

However, if you want to embed a composite set of elements, you are left with this lovely chunk:

{% highlight c# %}
writer.WriteStartElement("composite")

writer.WriteElementString("firstName", current.FirstName);
writer.WriteElementString("lastName", current.LastName);

writer.WriteEndElement();
{% endhighlight %}

And if you have a long document, with many composite elements, good luck remembering which element is being ended by `WriteEndElement()` (even if you functionalise it, you still run into the issue.)

The solution I came up with for this was a class and an extension method:

{% highlight c# %}
internal class WriteElement : IDisposable
{
	private XmlTextWriter _writer;

	internal WriteElement(XmlTextWriter writer, String element)
	{
		_writer = writer;
		_writer.WriteStartElement(element);
	}

	public void Dispose()
	{
		 _writer.WriteEndElement();
		 _writer = null;
	}
}

static class XmlTextWriterExtensions
{
	public static IDisposable WriteComposite(this XmlTextWriter self, String element)
	{
		return new WriteElement(self, element);
	}
}
{% endhighlight %}

This enables me to write composite elements like this:

{% highlight c# %}
using (writer.WriteComposite("composite"))
{
	writer.WriteElementString("firstName", current.FirstName);
	writer.WriteElementString("lastName", current.LastName);
}
{% endhighlight %}

With the two benefits of knowing when my composite elements are ending, and I also gain indentation of my elements, which allows me to *see* where the composites are a lot easier.
