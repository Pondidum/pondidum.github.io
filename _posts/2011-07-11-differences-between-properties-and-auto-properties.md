---
layout: post
title: Differences between Properties and Auto Properties
tags: code c#

---

While writing some of the specs for [ViewWeaver][1], I noticed that one was failing:

>When passed a type with one write only property<br />
>  - it should return no mappings

When I stepped through the code, it was indeed not filtering out the write only property.
This is the code used to find all readable properties:

```csharp
var allProperties = typeof(T).GetProperties(BindingFlags.Instance | BindingFlags.Public);
var readableProperties = allProperties.Where(p => p.CanRead && !p.GetIndexParameters().Any());
```

For some reason `CanRead` was returning true, then I noticed how I had defined my class under test:

```csharp
public class OnePublicWriteonlyProperty
{
	public String Test { private get; set; }
}
```

So it turns out that even though I had filtered to all Public Properties, a private Getter (or Setter) still passes through.

```csharp
var allProperties = typeof(T).GetProperties(BindingFlags.Instance | BindingFlags.Public);
var readableProperties = allProperties.Where(p => p.CanRead &&
											 p.GetGetMethod() != null &&
											 p.GetGetMethod().IsPublic &&
											 !p.GetIndexParameters().Any());
```

Changing the expression to check for the GetMethod existing, and being public fixed this, and seems obvious in retrospect, but it is worth remembering that an Auto Property is ever so slightly different from a plain property with only a Get or Set method defined.

[1]: https://github.com/Pondidum/ViewWeaver