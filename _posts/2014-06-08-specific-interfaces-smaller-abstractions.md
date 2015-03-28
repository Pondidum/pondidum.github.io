---
layout: post
title: Specific Interfaces
tags: design code net
permalink: specific-interfaces-smaller-abstractions
---

While writing my [CruiseCli][github-cruisecli] project, I needed to do some data storage, and so used my standard method of filesystem access, the `IFileSystem`.  This is an interface and implementation which I tend to copy from project to project, and use as is.  The interface looks like the following:

{% highlight c# %}
public interface IFileSystem
{
	bool FileExists(string path);
	void WriteFile(string path, Stream contents);
	void AppendFile(string path, Stream contents);
	Stream ReadFile(string path);
	void DeleteFile(string path);

	bool DirectoryExists(string path);
	void CreateDirectory(string path);
	IEnumerable<string> ListDirectory(string path);
	void DeleteDirectory(string path);
}
{% endhighlight %}

And the standard implementation looks like the following:

{% highlight c# %}
public class FileSystem : IFileSystem
{
	public bool FileExists(string path)
	{
		return File.Exists(path);
	}

	public void WriteFile(string path, Stream contents)
	{
		using (var fs = new FileStream(path, FileMode.Create, FileAccess.Write))
		{
			contents.CopyTo(fs);
		}
	}

	public Stream ReadFile(string path)
	{
		return new FileStream(path, FileMode.Open, FileAccess.Read, FileShare.Read);
	}
	//snip...
}
{% endhighlight %}

This (I think) is a very good solution to file system access as I can easily mock the interface and add expectations and stub values to it for testing.

However, on the CruiseCli project, I realised I didn't need most of what the interface provided, so I chopped all the bits off I didn't want, and added a property for a base directory I was using all the time:

{% highlight c# %}
public interface IFileSystem
{
	string HomePath { get; }

	void WriteFile(string path, Stream contents);
	Stream ReadFile(string path);
	bool FileExists(string path);
}
{% endhighlight %}

Which was better than the original, as I have a lot less methods to worry about, and thus it is more specific to my use case.

But I got thinking later in the project; "what are my use cases?", "what do I actually want to do with the filesystem?"  The answer to this was simple: Read a config file, and write to the same config file.  Nothing else.

So why not make the interface even more specific in this case:

{% highlight c# %}
public interface IConfiguration
{
	void Write(Stream contents);
	Stream Read();
}
{% endhighlight %}

Even simpler, and I now have the benefit of not caring what the filepaths are outside of the implementing class.

This means that in my integration tests, I can write an in-memory `IConfiguration` with far less hassle, and not need to worry about fun things like character encoding and case sensitivity on filepaths!

In a more complicated system, I would probably keep this new `IConfiguration` interface for accesing the config file, and make the concrete version depend on the more general `IFileSystem`:

{% highlight c# %}
public class Configuration : IConfiguration
{
	private const string FileName = ".cruiseconfig";
	private readonly IFileSystem _fileSystem;

	public Configuration(IFileSystem fileSystem)
	{
		_fileSystem = fileSystem;
	}

	public void Write(Stream contents)
	{
		_fileSystem.WriteFile(Path.Combine(_fileSystem.Home, FileName), contents);
	}

	public Stream Read()
	{
		return _fileSystem.ReadFile(Path.Combine(_fileSystem.Home, FileName));
	}
}
{% endhighlight %}

For a small system this would probably be overkill, but for a much larger project, this could help provide a better seperation of responsibilities.

[github-cruisecli]: https://github.com/Pondidum/CruiseCli
