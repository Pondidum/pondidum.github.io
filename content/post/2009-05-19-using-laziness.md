---
date: "2009-05-19T00:00:00Z"
title: Using Laziness
---

As I do a lot of forms development, I end up writing something like this a lot:

```vb
Try
    pnlSomething.SuspendLayout()
    '...
Finally
    pnlSomething.ResumeLayout()
End Try
```

Now as I am lazy, I thought I could make a class to do this for me:

```vb
Public Class Layout
    Implements IDisposable

    Private _control As Control

    Public Sub New(ByVal control As Control)
        _control = control
        _control.SuspendLayout()
    End Sub

    Public Sub Dispose() Implements IDisposable.Dispose
        _control.ResumeLayout()
        _control = Nothing
    End Sub

End Class
```

It is used like this:

```vb
Using l As New Layout(FlowLayoutPanel1)

    For I As Integer = 0 To 500
        Dim chk As New CheckBox
        chk.Name = i.ToString
        chk.Text = i.ToString
        chk.Parent = FlowLayoutPanel1
    Next

End Using
```

I suppose I haven't saved any typing, but I think it looks better...whether I will actually use it is another matter.  I might see if it's possible to extend it to do other things.  On the other hand I might not bother ;\
