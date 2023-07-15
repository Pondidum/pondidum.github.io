+++
date = '2010-01-12T00:00:00Z'
tags = ['c#']
title = 'Converting from NUnit to MSTest'

+++

While this is not something I personally would want to do, we (for whatever reason...) are to use MSTest at work (I think it is due to the whole "Its Microsoft, so it's supported" argument).

Now as no one else on the team does any kind of unit testing (serious), the only test projects we have are written by me, on the quiet before being told if I wanted to unit test then use MSTest.  So onto the point of this article.

When you create a project for tests with nunit, you just create a `Class Library`, add a reference to nunit (and Rhino.Mocks of course), build it and run with your preferred method (I like TDD.Net, but that involves paying for at work...so no go there).

When you want to do tests with MSTest, you just create a Test Project and start writing tests. On closer inspection, it's just a `Class Library` with a reference to `Microsoft.VisualStudio.QualityTools.UnitTestFramework`.  So converting one to the other should be easy, right?

Well not quite.  While there is nothing in the GUI to suggest so, you need to modify the csproj/vbproj file to get it to work.  This post on [MSDN][1], had all the details, but in the interest of having things in more than one place (not very DRY I will admit, but there), here are the steps:

1. Remove Reference to Nunit.Core & Nunit.Framework
2. Add Reference to Microsoft.VisualStudio.QualityTools.UnitTestFramework
3. Find and Replace:
  - `using NUnit.Framework;` with `using Microsoft.VisualStudio.TestTools.UnitTesting;` (I actually use a project level import, so I skip this)
  - [TestFixture] -> [TestClass]
  - [Test] -> [TestMethod]
  - [SetUp] -> [TestInitialize]
  - [TearDown] -> [TestCleanup]
  - [TestFixtureSetUp] -> [ClassInitialize]
  - [TestFixtureTearDown] -> [ClassCleanup]
4. Change your Asserts:
  - Assert.Greater(x, y) -> Assert.IsTrue(x > y)
  - Assert.AreEqual(x, Is.EqualTo(y).IgnoreCase) ->  Assert.AreEqual(x, y, True)
5. The 'hidden' part.  In your project file, locate `<PropertyGroup>` (not the one specifying debug|release settings), and add the following to it:
  - <FileAlignment>512</FileAlignment>
  - *.csproj files add:
`<ProjectTypeGuids>{3AC096D0-A1C2-E12C-1390-A8335801FDAB};{FAE04EC0-301F-11D3-BF4B-00C04F79EFBC}</ProjectTypeGuids>`
  - *.vbproj files add:
`<ProjectTypeGuids>{3AC096D0-A1C2-E12C-1390-A8335801FDAB};{F184B08F-C81C-45F6-A57F-5ABD9991F28F}</ProjectTypeGuids>`

This was all I had to do to get our (my) tests running again under MSTest.  Except they didn't run, with the lovely error of:

> The location of the file or directory 'D:\Projects\Dev\SDK\Rhino.Mocks.dll' is not trusted.

That's odd, the file is on my hard disk, its not a network share, so what's the problem?  Right click on Rhino.Mocks.dll and:

![Unblock File][2]

Click the Unblock button, hit Apply, re-run the tests.  All Working now :)

There are a few other points mentioned on the MSDN post too which you may run into:

> If you have relied on NUnit TestFixtureSetup and TestFixtureTearDown methods to do non-static things, will have to move functions in the former to a constructor and the latter to a destructor.  In MSTest, both of these methods must be declared as static.

> If you are relying on AppDomain.CurrentDomain.BaseDirectory to get the root directory, your test will break.  The fix is explained at http://www.ademiller.com/blogs/tech/2008/01/gotchas-mstest-appdomain-changes-in-vs-2008/.

Basically, you need to set your BaseDirectory in your MSTest TestClass constructor like this:

```csharp
string currDir = Environment.CurrentDirectory.Substring(0, Environment.CurrentDirectory.IndexOf("TestResults"));
AppDomain.CurrentDomain.SetData("APPBASE", currDir);
```


> MSTest launches each test method in a separate STA thread instead of the MTA thread you may be expecting.  This probably won't give you any problems.

Hope that helps everyone who has to do this kind of conversion.

[1]: http://social.msdn.microsoft.com/Forums/en/vststest/thread/433e4860-b61f-44fd-bef9-a569fb32d244
[2]: unblock-file.jpg
