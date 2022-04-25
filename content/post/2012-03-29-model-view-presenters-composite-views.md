---
date: "2012-03-29T00:00:00Z"
tags: ["design", "c#"]
title: 'Model View Presenters: Composite Views'
---

Table of Contents:
------------------
* [Introduction][1]
* [Presenter to View Communication][2]
* [View to Presenter Communication][3]
* **Composite Views**
* Presenter / Application communication
* ...

When working with MVP, it won't be long before you come across the need for multiple views on one form.  There are several ways to achive this, and which you choose is really down to how you intend to (re)use your views.

![Composite View][4]

The first method for dealing with the sub views is to expose them as a property of your main view, and set them up in the main view's presenter:

```csharp
interface IMainView
{
	ISubView1 View1 { get; }
	ISubView2 View2 { get; }

	/* Other properties/methods etc for MainView */
}

class MainView : Form, IMainView
{
	public ISubView1 View1 { get { return this.subView1; } }
	public ISubView2 View2 { get { return this.subView2; } }
}

class MainPresenter
{
	private readonly IMainView _view;
	private readonly SubPresenter1 _pres1;
	private readonly SubPresenter2 _pres2;

	public MainPresenter(IMainView view)
	{
		_view = view;
		_pres1 = new SubPresenter1(view.View1);
		_pres2 = new SubPresenter2(view.View2);
	}

}

static class Program
{
	static void Main()
	{
		using (var view = new MainView())
		using (var presenter = new MainPresenter(view))
		{
			presenter.Display();
		}
	}
}
```

This method's advantage is simplicity, just create a new view and presenter, and call `Display`.  The disadvantage is that the main presenter is tied to the sub presenters.  A slight modification alleviates this:

```csharp
interface IMainView
{
	ISubView1 View1 { get; }
	ISubView2 View2 { get; }

	/* Other properties/methods etc for MainView */
}

class MainView : Form, IMainView
{
	public ISubView1 View1 { get { return this.subView1; } }
	public ISubView2 View2 { get { return this.subView2; } }
}

class MainPresenter
{
	private readonly IMainView _view;
	private readonly SubPresenter1 _pres1;
	private readonly SubPresenter2 _pres2;

	public MainPresenter(IMainView view, SubPresenter1 pres1, SubPresenter2 pres2)
	{
		_view = view;
		_pres1 = pres1;
		_pres2 = pres2;
	}
}

static class Program
{
	static void Main()
	{
		using (var view = new MainView())
		using (var pres1 = new SubPresenter1(view.View1));
		using (var pres2 = new SubPresenter2(view.View2));
		using (var presenter = new MainPresenter(view, pres1, pres2))
		{
			presenter.Display();
		}
	}
}
```

The only change here is to pass our two sub presenters in to the main presenter as constructor parameters.  Ultimately this seems to be the 'best' solution from a coupling point of view, however, if you are unlikely to change the sub presenters out for completely different sub presenters, then I would use the first method.

The final method for composing sub views is to push the responsibility to the actual main view, and make your main view pass any events and data to and from the sub view:

```csharp
interface IMainView
{
	String FirstName { get; set; }
	String LastName { get; set; }

	String AddressLine1 { get; set; }
	String PostCode { get; set; }

	/* Other properties/methods etc for MainView */
}


class MainView : Form, IMainView
{
	private readonly SubPresenter1 _pres1;
	private readonly SubPresenter2 _pres2;

	void MainView()
	{
		InitializeComponent();
		_pres1 = new SubPresenter1(subView1);
		_pres2 = new SubPresenter2(subView2);
	}

	String FirstName
	{
		get { return subView1.FirstName; }
		set {subView1.FirstName = value;}
	}

	String LastName
	{
		get { return subView1.LastName; }
		set { subView1.LastName = value; }
	}

	String AddressLine1
	{
		get { return subView2.AddressLine1; }
		set { subView2.AddressLine1 = value; }
	}

	String PostCode
	{
		get { return subView2.PostCode; }
		set { subView2.PostCode = value; }
	}
}
```

The disadvantage to this is that if one of the subViews were to change in anyway, the MainView also has to change to reflect this.

Out of the three methods outlined, Method 2 is my personal preference, especially when not using a DI Container, and Method 2 when I am using one.  The 3rd Method I find is too brittle for most usage, especially during earlier stages of development when the UI is more likely to be changing.

[1]: /model-view-presenter-introduction
[2]: /model-view-presenters-presenter-to-view-communication
[3]: /model-view-presenters-view-to-presenter-communication
[4]: /images/mvp-sub-view-diagram.jpg
