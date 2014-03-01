---
layout: post
title: Using Visual Studio's Regex Find and Replace
Tags: code, net
permalink: using-visual-studio-s-regex-find-and-replace
---

The Visual Studio Find and Replace dialog is often overlooked, and when parts of it are looked at (Regex searching) it often gets a bad rep.  Sure it doesnâ€™t implement all of the Regex syntax (non greedy search springs to mind), but thatâ€™s not to say it isn't useful.  

For instance, I was working on some code that involved a Model View Presenter type style, but used Subroutines (void methods) rather than WriteOnly properties for brevity (in C# you can do a Set only property in 1 line, VB it takes 5).  As the View is doing nothing other than assigning labels from these "Setters" who cares how many lines it takes?

A quick breakdown of the parts of the expressions used:

	Finding:
	{}		//Tag an expression, used in replacements.  Numbered sequentially from 1, not 0.
	(.*)	//Any character, any number of times, as many as possible.
	\		//escape character, allows us to search for a literal '.' or other Regex used symbol.
	
	Replacing:
	\1		//The content of a tagged expression.
	\n		//New line
	\t		//Tab (although after running all these find and replaces, a quick {CTRL+E, CTRL+D} (format document) does most of the tidying for you).

So we start with the Interface:

	Public Interface IProcessDetailsView
		Sub FileID(ByVal value As Integer)
		Sub SubmittedBy(ByVal value As String)
		Sub ReceivedDate(ByVal value As DateTime)
		//...
	End Interface
	
So in the find and replace dialog I enter the following:
	
	Find what: 
	Sub {(.*)}\(ByVal value As {(.*)}\)
	
	Replace with:
	WriteOnly Property \1() As \2

The interface definition now changes to this: 

	Public Interface IProcessDetailsView
		WriteOnly Property FileID() As Integer
		WriteOnly Property SubmittedBy() As String
		WriteOnly Property ReceivedDate() As DateTime
		//...
	End Interface

Not too difficult right?  Good. Now onto the View's methods:

	Public Class ProcessDetails
		Implements IProcessDetailsView

		Public Sub FileID(ByVal value As Integer) Implements IProcessDetailsView.FileID
			lblFileID.Text = value.ToString
		End Sub

		Public Sub SubmittedBy(ByVal value As String) Implements IProcessDetailsView.SubmittedBy
			lblAccountName.Text = value
		End Sub
		
		//...
	End Class
	
Into the find and replace dialog:

	Find what: 
	Public Sub {(.*)}\(ByVal value As {(.*)}\) Implements IProcessDetailsView\.(.*)
	
	Replace with:
	Public WriteOnly Property \1() As \2 Implements IProcessDetailsView.\1\n\t\tSet(ByVal value As \2)
	
	Find what: 
	End Sub
	
	Replace with:
	End Set\n\tEnd Property

You could do this with one expression, although I have found its far less hassle to use two find and replace runs rather than trying to find new lines etc

	Public Class ProcessDetails
		Implements IProcessDetailsView
		
		Public WriteOnly Property FileID() As Integer Implements IProcessDetailsView.FileID
			Set(ByVal value As Integer)
				lblFileID.Text = value.ToString
			End Set
		End Property

		Public WriteOnly Property SubmittedBy() As String Implements IProcessDetailsView.SubmittedBy
			Set(ByVal value As String)
				lblAccountName.Text = value
			End Set
		End Property
		
	End Class
	
Now the main reason for this change was the presenter code, which doesnâ€™t sit right with me.  At a glance, am I expecting something to be calculated or what?

	Public Sub Display(ByVal processHistory As ICVProcessHistory)
		_view.FileID(processHistory.FileID)	
		_view.SubmittedBy(processHistory.AccountName)
		//...
	End Sub
	
Find and replace dialog again:

	Find what: 
	\_view\.{(.*)}\({(.*)}\.{(.*)}\)
	
	Replace with:
	_view.\1 = \2.\3
	
Which gives us this:

	Public Sub Display(ByVal processHistory As ICVProcessHistory)
		_view.FileID = processHistory.FileID
		_view.SubmittedBy = processHistory.AccountName
		//...
	End Sub
	
Much better in my opinion.