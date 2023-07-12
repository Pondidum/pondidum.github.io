+++
date = '2010-10-20T00:00:00Z'
tags = ['controls', 'bug', 'c#']
title = 'Databinding to a DataGridView - The order of columns'

+++

A while ago I was writing a small history grid in one of our applications at work.  It has a single `HistoryItem` object, which is fairly straightforward, something like this:

```csharp
Class HistoryItem
{
	public int ID { get{ return _id; } }
	public DateTime CreateDate { get { return _createDate; } }
	public String Creator { get { return _creatorName; } }
	public String Note { get { return _note; } }
}
```

This was populated into a `List<HistoryItem>` and bound to the `DataGridView` directly:

```csharp
	dgvHistory.DataSource = ScreenEntity.History.ToList();
```

This exposes something interesting about how the DataGridView picks column order: It's not done by Alphabetical Order; it is done by Definition Order.  So the order in which the properties in the class are defined is the order that the grid view will display. Usually.

When the piece of software was deployed (in house software, to be used by about 10 people), one user requested the order of the columns be changed.  She didn't like the fact that the order was this for her: `Note, ID, CreateDate, Creator`.

After checking my copy of the software and several other users' copies, it turned out the order was only different on her machine.  She could login to another machine and it would be fine.  At the time I never got to the bottom of why it was setting the wrong order, but fixed it by manually specifying the column order after binding.

Yesterday however I was reading an article by [Abhishek Sur on the Hidden Facts of C# Structures in terms of MSIL][1] and noticed this piece of information:

> DemoClass is declared as auto...Auto allows the loader to change the layout of the class which it sees fit. That means the order of the members will not be kept intact while the object is created. It is also going to ignore any layout information for the class mentioned explicitly.

Now while I am unable to reproduce this problem currently as I am not near work, I do wonder if the reason column orders were fine on most machines was because the CLR was keeping the properties in definition order, with the exception of one machine, where for whatever reason it was reordering the properties.

If this problem arises again then I will have a go at fixing it by changing to a Structure (which by default are declared as Sequential in IL) and see if that fixes the problem.

[1]: http://www.abhisheksur.com/2010/10/hidden-facts-on-c-constructor-in.html
