---
layout: post
title: "Model View Presenters: Presenter to View Communication"
tags: design code net

---

Table of Contents:
------------------
* [Introduction][1]
* **Presenter to View Communication**
* [View to Presenter Communication][2]
* [Composite Views][3]
* Presenter / Application communication
* ...


Presenter to View Communication
-------------------------------

There are two styles utilised for populating the View with data from the Presenter and Model that I have used.  The only difference between them is how tightly coupled you mind your View being to the Model.  For the example of this, we will have the following as our Model:

{% highlight c# %}
public class Person
{
	public int ID { get; private set; }
	public int Age { get; set; }
	public String FirstName { get; set; }
	public String LastName { get; set; }
	Public Genders Gender { get; set; }
}
{% endhighlight %}

Method 1: Using the Model
---------------

Now our View code:

{% highlight c# %}
public interface IEmployeesView
{
	void ClearList();
	void PopulateList(IEnumerable<Person> people);
}
{% endhighlight %}

And finally the Presenter:

{% highlight c# %}
public class IEmployeesPresenter
{
	public void Display()
	{
		_view.ClearList();
		_view.PopulateList(_model.AllEmployees);
	}
}
{% endhighlight %}

This method of population produces a link between the Model and the View; the Person object used as a parameter in `PopulateList`.

The advantage of this is that the concrete implementation of the IEmployeesView can decide on what to display in its list of people, picking from any or all of the properties on the `Person`.

There are two disadvantages of this method.  The first is that there is nothing stopping the View from calling methods on the `Person`, which makes it easy for lazy code to slip in.  The second is that if the model were to change from a `List<Person>` to a `List<Dog>` for instance, not only would the Model and the Presenter need to change, but so the View would too.


Method 2: Using Generic Types
-----------------------------

The other method population relies on using `Tuple<...>`, `KeyValuePair<,>` and custom classes and structs:

Now our View code:

{% highlight c# %}
public interface IEmployeesView
{
	void ClearList();
	void PopulateList(IEnumerable<Tuple<int, String> names);
}
{% endhighlight %}

And finally the Presenter:

{% highlight c# %}
public class IEmployeesPresenter
{
	public void Display()
	{
		var names = _model.AllEmployees.Select(x => new Tuple<int, String>(x.ID, x.FirstName + " " + x.LastName));

		_view.ClearList();
		_view.PopulateList(names);
	}
}
{% endhighlight %}

The advantages of this method of population is that the Model is free to change without needing to update the View, and the View has no decisions to make on what to display.  It also prevents the View from calling any extra methods on the `Person`, as it does not have a reference to it.

The down sides to this method, are that you loose strong typing, and discoverability - It is quite obvious what a `Person` is but what a `Tuple<int, String>` is less obvious.

[1]: /model-view-presenter-introduction
[2]: /model-view-presenters-view-to-presenter-communication
[3]: /model-view-presenters-composite-views