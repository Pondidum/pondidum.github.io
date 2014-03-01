---
layout: post
title: Actually, I'll mutate if you don't mind
Tags: design, code, net
permalink: actually-i-ll-mutate-if-you-don-t-mind
---

After I had changed all my extension methods to be functions and return a new object rather than mutating the self parameter, I changed them all back to be refs.

Why? Well mainly because the library I am writing is in VB, and these methods are internal.  VB supports ByRef parameters as the first param in an extension method, so no problems there.  The only reason I was changing them so that they were C# compatible was so that I could test them with MSpec in C#. I solved this little dilemma by just calling the extension method on the static class like so:

	Because of = () => EnumExtensions.Add(ref testEnum, (int)FlagsTest.Four);

This works, and lets me use the extensions how I think they should work.  The real question is why do I think my flags methods (Add, Remove) should mutate the instance, when I am quite happy with `string` and `DateTime` methods returning new instances?  I think it might be in the naming conventions.

A `List<T>` has `Add` and `Remove` methods, which modify the existing instance.  SO maybe if I had called my methods `WithFlag()` and `WithoutFlag()` I wouldn't have expected mutation?  I'm not entirely convinced as `DateTime` has `AddMinutes` and `AddHours`, which don't mutate and return a new instance.  Now that I think about it, that surprised me when I first used them.  I think, as usual, itâ€™s down to doing what makes the most sense in the situation.
