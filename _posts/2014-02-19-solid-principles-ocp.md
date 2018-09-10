---
layout: post
title: SOLID Principles - OCP
tags: design c# solid

---

## Open Closed Principle

[Single Responsibility][blog-solid-srp] | [Open Closed][blog-solid-ocp] | [Liskov Substitution][blog-solid-lsp] | [Interface Segregation][blog-solid-isp] | [Dependency Inversion][blog-solid-dip]

The Open Closed Principle is one that I often find is miss-understood - how can something be open for extension, but closed for modification?
A good example of this principle being implemented cropped up at work a while ago, we had a UI element which has a reusable grid, which gets populated with data based on a menu selection.  The user can also add, edit and delete items from the grids.

The class was originally implemented something like this:

```csharp
public class UserGrid
{

	public UserGrid()
	{
		_menu.Add(new ToolStripMenuItem { Text = "Emails", Tag = MenuTypes.Emails });
		_menu.Add(new ToolStripMenuItem { Text = "Addresses", Tag = MenuTypes.Addresses });
		_menu.Add(new ToolStripMenuItem { Text = "Phone Numbers", Tag = MenuTypes.Phones });
	}

	public void Populate()
	{
		var selection = GetMenuSelection();
		var rows = new List<DataGridViewRow>();

		switch (selection)
		{
			case MenuTypes.Emails:
				rows.AddRange(_user.EmailAddresses);
				break;

			case MenuTypes.Addresses:
				rows.AddRange(_user.Addresses);
				break;

			case MenuTypes.Phones:
				rows.AddRange(_user.PhoneNumbers);
				break;
		}

		_grid.Rows.Clear();
		_grid.Rows.AddRange(rows.ToArray());
	}

	public void OnAddClicked()
	{
		var selection = GetMenuSelection();

		switch (selection)
		{
			case MenuTypes.Emails:

				var emailEditor = new EmailEditor(new Email());
				emailEditor.ShowDialog();

				break;

			case MenuTypes.Addresses:

				var addressEditor = new AddressEditor(new Address());
				addressEditor.ShowDialog();

				break;

			case MenuTypes.Phones:

				var phoneEditor = new PhoneEditor(new Phone());
				phoneEditor.ShowDialog();
				break;
		}
	}
}
```

I haven't listed all the methods here, but you get the idea - a lot of repeated-ish code (switch statements), and when you want to add a new grid type you have to do the following steps:

* Add a new entry to the MenuTypes enum.
* Add the new menu item in the constructor.
* Add an implementation to the Populate method.
* Add an implementation for each action to the add, edit and delete methods.

This pretty much defines the opposite of the Open Closed Principle - the class has to be edited to add in any new functionality, and grows larger each time.  Throw in some more logic to the class, such as:

* You cannot edit Addresses, they can only be added or removed.
* You can only delete an Email if it was added less than 1 week ago.
* A Super User can do anything.
* A General User can only view items.

and you are asking for trouble, and when those requirements change or get added to, you will have to go back through all the different methods to make sure your logic holds true.

## The Solution

In a similar way to how we handled refactoring and improving the code of the `JobPostingService` in the last post, we can make a set of small steps to improve this class.

Unlike the last solution, we are going to use an abstract class as our base, rather than an Interface.  This is picked as we have some methods which are optional (see the first requirement), so we may not wish to implement all methods.

Our first step is to create our base class:

```csharp
public abstract class GridHandler
{
	public User User { get; set; }
	public abstract String Title { get; }
	public abstract IEnumerable<DataGridViewRow> Populate();

	public virtual void Add()
	{}

	public virtual void Edit(object item)
	{}

	public virtual void Delete(object item)
	{}
}
```

Note that the `Title` property and `Populate` method are abstract - you must implement these at the very least to be a `GridHandler`.
At the same time as this, we will lay our groundwork in the `UserGrid` class:

