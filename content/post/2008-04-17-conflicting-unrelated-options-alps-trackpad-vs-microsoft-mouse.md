---
date: "2008-04-17T00:00:00Z"
tags: bug
title: 'Conflicting Unrelated Options: Alps Trackpad vs Microsoft Mouse'
---

Usability.  It's one of those things that everyone wants, you know, stuff that 'just works'.  It's nice when companies go out of their way to make things 'just work'.  It's a shame Sony (and others, but I have a Sony, so it's their fault for this exercise) decided to make things harder for me.

Allow me to explain.  I have a Sony Vaio (VGN-FE21M), which has an Alps touch pad, which like all touch pad has tapping enabled by default, and that I switch off straight away.  I also have a Microsoft Intellimouse 4, which has smooth scrolling, and scroll acceleration.

Now despite the fact that these two settings are completely unrelated, you can't have both at the same time.  The problem is that if I have the touch pad driver installed, it's scrolling setting overwrites the intellimouse's scroll setting, but I need the touchpad driver to disable tapping.

I tried the Vista drivers for both mice; I tried the XP drivers for both.  I even tried combinations of the drivers, and drivers for the touchpad from IBM and Siemens.  None worked.  I messed around in the registry with settings, broke lots of stuff, which also didn't help (luckily I did a registry backup before I started...)

Eventually I discovered the 'latest' drivers from Sony, IBM, Fujitsu and Siemens were not as new as they could be.  The Dell drivers which I (eventually) found were newer, and although not designed for my laptop worked a charm.  I now have a tap free track pad, and smooth scrolling.

If anyone else is having this problem, here  is a link to the Dell drivers: [32 Bit Vista][32-bit], [64 Bit Vista][64-bit].


[32-bit]: http://support.us.dell.com/support/downloads/download.aspx?c=us&l=en&s=gen&releaseid=R140031&formatcnt=1&libid=0&fileid=187207
[64-bit]: http://support.us.dell.com/support/downloads/download.aspx?c=us&l=en&s=gen&releaseid=R140029&formatcnt=1&libid=0&fileid=187206