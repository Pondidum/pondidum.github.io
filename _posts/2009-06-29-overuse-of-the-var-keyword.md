---
layout: post
title: Overuse of the Var keyword
Tags: design, code, net
permalink: overuse-of-the-var-keyword
---

When I first got hold of VS2008, and had a play with the new version of C# I loved the Var keyword.  To me the most amazing thing was no more declarations like this:

    System.Text.RegularExpressions.Regex rx = new System.Text.RegularExpressions.Regex();

Instead I could write the following:

    Var rx = new System.Text.RegularExpressions.Regex();

Making it akin to VB developers being able to write:

    Dim rx As New System.Text.RegularExpressions.Regex()

(I have had however to cope with a coding standard that explicitly forbid this declaration in VB...Backwards or what?)

My only gripe with the Var keyword is that it is being overused. Horribly.  Every day I come across people (mainly on StackOverflow, but development blogs, people I know do this too) writing code something like this:

    Var fileName = "C:\\text.xml";
	Var itemCount = 1;
	Var xml = new System.Xml.XmlDocument();

	for (Var i = 0; i < 10; ++i) {/*...*/}
	
In that code snippet there is *one* place where Var is used well.  Don't declare strings as Var, it's a string. Don't declare int as Var, not only is it not necessary, it hasn't saved you any typing, they are both 3 characters long.

The other point (one I seem to keep coming back to) is code readability:

    Var result = something.FunctionThatReturnsSomething();

Now, what is the type of result?  Admittedly, this could be improved by [naming your functions properly][1] and naming variables a little less generically, like so:

    Var polar = something.PolarCoordinates();

This way you can see what the return type is and you can use Var, but I would still rather this:

    Polar polar = something.PolarCoordinates();

The explicitness of this statement is nice as it is concise, and clear. Just having Var result is not so easy, although the IDE helps a lot.

[1]: http://www.stormbase.net/coming-from-something-as-opposed-to-going-to-something
