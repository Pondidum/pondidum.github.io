---
date: "2009-07-14T00:00:00Z"
tags: ci
title: 'CI: Thoughts on CC.Net and Hudson'
---

I have been a fan of CI (Continuous Integration) for a long time now, and ever since I started with CI I have been using [CruiseControl.Net][1]. CCNet is incredibly powerful; you can make to do practically anything, and writing plugins for it is a breeze.

However, I do find that the config files get rather messy.  I have tried many things and the current best solution seems to be to have one 'master' config file with a set of includes to other files.  While this splits it all out nicely I find my config files are all very similar especially for projects which I build in Debug mode and in Release mode.  These configs are identical bar Build Location, and the `/p:Configuration=Debug` flag passed to MSBuild.  I have been reading about [Dynamic Parameters][2] and I think I can solve the problems with that, however time is a little short at work, so that is defiantly on the back burner.

I have also been reading a lot of good things about [Hudson][3] which while being a Java aimed CI Server, can be used with MSBuild through Nant, or plugins to let you use MSBuild directly (a nice guide is at [redsolo's blog]).  While I also have not had the time to have a proper play with it, I must say it does look very good.

It may still have messy configs (I don't know yet, haven't really looked), but as everything is done through a nice web interface rather than a CLI, who cares?  I was very impressed with how quick it was to get running too: `java -DHUDSON_HOME=data -jar hudson.war`. It uncompressed itself, and got going straight away. No messing with installers. Very nice.

The only thing I dislike so far is the background picture in the web interface.  So I deleted it.  Other than that (very) minor niggle, I think I like Hudson a lot, and look forward to playing around with it in the future.

[1]: http://confluence.public.thoughtworks.org/display/CCNET
[2]: http://confluence.public.thoughtworks.org/display/CCNET/Dynamic+Parameters
[3]: https://hudson.dev.java.net/
[4]: http://redsolo.blogspot.com/2008/04/guide-to-building-net-projects-using.html
