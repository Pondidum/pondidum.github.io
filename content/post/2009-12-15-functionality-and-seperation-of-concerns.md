---
date: "2009-12-15T00:00:00Z"
tags: ["design", "c#"]
title: Functionality and Seperation of Concerns
---

When I am writing a winform in an MVP style, I often wonder how far to go with the separation.  Say I have the following situation:

A small form which should display a list of messages, and allow the user to select which ones they want processed.  It processes each message in turn.  If a message has more than one attachment, a dialog is shown to ask the user to select which attachment should be used for that message.

Now while this is fairly simple, my interface for the message dialog looks like this:

```vb
Public Interface IMessageSelector

	Event Submit()
	Event Cancel()

	WriteOnly Property Messages() As IList(Of MessageData)

	ReadOnly Property Selected() As IList(Of String)
	ReadOnly Property AttachmentView() As IAttachmentScreen

	Sub ShowScreen()
	Sub CloseScreen()
	Sub DisplayWarning(ByVal text As String)

End Interface
```

In the form I have (roughly) the following:

```vb
Public Class frmMessages
	Implements IMessageSelector
	'...'

	Public WriteOnly Property Messages() As IList(Of MessageData) Implements IMessageSelector.Messages
		Set(ByVal value As IList(Of MessageData))

			For Each d As MessageData In value

				Dim r As Grid.Row = grid.Rows.Add
				r("id") = d.ID
				r("subject") = d.Subject
				r("from") = d.Sender
				r("received") = d.SendDate

			Next

			flx.AutoSizeCols()

		End Set
	End Property

	Public ReadOnly Property Selected() As IList(Of String) Implements IMessageSelector.Selected
		Get
			Dim result As New List(Of String)

			For i As Integer = 1 To grid.Rows.Count - 1

				If Convert.ToBoolean(grid(i, "selected")) Then
					result.Add(grid(i, "id").ToString)
				End If

			Next

			Return result

		End Get
	End Property

End Class
```

Now I think that this is ok.  There is not logic as such in the population property, and the Selected property just determines which rows have had their checkboxes ticked.

However it has been requested that I add a 'Select All/None' checkbox to the form.  Where do I add the code for this?  As they want a checkbox to tick or detick its not as trivial as it could be.  If it were separate buttons, I could just use a for loop in each setting the values to True or False.  A checkbox however has some uncertainties:

 - Checking the master checkbox should make all rows checked. Fine.
 - DeChecking the master checkbox should make all rows unchecked. Also fine.
 - Checking one row when none are checked should do what to the master checkbox?
 - DeChecking one row when all are checked should do what to the master checkbox?
 - 25%/50%/75% of rows are checked, what does the master checkbox look like?
 - Some rows are checked.  What happens when the checkbox is clicked?

So many questions for such a simple looking feature.  With so many possibilities for it maybe it should go into the presenter/interface?  At least it's testable then.  Maybe a separate controller for it as it's not really anything to do with the *purpose* of the form?

If anyone knows of answers to this I would be very interested to hear them.
