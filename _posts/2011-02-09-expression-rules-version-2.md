---
layout: post
title: Expression Rules, Version 2
tags: design, code, net
permalink: expression-rules-version-2
---

Recently I have written a rules engine for a very large menu system in an application I work on.  Many of the rules apply many items, so I didn't wish to have to express the same rule many times.  To avoid this, the rule engine DSL was born:

{% highlight c# %}
Concerns.When(item => /* rule of some sort */)
		.AppliesToAll()
		.Except(MenuItems.ToggleHidden, MenuItems.Refresh)
{% endhighlight %}

And rules are rolled together, so a specific menu item must have all of its rules evaluating to true to be displayed.

The problem arose when an item was displaying when it shouldn't (or vice versa).  Debugging with rules specified like this was a pain, and when I saw the article about [ExpressionRules][2] by [Daniel Wertheim][1], I thought it would help solve my problem.  He replaces Lambda conditions with a class and implicit operator, allowing code to be changed from something like this:

{% highlight c# %}
var bonusCustomers = _customers.Where(c =>
		(c.NumOfYearsAsMember == 0 && c.CashSpent >= 3000) ||
		(c.NumOfYearsAsMember > 0 && (c.CashSpent / c.NumOfYearsAsMember) >= 5000));
{% endhighlight %}

To something like this:

{% highlight c# %}
var bonusCustomers = _customers.Where(new IsBonusCustomer());
{% endhighlight %}

He does this using a base class and then inheriting from it to create the rule:

{% highlight c# %}
public class IsBonusCustomer : ExpressionRule<Customer>, IIsBonusCustomer
{
	public IsBonusCustomer()
		: base(c =>
				(c.NumOfYearsAsMember == 0 && c.CashSpent >= 3000) ||
				(c.NumOfYearsAsMember > 0 && (c.CashSpent / c.NumOfYearsAsMember) >= 5000))
	{
	}
}
{% endhighlight %}

I took his base class and modified it to this:

{% highlight c# %}
public abstract class ExpressionRule<T> where T : class
{
	protected abstract bool Rule(T item);

	public static implicit operator Func<T, bool>(ExpressionRule<T> item)
	{
		return item.Rule;
	}

	public bool Evaluate(T item)
	{
		return Rule(item);
	}
}
{% endhighlight %}

This means the IsBonusCustomer now becomes this:

{% highlight c# %}
public class IsBonusCustomer : ExpressionRule<Customer>
{
	protected override bool Rule(Customer customer)
	{
		return (c.NumOfYearsAsMember == 0 && c.CashSpent >= 3000) ||
			   (c.NumOfYearsAsMember > 0 && (c.CashSpent / c.NumOfYearsAsMember) >= 5000)
	}
}
{% endhighlight %}

Not only do we still have the readability of the first version, but a full function that can have logging added to it, and easier debugging.

[1]: http://daniel.wertheim.se/
[2]: http://daniel.wertheim.se/2011/02/07/c-clean-up-your-linq-queries-and-lambda-expressions/
