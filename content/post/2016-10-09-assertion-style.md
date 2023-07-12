+++
date = '2016-10-09T00:00:00Z'
tags = ['c#', 'nunit', 'testing', 'shouldly', 'assert']
title = 'Shouldly: Why would you assert any other way?'

+++

I like to make my development life as easy as possible - and removing small irritations is a great way of doing this.  Having used [Shouldly](http://docs.shouldly-lib.net/v2.4.0/docs) in anger for a long time, I have to say I feel a little hamstrung when going back to just using NUnit's assertions.

I have been known on a couple of projects which use only NUnit assertions, when trying to solve a test failure with array differences, to install Shouldly, fix the test, then remove Shouldly again!

The rest of this post goes through the different assertion models, and how they differ from each other and, eventually, why everyone should be using Shouldly!


## The Most Basic

```csharp
var valueOne = "Something";
var valueTwo = "Something else";

Debug.Assert(valueOne == valueTwo);
Debug.Assert(valueOne == valueTwo, $"{valueOne} should have been {valueTwo}");
```

This is an assertion at it's most basic.  It will only assert if the condition is false, and optionally you can specify a 2nd parameter with a message.

This has a couple of good points to it. No external dependencies are required, and it is strong typed (as your condition has to compile.)  The down sides to this are that it is not very descriptive, and can only be used in Debug compiles (or with the DEBUG constant defined), meaning a Release mode build cannot be tested with this.

This also suffers from the descriptiveness problem - an output from this will only have a message saying an assertion failed, rather than anything helpful in figuring out why an assertion failed.

## NUnit's First Attempt
```csharp
var valueOne = "Something";
var valueTwo = "Something else";

Assert.AreEqual(valueOne, valueTwo);
Assert.AreEqual(valueOne, valueTwo, $"{valueOne} should have been {valueTwo}");
```
This improves on the Most Basic version by working in Release mode builds, and as it only depends on the test framework, it doesn't add a dependency you didn't already have.

There are two things I dislike about this method: it remains as undescriptive as the first method, and it adds the problem of parameter ambiguity:  Which of the two parameters is the expected value, and which is the value under test? You can't tell without checking the method declaration.  While this is a small issue, it can cause headaches when you are trying to debug a test which has started failing, only to discover the assertion being the wrong way around was leading you astray!


## NUnit's Second Attempt

```csharp
var valueOne = "Something";
var valueTwo = "Something else";

Assert.That(valueOne, Is.EqualTo(valueTwo));
Assert.That(valueOne, Is.EqualTo(valueTwo), $"{valueOne} should have been {valueTwo}");
```

This is an interesting attempt at readability.  On the one hand, it's very easy to read as a sentence, but it is very wordy, especially if you are wanting to do a Not equals `Is.Not.EqualTo(valueTwo)`.

This biggest problem with this however, is the complete loss of strong typing - both arguments are `object`.  This can trip you up when testing things such as Guids - especially if one of the values gets `.ToString()` on it at some point:

```csharp
var id = Guid.NewGuid();
Assert.That(id.ToString(), Is.EqualTo(id));
```

Not only will this compile, but when the test fails, unless you are paying close attention to the output, it will look like it should've passed, as the only difference is the `"` on either side of one of the values.


## Shouldly's Version

```csharp
var valueOne = "Something";
var valueTwo = "Something else";

valueOne.ShouldBe(valueTwo);
valueOne.ShouldBe(valueTwo, () => "Custom Message");
```

Finally we hit upon the [Shouldly](http://docs.shouldly-lib.net/v2.4.0/docs) library.  This assertion library not only solves the code-time issues of strong typing, parameter clarity, and wordiness, it really improves the descriptiveness problem.

Shouldly uses the expression being tested against to create meaningful error messages:

```csharp
//nunit
Assert.That(map.IndexOfValue("boo"), Is.EqualTo(2));    // -> Expected 2 but was 1

//shouldly
map.IndexOfValue("boo").ShouldBe(2);                    // -> map.IndexOfValue("boo") should be 2 but was 1
```

This is even more pronounced when you are comparing collections:

```csharp
new[] { 1, 2, 3 }.ShouldBe(new[] { 1, 2, 4 });
```

Produces the following output
```
should be
    [1, 2, 4]
but was
    [1, 2, 3]
difference
    [1, 2, *3*]
```

And when comparing strings, not only does it tell you they were different, but provides a visualisation of what was different:

```
input
    should be
"this is a longer test sentence"
    but was
"this is a long test sentence"
    difference
Difference     |                                |    |    |    |    |    |    |    |    |    |    |    |    |    |    |    |
               |                               \|/  \|/  \|/  \|/  \|/  \|/  \|/  \|/  \|/  \|/  \|/  \|/  \|/  \|/  \|/  \|/
Index          | ...  9    10   11   12   13   14   15   16   17   18   19   20   21   22   23   24   25   26   27   28   29
Expected Value | ...  \s   l    o    n    g    e    r    \s   t    e    s    t    \s   s    e    n    t    e    n    c    e
Actual Value   | ...  \s   l    o    n    g    \s   t    e    s    t    \s   s    e    n    t    e    n    c    e
```

## Finishing

So having seen the design time experience and rich output Shouldly gives you, why would you not use it?
