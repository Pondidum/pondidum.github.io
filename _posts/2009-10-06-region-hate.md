---
layout: post
title: Region Hate
tags: design code net

---

There seems to be a [lot of][5] [negativity][6] [towards][7] the `#Region` in .net at the moment, with many people hating them and calling all usages of them ['retarded'][1].

I can see their point, especially when you see the odd class with regions like this:

{% highlight c# %}
Class Foo
{
	#Private Members
	#Protected Members
	#Friend Members
	#Public Members
	#Private Constructors
	#Protected Constructors
	#Friend Constructors
	#Public Constructors
	#Private Methods
	#Protected Methods
	#Friend Methods
	#Public Methods
}
{% endhighlight %}

Clearly the person who wrote this was ill at the time (I hope...), and besides, where would `Protected Friends` go? Hmm?

I however find regions useful, especially when writing objects (see what I did there?).  Now while an object might have might be [DRY][3] and only have a [Single Responsibility][2], it might also have many properties.  What I tend to do with regions is hide my getters and setters:

{% highlight c# %}
Class Bar
{
	Member1
	...
	Member2

	#Region Properties
		//....
	#End Region

	Method1(){/* */}
	...
	Method1(){/* */}
}
{% endhighlight %}

This way I am hiding standard boiler plate code, and everything that actually matters is visible.  If you don't like hiding properties that have a lot of code in them, then your problem may be the fact that you have lots of code in the properties.  Something like [PostSharp][4] could allow you to inject all your properties with the common code such as `PropertyChanging(sender, e)`, `PropertyChanged(sender, e)`.

If you need lots of specific code in a property, then it is surely under unit test?  If it isn't, why not? And if it is, does it matter that you can't see the property without clicking the little + sign?

One other slight point: with my method of `#region` usage, if you don't like regions, you have one click to expand it (or if you don't like clicking, `Ctrl+M, Ctrl+M` in VS will expand/collapse whatever is at the cursor position), so it really is not that difficult to cope with.

Like all technologies, use it when it makes sense.  No Regions can be just as bad as many Regions.


[1]: http://extractmethod.wordpress.com/2008/02/29/just-say-no-to-c-regions/
[2]: http://en.wikipedia.org/wiki/Single_responsibility_principle
[3]: http://en.wikipedia.org/wiki/DRY
[4]: http://www.postsharp.org/
[5]: http://stackoverflow.com/questions/755465/do-you-say-no-to-c-regions
[6]: http://stackoverflow.com/questions/1027504/using-regions-in-c-is-considered-bad-practice
[7]: http://stackoverflow.com/questions/1524248/use-of-region-in-c-closed
