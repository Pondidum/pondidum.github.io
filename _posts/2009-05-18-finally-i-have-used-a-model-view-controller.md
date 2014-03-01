---
layout: post
title: Finally, I have used a Model View Controller!
Tags: design, code
permalink: finally-i-have-used-a-model-view-controller
---

Today I actually managed to use a Model View Controller in an application.  I have been looking for an opportunity to use one fore a while, and have been reading a lot about them (Jeremy Miller's [Build Your Own CAB Series][build-your-own-cab] has been a very good guide).

The type of MVC I like most (so far) is the [Passive View][fowler-passive-view] type, where the View does almost nothing, and has no link to the Model:

!["Passive View" Model View Presenter](/images/passive-view.png)<br />
Image Source: [Microsoft][ms-passive-view]

There are two main ways of wiring your View to the Presenter/Controller: Events and Interfaces.  The advantage of using an Interface is that they are easier to test (using [Rhino Mocks][rhino-mocks]), but as work does not do unit testing (I'm working on it!), that didn't matter too much.  I used events in this case simply because I prefer them.

As we already have a data layer, and I was just designing a form to expose some functionality I didn't really use a Model either (unless a DAL counts, and I'm not sure it does).

In the end my Controller and Form looked something like this (much snipped, but you get the idea):

{% highlight vbnet %}
 Public Class SearchController

    Private _control As ISynchronizeInvoke

    Private Delegate Sub OnSearchDelegate(ByVal sender As Object, ByVal e As SearchEventArgs)

    Public Event SearchStarted(ByVal sender As Object, ByVal e As SearchEventArgs)
    Public Event SearchProgress(ByVal sender As Object, ByVal e As SearchEventArgs)
    Public Event SearchFinished(ByVal sender As Object, ByVal e As SearchEventArgs)

    Public Sub New(ByVal parent As ISynchronizeInvoke)
        _control = parent
    End Sub

    Private Sub OnSearchStarted(ByVal sender As Object, ByVal e As SearchEventArgs)
        If _control.InvokeRequired Then
            _control.Invoke(New OnSearchDelegate(AddressOf OnSearchStarted), New Object() {sender, e})
        Else
            RaiseEvent SearchStarted(sender, e)
        End If
    End Sub
    'snip for other events...

    Public Sub SetPhrase(ByVal phrase As String)
        '...
    End Sub

    Public Sub Search()
        OnSearchStarted(Me, New SearchEventArgs())
        '...
    End Sub
    '...
End Class

Public Class frmSearch

    Private _controller as new SearchController(Me)

    Private Sub btnSearch_Click(ByVal sender As System.Object, ByVal e As System.EventArgs)
        _controller.SetPhrase(txtInput.Text.Trim)
    End Sub

    Private Sub controller_SearchStarted(ByVal sender As Object, ByVal e As SearchEventArgs)
        '...
    End Sub
    '...
End Class
{% endhighlight %}

Hopefully I will get the opportunity to use MVC/MVP more completely in the future.

[build-your-own-cab]: http://codebetter.com/blogs/jeremy.miller/archive/2007/07/25/the-build-your-own-cab-series-table-of-contents.aspx
[fowler-passive-view]: http://martinfowler.com/eaaDev/PassiveScreen.html
[rhino-mocks]: http://hibernatingrhinos.com/oss/rhino-mocks
[ms-passive-view]: http://msdn.microsoft.com/en-us/library/cc304760.aspx
