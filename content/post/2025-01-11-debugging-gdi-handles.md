+++
title = 'Debugging GDI Handle Leaks'
tags = [ "dotnet", "story", "debugging" ]
+++

Many years ago, I was working on a dotnet Windows Forms application.  The application had many issues overall: memory leaks, random crashes, data loss, and in this case, the "red x" problem.

The problem showed up at random, and instead of a window, dialogue, or control being rendered, it would be replaced with a white box with a red outline and red diagonal cross, and I _think_ some error text in one corner, saying something about GDI handles.  The issue itself didn't seem to be related to either time or memory usage; when the app crashed, we got an error report (usually), and that never suggested that the application had been open particularly long or that it was using an excessive amount of memory.

I had been given the job of fixing this problem (and others), so armed with a memory profiler (RedGate's, if I recall correctly), I went to work.  With no given reproduction, it was hard.  Some searching had shown that GDI handles were created when doing custom control painting; usually, this would be a fairly strong indicator of where to look, but in the case of this application, nearly all controls were custom-drawn, and there were many.

After days of taking memory snapshots and running comparisons, the only thing I was really noticing was that the number of font instances in use seemed high and got higher whenever I opened dialogues; it never went down again.

So, something to do with fonts.  Reading through our common control base code, I noticed this:

```c#
public class ControlBase : Control
{
  private Font _normal = new Font("some-font", 12);
  private Font _bold = new Font("some-font", 12, Options.Bold);
  private Font _italic = new Font("some-font", 12, Options.Italic);

  // ...
}
```

This snippet means that every single control had 3 instances of a `Font`, which was never disposed (no `_normal.Dispose()` in the `ControlBase.Dispose()` function.)  My first reaction was to add the three `.Dispose()` calls, but realising the fonts were never modified after creation led me to make them `static` so that all control instances shared the same font instances.


A week or two of work, and the only output was adding 3x `static` words to the codebase - but the effect was that our application went from using 1000s of font instances to 3.  Quite the saving - and we never had the red X problem again!


The first lesson I took from this was that memory profiling is hard - running the app, taking a snapshot, running a bit more, taking a snapshot, etc. was not a fun feedback loop, especially as running with the profile slowed everything down massively.

The second, and probably more important, lesson was that the number of lines changed doesn't reflect the amount of effort that went into changing those lines.
