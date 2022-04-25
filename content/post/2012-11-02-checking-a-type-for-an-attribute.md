---
date: "2012-11-02T00:00:00Z"
title: Checking a Type for an Attribute
---

I needed to be able to detect at run time if an Enum has a specific Attribute on it.  Generalizing it, I came up with this:

Calling:

```csharp
var hasFlags = typeof(EnumWithFlags).HasAttribute<FlagsAttribute>();
```

Implementation:

```csharp
public static Boolean HasAttribute<T>(this Type self) where T : Attribute
{
	if (self == null)
	{
		throw new ArgumentNullException("self");
	}

	return self.GetCustomAttributes(typeof(T), false).Any();
}
```

It may only be two lines, but it is very useful none the less.
