+++
date = '2012-01-31T00:00:00Z'
tags = ['design', 'c#']
title = 'Model View Presenters: View to Presenter Communication'

+++

Table of Contents:
------------------
* [Introduction][3]
* [Presenter to View Communication][4]
* **View to Presenter Communication**
* [Composite Views][5]
* Presenter / Application communication
* ...

Communicating from the View to the Presenter is a reasonably straight forward affair.  To signal something happening, we use an `Event`, but one with no parameters.  We pass no parameters, as we are not going to be using them anyway, so what is the point is raising an event every time with `OkayClicked(this, EventArgs.Empty)`?

To get around this, we define a new event type, so that we can get rid of our redundant parameters:

```csharp
public delegate void EventAction();
```

In the View we define our events:

```csharp
public interface IEmployeesView
{
	event EventAction OkayClicked;
	event EventAction CancelClicked;
}
```

And in the Presenter we hook up the events:

```csharp
public class EmployeesPresenter : IDisposable
{
	private readonly IEmployeeView _view;

	public EmployeesPresenter(IEmployeesView view)
	{
		_view = view;
		_view.OkayClicked += OnOkayClicked;
		_view.CancelClicked += OnCancelClicked;
	}

	private void OnOkayClicked() { /* ... */ }
	private void OnCancelClicked() { /* ... */ }

	public void Dispose()
	{
		_view.OkayClicked -= OnOkayClicked;
		_view.CancelClicked -= OnCancelClicked;
	}
}
```

Now I don't know about you, but I dislike having to wire and unwire an event like this - there is too much chance that I will either forget to unwire one of the events, or when copying and pasting the Add code (I know, I know) to the Remove section, I will forget to change a `+` to a `-`.

To this end, I created a class that will auto wire events to handlers based on a convention.  It was based off of the Presenter base class written by [Mark Nijhof][1] in his [Fohjin.DDD][2] sample application, with a few improvements (namely the unwiring of events).  To avoid the need to make your Presenters inherit from a base class (and to not violate SRP!), it is wrapped up into its own class, which can be used like so:

```csharp
public class EmployeesPresenter : IDisposable
{
	private readonly IEmployeeView _view;
	private readonly EventAutoWirer<IEmployeeView> _autoWire;

	public EmployeesPresenter(IEmployeesView view)
	{
		_view = view;
		_autoWire = new EventAutoWire<IEmployeeView>(view, this);
		_autoWire.Wire();
	}

	private void OnOkayClicked() { /* ... */ }
	private void OnCancelClicked() { /* ... */ }

	public void Dispose()
	{
		_autoWire.Unwire();
	}
}
```

The wirer supports conventions to allow easier hook-up - by default it will only hook events defined in the View's interface to private methods prefixed with "On" in the presenter.  This means that any events which are only defined in the concrete implementation of the View are not wired automatically - allowing extra view only functionality such as when an item is MouseOver'd etc.

[1]: http://cre8ivethought.com/blog/index
[2]: https://github.com/MarkNijhof/Fohjin

[3]: /model-view-presenter-introduction
[4]: /model-view-presenters-presenter-to-view-communication
[5]: /model-view-presenters-composite-views
