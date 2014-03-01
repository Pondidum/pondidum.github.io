---
layout: post
title: To mutate or not to mutate
Tags: design, code, net
permalink: to-mutate-or-not-to-mutate
---

I have been working on a project recently that involves a lot of work with Flags Enums.  To aid with this I created a set of Extension Methods:

	Add(Of T as Structure)(self as T, value as Int) as T
	Add(Of T as Structure)(self as T, values() as Int) as T
	Remove(Of T as Structure)(self as T, value as Int) as T
	Remove(Of T as Structure)(self as T, values() as Int) as T

	Has(Of T as Structure)(self as T, value as Int) as Boolean
	HasAll(Of T as Structure)(self as T, values() as Int) as Boolean
	HasAny(Of T as Structure)(self as T, values() as Int) as Boolean

Now the last 3 methods I am happy with - they are self explanatory and do whatâ€™s expected.  The first four however I am less convinced by.  

My main problem is how I wrote some code:

	Dim state = States.Blank
	
	If someCondition Then state.Add(States.Disabled)
	If someOtherCondition Then state.Add(States.Disconnected)
	
	return state

Which to my surprise always returned `States.Blank` rather than `Disabled` or `Disconnected` or a combination of the two.  After a lot of close looking, I realised it was because the `Add` method was a function and I was not using the return value.

The logical thing seemed to be changing the extension methods to use a reference parameter rather than a value parameter.  While this worked in my vb.net library, the second I tried to use it in my C# test project (MSpec), it broke with the following error:

	Error	Argument 1 must be passed with the 'ref' keyword
	
So it cannot work like this, I have to return the result as a new instance of the enum.  I don't like it, but other Structure based code (such as DateTime, String) work like this too.

On the point of mutability, I think a system like Ruby's of indicating a destructive method would be good:

	stringValue.chomp!		//This will modify stringValue
	stringValue.chomp		//This will return a new instance which has been chomped

But for now I will settle for returning a new instance.
