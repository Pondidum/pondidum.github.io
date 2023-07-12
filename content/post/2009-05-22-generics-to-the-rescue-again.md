+++
date = '2009-05-22T00:00:00Z'
tags = ['design', 'generics', 'c#']
title = 'Generics to the rescue! Again!'

+++

I was writing a component at work that has many events that all need to be thread safe, and was getting annoyed at the amount of duplicate code I was writing:

```vb
Public Event FilterStart(ByVal sender As Object, ByVal e As EventArgs)
'...
Private Delegate Sub OnFilterCompleteDelegate(ByVal sender As Object, ByVal e As FilterCompleteEventArgs)
'...
Private Sub OnFilterComplete(ByVal sender As Object, ByVal e As DataAccess.LoadEventArgs)
    If _parent.InvokeRequired Then
        _parent.Invoke(new OnFilterCompleteDelegate(AddressOf OnFilterComplete), new Object() {sender, e})
    Else
        RaiseEvent FullResultsStart(sender, e)
    End If
End Sub
'... repeat for all
```

Hmm. There has to be a better way of doing this. Enter some Generic magic in the form of a Generic Delegate Sub:

```vb
Private Delegate Sub EventAction(Of TArgs)(ByVal sender As Object, ByVal args As TArgs)
```

This then allows me to write my Event Raisers like so:

```vb
Private Delegate Sub EventAction(Of TArgs)(ByVal sender As Object, ByVal args As TArgs)

Private Sub OnFilterStart(ByVal sender As Object, ByVal e As EventArgs)
    If _parent.InvokeRequired Then
        _parent.Invoke(New EventAction(Of EventArgs)(AddressOf OnFilterStart), New Object() {sender, e})
    Else
        RaiseEvent FilterStart(sender, e)
    End If
End Sub
```

Further optimisation let me do the fiollowing, as the sender is always `Me` :

```vb
Private Sub OnFilterStart(ByVal e As EventArgs)
    If _parent.InvokeRequired Then
        _parent.Invoke(New Action(Of EventArgs)(AddressOf OnFilterStart), New Object() {e})
    Else
        RaiseEvent FilterStart(Me, e)
    End If
End Sub
```

Which meant I no longer needed my customer Action Delegate, as there is one for a single parameter in System for this already!

Now if only I could find a way to wrap the thread safe checks and invokes into a single generic function...
