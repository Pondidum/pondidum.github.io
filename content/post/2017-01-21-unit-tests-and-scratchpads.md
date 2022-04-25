---
date: "2017-01-21T00:00:00Z"
tags: c# testing xunit
title: Unit Tests & Scratchpads
---

Often when developing something, I have the need to check how a function or library works.  For example, I *always* have to check for this question:

> Does `Directory.ListFiles(".\\temp\\")` return a list of filenames, a list of relative filepaths, or a list of rooted filepaths?

It returns relative filepaths by the way:

```
Directory.ListFiles(".\\temp\\");
[ ".\temp\NuCrunch.Tests.csproj", ".\temp\packages.config", ".\temp\Scratchpad.cs" ]
```

Now that there is a C# Interactive window in Visual Studio, you can use that to test the output.  Sometimes however the C# Interactive window is not suitable:

* You want to test needs a little more setup than a couple of lines
* You wish to use the debugger to check on intermediate state
* You are not in Visual Studio (I am 99% of the time in [Rider](https://www.jetbrains.com/rider/))

When this happens, I turn to the unit test file which I add to all unit test projects:  the `Scratchpad.cs`.

The complete listing of the file is this:

```csharp
using Xunit;
using Xunit.Abstractions;

namespace NuCrunch.Tests
{
	public class Scratchpad
	{
		private readonly ITestOutputHelper _output;

		public Scratchpad(ITestOutputHelper output)
		{
			_output = output;
		}

		[Fact]
		public void When_testing_something()
		{

		}
	}
}
```

It gets committed to the git repository with no content in the `When_testing_something` method, and is never committed again afterwards.  The `_output` field is added to allow writing to console/test window easily too.

Now whenever I wish to experiment with something, I can pop open the `Scratchpad` write some test content, then execute and debug it to my hearts content.

After I am done with the test code, one of two things happen:  it gets deleted, or it gets moved into a proper unit test.
