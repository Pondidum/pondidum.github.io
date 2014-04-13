---
layout: post
title: VB.NET &amp; C# Fixed height User Controls
tags: design, code, controls, net
permalink: vbnet-c-fixed-height-user-controls
---

Another problem I came across recently was fixed height user controls.  Someone at work had created a fixed height user control, by putting the following code in the paint event:

{% highlight vbnet %}
Me.Width = 20
{% endhighlight %}

Now while for the majority of cases this works, it doesn't if you dock the control to the left or right of the form, as each time the Layout Engine tries to stick the top of the control to the top of the parent and the bottom of the control to the bottom of the parent, it fires the `Paint()` event.  This causes the user control to change its size, which causes the Layout Engine to activate, and the whole cycle starts over, and as a by product, creates a horrid flickering.

Some suggestions were made to fix the problem such as disabling docking (why fix a problem by causing another one?), moving the code to the resize event (same effect, with the added benefit of allowing a resize until it is complete, then resizing...).

Some googling revealed one very angry fellow on the [xtreme dot net talk][xtreme-dot-net] forums, and no real answer.  The method he had tried was to set the following flag in the initialize event:

{% highlight vbnet %}
Control.SetStyle(ControlStyles.FixedHeight, true)
{% endhighlight %}

Which if you read the documentation for ControlStyles.FixedHeight (it's on the intellitype, so there's no reason for not doing so) it says the following:

If true, the control has a fixed height when auto-scaled. For example, if a layout operation attempts to rescale the control to accommodate a new Font, the control's Height remains unchanged.

So another solution was needed.  In the end, I and a fellow developer found that overriding the controls MaximumHeight and MinimumHeight was the way to do it:

{% highlight vbnet %}
Const MaxHeight As Integer = 20

Public Overrides Property MaximumSize() As Drawing.Size
	Get
		Return New Drawing.Size(MyBase.MaximumSize.Width, MaxHeight)
	End Get
	Set(ByVal value As Drawing.Size)
		MyBase.MaximumSize = New Drawing.Size(value.Width, MaxHeight)
	End Set
End Property

Public Overrides Property MinimumSize() As Drawing.Size
	Get
		Return New Drawing.Size(MyBase.MinimumSize.Width, MaxHeight)
	End Get
	Set(ByVal value As Drawing.Size)
		MyBase.MinimumSize = New Drawing.Size(value.Width, MaxHeight)
	End Set
End Property
{% endhighlight %}

This allows the end user to modify the maximum width (in this case) to their heart's content, and still have a control of fixed height, that can be docked properly, doesn't flicker, and above all resizes properly in the forms designer.

[xtreme-dot-net]: http://www.xtremedotnettalk.com/showthread.php?t=94118
