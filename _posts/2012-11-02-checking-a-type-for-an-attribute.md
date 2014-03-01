---
layout: post
title: Checking a Type for an Attribute
Tags: code
permalink: checking-a-type-for-an-attribute
---

I needed to be able to detect at run time if an Enum has a specific Attribute on it.  Generalizing it, I came up with this:

Calling:

	var hasFlags = typeof(EnumWithFlags).HasAttribute<FlagsAttribute>();

Implementation:

	public static Boolean HasAttribute<T>(this Type self) where T : Attribute
	{
		if (self == null) 
		{
			throw new ArgumentNullException("self");
		}

		return self.GetCustomAttributes(typeof(T), false).Any();

	}

It may only be two lines, but it is very useful none the less.