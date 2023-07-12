+++
date = '2008-03-29T00:00:00Z'
tags = ['bug']
title = 'Flow Layout Panel and Scroll Wheel Problem'

+++

I came across a problem while writing an application for my parents today.  If you have a FlowLayoutPanel on your form, and have many items in it, causing it to overflow and require scroll bars, you are unable to scroll the control's content using the mouse wheel.

This is somewhat trying in today's applications, as nearly everyone has a mouse with a scroll wheel, or a track pad on their laptop with a scroll area.

I came across the solution to the problem at Scott Waldron's [blog][flow-layout-panel], where he adds code to the Flow Layout Panel's MouseEnter Event that focuses the Flow Layout Panel, and thus allows it to be scrolled with the wheel.  Many thanks to him for finding the solution and blogging about it.

My real bone of contention with this is that the Flow Layout Panel does not support this behaviour by default.  If it is a feature not everyone wants, just put a Boolean property on the control called "Accepts Scroll Wheel" or something similar, and have it False by default.

[flow-layout-panel]: http://www.thewayofcoding.com/2008/02/c-net-programming-tip-flowlayoutpanel.html
