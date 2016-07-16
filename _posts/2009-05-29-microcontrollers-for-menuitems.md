---
layout: post
title: Microcontrollers for MenuItems
tags: design code generics controls net

---

I have been working my way through Jeremy Miller's excellent [Build Your Own CAB Series][jeremy-cab] (which would be even better if he felt like finishing!) and was very interested by the article on controlling menus with [Microcontrollers][jeremy-micro].

After reading it and writing a version of it myself, I came to the conclusion that some parts of it seem to be wrong.  All of the permissioning is done based on the menu items which fire `ICommands`, and several menu items could use the same `ICommand`.  This means that you need to use the interface something like this:

{% highlight vbnet %}
MenuController.MenuItem(mnuFileNew).Executes(Commands.Open).IsAvailableToRoles("normal", "editor", "su");
MenuController.MenuItem(tsbStandardNew).Executes(Commands.Open).IsAvailableToRoles("normal", "editor", "su");
{% endhighlight %}

Now to me this seems somewhat wrong, I would rather have something like this:

{% highlight vbnet %}
MenuController.Command(new MenuCommands.New).IsAttachedTo(mnuFileNew, tsbStandardNew).IsAvailableToRoles("normal", "editor", "su");
{% endhighlight %}

So I decided to have a go at re-working it to my liking.  To start with we have the mandatory `ICommand` interface:

{% highlight vbnet %}
Public Interface ICommand
    Sub Execute()
End Interface
{% endhighlight %}

Then a class that manages the actual `ICommand` and its menuitem(s):

{% highlight vbnet %}
Public NotInheritable Class CommandItem(Of T)
    Implements IDisposable      'used to remove handlers that we dont want to leave lying around

    Private ReadOnly _command As ICommand
    Private ReadOnly _id As T

    Private _roles As New List(Of String)
    Private _menuItems As New List(Of ToolStripItem)
    Private _alwaysEnabled As Boolean = False
    Private _disposed As Boolean = False

    Public Property AlwaysEnabled() As Boolean
        Get
            Return _alwaysEnabled
        End Get
        Set(ByVal value As Boolean)
            _alwaysEnabled = value
        End Set
    End Property

    Public Property Roles() As List(Of String)
        Get
            Return _roles
        End Get
        Set(ByVal value As List(Of String))
            _roles = value
        End Set
    End Property

    Public ReadOnly Property MenuItems() As ToolStripItem()
        Get
            Return _menuItems.ToArray
        End Get
    End Property

    Public ReadOnly Property IsDisposed() As Boolean
        Get
            Return _disposed
        End Get
    End Property

    Public Sub New(ByVal cmd As ICommand, ByVal id As T)
        _command = cmd
        _id = id
    End Sub

    Public Sub AddMenuItem(ByVal menuItem As ToolStripItem)
        _menuItems.Add(menuItem)
        AddHandler menuItem.Click, AddressOf _item_Click
    End Sub

    Public Sub RemoveMenuItem(ByVal menuItem As ToolStripItem)
        RemoveHandler menuItem.Click, AddressOf _item_Click
        _menuItems.Remove(menuItem)
    End Sub

    Public Function IsEnabled(ByVal state As CommandState(Of T)) As Boolean

        If _alwaysEnabled Then Return True

        If Not state.IsEnabled(_id) Return False

        For i As Integer = 0 To _roles.Count - 1
            If Thread.CurrentPrincipal.IsInRole(_roles(i)) Then Return True
        Next

        Return False

    End Function

    Public Sub SetState(ByVal state As CommandState(Of T))

        Dim enabled As Boolean = IsEnabled(state)

        For Each ts As ToolStripItem In _menuItems
            ts.Enabled = enabled
        Next

    End Sub

    Public Sub Dispose(ByVal disposing As Boolean)

        If Not _disposed AndAlso disposing Then

            For Each menuItem As ToolStripItem In _menuItems
                RemoveMenuItem(menuItem)
            Next

        End If

        _disposed = True

    End Sub

    Public Sub Dispose() Implements IDisposable.Dispose
        Dispose(True)
        GC.SuppressFinalize(Me)
    End Sub

    Private Sub _item_Click(ByVal sender As Object, ByVal e As EventArgs)
        _command.Execute()
    End Sub

End Class
{% endhighlight %}

As you can see, the `Dispose` Method is used to allow for handlers to be removed, otherwise the objects might be hanging around longer than they should be. We also have a list of menu items that this command controls, and a list of roles that the command is available to.

Next we have the class that holds the state of each menu item, which is generic to allow the end user to use whatever they wish to identify each menu item:

{% highlight vbnet %}
Public NotInheritable Class CommandState(Of T)

    Private _enabledCommands As New List(Of T)

    Public Function Enable(ByVal id As T) As CommandState(Of T)
        If Not _enabledCommands.Contains(id) Then
            _enabledCommands.Add(id)
        End If

        Return Me
    End Function

    Public Function Disable(ByVal id As T) As CommandState(Of T)
        If _enabledCommands.Contains(id) Then
            _enabledCommands.Remove(id)
        End If

        Return Me
    End Function

    Public Function IsEnabled(ByVal id As T) As Boolean
        Return _enabledCommands.Contains(id)
    End Function