```csharp
public class UserGrid
{
	private readonly List<GridHandler> _handlers;

	public UserGrid()
	{
		_handlers = new List<GridHandler>();
		_grid = new DataGridView();
		_menu = new List<ToolStripMenuItem>();

		_menu.Add(new ToolStripMenuItem { Text = "Emails", Tag = MenuTypes.Emails });
		_menu.Add(new ToolStripMenuItem { Text = "Addresses", Tag = MenuTypes.Addresses });
		_menu.Add(new ToolStripMenuItem { Text = "Phone Numbers", Tag = MenuTypes.Phones });

	}

	public void AddHandler(GridHandler handler)
	{
		_handlers.Add(handler);
		_menu.Add(new ToolStripMenuItem { Text = handler.Title });
	}

	public void SetUser(User user)
	{
		_user = user;
		_handlers.ForEach(handler => handler.User = user);
	}

	public void Populate()
	{
		var handler = GetHandlerForSelection();

		if (handler != null)
		{
			_grid.Rows.Clear();
			_grid.Rows.AddRange(handler.Populate().ToArray());
			return;
		}

		var selection = GetMenuSelection();
		var rows = new List<DataGridViewRow>();

		switch (selection)
		{
			case MenuTypes.Emails:
				rows.AddRange(_user.EmailAddresses);
				break;

			case MenuTypes.Addresses:
				rows.AddRange(_user.Addresses);
				break;

			case MenuTypes.Phones:
				rows.AddRange(_user.PhoneNumbers);
				break;
		}

		_grid.Rows.Clear();
		_grid.Rows.AddRange(rows.ToArray());
	}
}
```

The `UserGrid` class has had a new method called `AddHandler`, which allows handlers to be added to the grid.  The `SetUser` method has been updated to also set the `User` property on all handlers, and all the `Add`, `Edit`, `Delete` and `Populate` methods have been updated to attempt to try and use a handler, and if none is found, use the existing implementation.

Our next step is to create the first `GridHandler`, which will be for Email Addresses:

```csharp
public class EmailGridHandler : GridHandler
{
	public override string Title
	{
		get { return "Email Addresses"; }
	}

	public override IEnumerable<DataGridViewRow> Populate()
	{
		return User.EmailAddresses;
	}

	public override void Add()
	{
		var email = new Email();
		var editor = new EmailEditor(email);

		editor.ShowDialog();

		User.AddEmail(email);
	}

	public override void Edit(object item)
	{
		var email = (Email)item;
		var editor = new EmailEditor(email);

		editor.ShowDialog();
	}

	public override void Delete(object item)
	{
		var email = (Email)item;
		User.RemoveEmail(email);
	}
}
```

As you can see, this class obeys the [Single Responsibility Principle][blog-solid-srp] as it only deals with how to change data from the `User` object into data and actions for the grid.

We can now update the usage of our `UserGrid` to take advantage of the new `GridHandler`:

```csharp
public class Usage : Form
{
	private UserGrid _grid;

	public Usage()
	{
		_grid = new UserGrid();
		_grid.AddHandler(new EmailGridHandler());
	}
}
```

All that remains to be done now is to go through the `UserGrid` and remove all the code relating to `Email`s.  The extraction of functionality steps can then be repeated for each of the existing grid types (`Address` and `Phone` in our case.)

Once this is done, we can go back to the `UserGrid` and remove all non-grid code, leaving us with this:

```csharp
public class UserGrid
{
	private readonly List<GridHandler> _handlers;

	public UserGrid()
	{
		_handlers = new List<GridHandler>();
	}

	public void AddHandler(GridHandler handler)
	{
		_handlers.Add(handler);
		_menu.Add(new ToolStripMenuItem { Text = handler.Title });
	}

	public void SetUser(User user)
	{
		_handlers.ForEach(handler => handler.User = user);
	}

	public void Populate()
	{
		var handler = GetHandlerForSelection();

		if (handler != null)
		{
			_grid.Rows.Clear();
			_grid.Rows.AddRange(handler.Populate().ToArray());
		}
	}

	public void OnAddClicked()
	{
		var handler = GetHandlerForSelection();

		if (handler != null)
		{
			handler.Add();
			Populate();
		}
	}
}
```

As you can see, the `UserGrid` class is now much smaller, and has no user specific logic in it.  This means we don't need to modify the class when we want to add a new grid type (it is **closed for modification**), but as adding new functionality to the grid just consists of another call to `.AddHandler(new WebsiteGridHandler());` we have made it **open for extension**.

All source code is available on my Github: [Solid.Demo Source Code][solid-demo-repo]

[blog-solid-srp]: http://andydote.co.uk/solid-principles-srp
[blog-solid-ocp]: http://andydote.co.uk/solid-principles-ocp
[blog-solid-lsp]: http://andydote.co.uk/solid-principles-lsp
[blog-solid-isp]: http://andydote.co.uk/solid-principles-isp
[blog-solid-dip]: http://andydote.co.uk/solid-principles-dip
[solid-demo-repo]: https://github.com/Pondidum/Solid.Demo
