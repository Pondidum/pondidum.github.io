---
layout: post
title: Adding MSpec to your Git Bash
tags: net, git
permalink: adding-mspec-to-your-git-bash
---

My workflow involves Visual Studio, Notepad++ and Git Bash.  I don't use much Visual Studio integration, and prefer to run most things from the command line.

Now when it comes to testing projects, my tool of choice is MSpec (Machine.Specifications), which I decided would be nice if I could run from my Git Bash.

    $ mspec bin/project.specs.dll

To do this, you need to write a Shell Script with the following contents:

{% highlight bash %}
    #!/bin/sh
    "D:\dev\downloaded-src\machine.specifications\Build\Release\mspec.exe" "$*"
	#obviously change this to your mspec path...
{% endhighlight %}

Save it as `mspec` (no extension), and you can place it in one of two places:

* Your Home Directory: `C:\Users\<name>\`, useful if it's just for you
* The Git Bin Directory: `C:\Program Files\Git\bin`, for if you want all users to be able to run the script

Restart your git bash, and you can now use the command `mspec` to run all your specifications.
