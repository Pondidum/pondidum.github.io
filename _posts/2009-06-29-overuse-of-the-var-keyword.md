---
layout: post
title: Overuse of the Var keyword
tags: design code net

---

When I first got hold of VS2008, and had a play with the new version of C# I loved the Var keyword.  To me the most amazing thing was no more declarations like this:

```csharp
    System.Text.RegularExpressions.Regex rx = new System.Text.RegularExpressions.Regex();
```

Instead I could write the following:

```csharp
    var rx = new System.Text.RegularExpressions.Regex();
```

Making it akin to VB developers being able to write:

```vb
    Dim rx As New System.Text.RegularExpressions.Regex()
```

(I have had however to cope with a coding standard that explicitly forbid this declaration in VB...Backwards or what?)

My only gripe with the var keyword is that it is being overused. Horribly.  Every day I come across people (mainly on StackOverflow, but development blogs, people I know do this too) writing code something like this:

```csharp
    var fileName = "C:\\text.xml";
    var itemCount = 1;
    var xml = new System.Xml.XmlDocument();

    for (var i = 0; i < 10; ++i) {/*...*/}
```

In that code snippet there is *one* place where var is used well.  Don't declare strings as var, it's a string. Don't declare int as var, not only is it not necessary, it hasn't saved you any typing, they are both 3 characters long.

The other point (one I seem to keep coming back to) is code readability:

```csharp
    var result = something.FunctionThatReturnsSomething();
```

Now, what is the type of result?  Admittedly, this could be improved by [naming your functions properly][1] and naming variables a little less generically, like so:

```csharp
    var polar = something.PolarCoordinates();
```

[1]: /coming-from-something-as-opposed-to-going-to-something