End Class
{% endhighlight %}

Finally we have the Manager class which stitches the whole lot together with a health dollop of Fluent Interfaces.  We have a unique list of Commands (as I wrote this in VS2005, I just had to make a unique List class, rather than use a dictionary of `CommmandItem` and `Null`) and a sub class which provides the Fluent Interface to the manager. (`IDisposeable` parts have been trimmed out for brevity, it's just contains a loop that disposes all child objects).

{% highlight vbnet %}
Public NotInheritable Class Manager(Of T)

    Private _commands As New UniqueList(Of CommandItem(Of T))

    Public Function Command(ByVal cmd As ICommand, ByVal id As T) As CommandExpression
        Return New CommandExpression(Me, cmd, id)
    End Function

    Public Sub SetState(ByVal state As CommandState(Of T))

        For Each ci As CommandItem(Of T) In _commands
            ci.SetState(state)
        Next

    End Sub

    Public NotInheritable Class CommandExpression

        Private ReadOnly _manager As Manager(Of T)
        Private ReadOnly _commandItem As CommandItem(Of T)

        Friend Sub New(ByVal mgr As Manager(Of T), ByVal cmd As ICommand, ByVal id As T)

            _manager = mgr
            _commandItem = New CommandItem(Of T)(cmd, id)
            _manager._commands.Add(_commandItem)

        End Sub

        Public Function IsAttachedTo(ByVal menuItem As ToolStripItem) As CommandExpression
            _commandItem.AddMenuItem(menuItem)
            Return Me
        End Function

        Public Function IsInRole(ByVal ParamArray roles() As String) As CommandExpression
            _commandItem.Roles.AddRange(roles)
            Return Me
        End Function

        Public Function IsAlwaysEnabled() As CommandExpression
            _commandItem.AlwaysEnabled = True
            Return Me
        End Function

    End Class

    Private Class UniqueList(Of TKey)
        Inherits List(Of TKey)

        Public Shadows Sub Add(ByVal item As TKey)
            If Not MyBase.Contains(item) Then
                MyBase.Add(item)
            End If
        End Sub

    End Class

End Class
{% endhighlight %}

In my test application I have a file containing my menuCommands and an Enum used for identification:

{% highlight vbnet %}
Namespace MenuCommands
    Public Enum Commands
        [New]
        Open
        Save
        Close
    End Enum

    Public Class Open
        Implements ICommand

        Public Sub Execute() Implements ICommand.Execute
            MessageBox.Show("Open")
        End Sub

    End Class
End Namespace
{% endhighlight %}

And in the main form I have this code.  The Thread Principle is used for the roles, and the actual roles could (should) be loaded from a database or anywhere other than hard coded constants of course.

{% highlight vbnet %}
Private _menuManager As New Manager(Of MenuCommands.Commands)
Private _state As New CommandState(Of MenuCommands.Commands)

Private Sub Form1_Load(ByVal sender As System.Object, ByVal e As System.EventArgs) Handles MyBase.Load


    Thread.CurrentPrincipal = New GenericPrincipal(Thread.CurrentPrincipal.Identity, New String() {"normal"})

    _menuManager.Command(New MenuCommands.[New], MenuCommands.Commands.[New]) _
                .IsAttachedTo(mnuFileNew) _
                .IsAttachedTo(tsbNew) _
                .IsInRole("normal")

    _menuManager.Command(New MenuCommands.Open, MenuCommands.Commands.Open) _
                .IsAttachedTo(mnuFileOpen) _
                .IsAttachedTo(tsbOpen) _
                .IsInRole("normal", "reviewer", "viewer")

    _menuManager.Command(New MenuCommands.Save, MenuCommands.Commands.Save) _
                .IsAttachedTo(mnuFileSave) _
                .IsAttachedTo(tsbSave) _
                .IsInRole("normal", "reviewer")

    _menuManager.Command(New MenuCommands.Close, MenuCommands.Commands.Close) _
                .IsAttachedTo(mnuFileExit) _
                .IsAlwaysEnabled()

    _state.Enable(MenuCommands.Commands.Open) _
          .Enable(MenuCommands.Commands.Save) _
          .Enable(MenuCommands.Commands.Close)

    _menuManager.SetState(_state)

End Sub
{% endhighlight %}

The state object is used to enable and disable menu items and could be wrapped in another object if it needed to be exposed further than the form.

[jeremy-cab]: http://codebetter.com/blogs/jeremy.miller/archive/2007/07/25/the-build-your-own-cab-series-table-of-contents.aspx
[jeremy-micro]: http://codebetter.com/blogs/jeremy.miller/pages/build-your-own-cab-14-managing-menu-state-with-microcontroller-s-command-s-a-layer-supertype-some-structuremap-pixie-dust-and-a-dollop-of-fluent-interface.aspx

