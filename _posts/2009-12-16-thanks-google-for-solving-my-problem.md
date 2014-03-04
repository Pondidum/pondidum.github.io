---
layout: post
title: Thanks Google for solving my problem!
Tags: design, code, net
permalink: thanks-google-for-solving-my-problem
---

Following on from [yesterday's][1] post about separation on concerns and where to put some undefined logic for a multi state checkbox, I did a fair amount of research.

I must say the [Quince][2] website is a good repository of UI Design Patterns, as is [Welie][3].  I couldn't find anything like what I was after, which I guess means I shouldn't be doing it this way?

After a while a brainwave struck me: "Gmail lets you select things, how does it do it?  One click on the Gmail icon and I'm presented with this:

![Gmail Selection][4]

Perfect.  So I went back to my sponsor and showed them a mock-up with this style of selection.  The reaction was: "Oh I like that". Excellent news, for me its easier code to write (I'm happy with a for loop setting a grid cell to true in the view) and if they want to add other selections its easy enough (though there is not much else they could select by...).

The moral of the story?  If in doubt, copy Google.

[1]: /functionality-and-seperation-of-concerns
[2]: http://quince.infragistics.com
[3]: http://www.welie.com
[4]: /images/gmail-selection.jpg