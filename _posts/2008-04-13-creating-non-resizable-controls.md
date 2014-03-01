---
layout: post
title: Creating Non resizable controls
Tags: design, code, controls, net
permalink: creating-non-resizable-controls
---

A control I was recently developing required being non-resizable when on the form.  When the application is running, this would be easy enough, just set its `AutoSize` property to False, and don't dock the control.

However, this leaves the problem of resizing in the designer.  You could override the resize event of the control, but for "reasons outlined earlier":http://www.stormbase.net/index.php?id=32, such as flickering, I decided against this.

Somewhere on the internet (where else...?) I can upon the idea of using a custom designer.  The ControlDesigner Class allows us to specify the designer behaviours of the control it is attached to.

To do this, we create Friend Class, and make it inherit from `System.Windows.Forms.Design.ControlDesigner`, then override the SelectionRules property:

pre(prettyprint). 
 Friend Class NonResizableDesigner
 	Inherits System.Windows.Forms.Design.ControlDesigner
 
 	Public Overrides ReadOnly Property SelectionRules() As System.Windows.Forms.Design.SelectionRules
 		Get
 			Return MyBase.SelectionRules
 		End Get
 	End Property
 End Class

As SelectionRules is a FlagsEnum, to remove the particular functionality from it, we have to NOT the flag we want to remove, then AND it with the existing flags.  In other words, take the controls existing flags and add `And Not SelectionRules.AllSizeable` to it.  So the entire designer class becomes this:

pre(prettyprint). 
 Friend Class NonResizableDesigner
 	Inherits System.Windows.Forms.Design.ControlDesigner
 
 	Public Overrides ReadOnly Property SelectionRules() As System.Windows.Forms.Design.SelectionRules
 		Get
 			Return MyBase.SelectionRules And Not SelectionRules.AllSizeable
 		End Get
 	End Property
 End Class

Simple huh?  Now all we need to do is apply it to the control that we wish to be non-resizable, which just takes one attribute on the class:

pre(prettyprint). 
 <Designer(GetType(NonResizableDesigner))> _
 Public Class Foo
 	Public Function Bar()
 		Return False
 	End Function
 End Class

Now when this control is viewed in the designer, it has the same outline as a label when the AutoSize property is set to true.  You can move the control to your hearts content, but no resizing.
