---
layout: post
title: Don't Let The Database Dictate Your Design
tags: design code postgres sql architecture
---

I have been thinking recently about how the database can influence our design decisions, and perhaps makes them harder than they need to be in some cases.  An example of this is the design of a system which stores data about people, specifically for this, their email addresses.  A cut down version of the structure is this:

{% highlight sql %}
table people
id serial primary key
firstname varchar(50)
lastname varchar(50)

table emails
id serial primary key
person_id int => people.id
address varchar(100)
type int
{% endhighlight %}

Which is represented in code like so:

{% highlight c# %}
public class Person
{
	public int ID { get; private set; }
	public string FirstName { get; set; }
	public string LastName { get; set; }

	public List<Email> Emails { get; private set; }

	public Person()
	{
		Emails = new List<Email>();
	}
}

public class Email
{
	public int ID { get; private set; }
	public int PersonID { get; set; }
	public string Address { get; set; }
	public EmailTypes Type { get; set; }
}
{% endhighlight %}

While this works, it is heavily influenced by the storage technology.  Email addresses by definition are unique already, so why do we need a primary key column? They are also associated with exactly one person, so the `person_id` column is only here to facilitate that.  Why not get rid of the emails table completely, and store the person's email addresses in a single column in the person table?  This could be done with a simple csv, but it would be more fore-thinking to use json, so we can associate a little more data with each email address.

So before we get on to putting data in one column, what reasons we have to keep emails as a separate table?

* So they can be queried by ID.
* So we can put a constraint across `[person_id, address]` and `[person_id, type]`.
* So all emails of a given type can be found.
* So a person can be looked up by email.
* So we can attach rules to them.

The first three can be answered easily: you never query for an email address by its primary key, only by the address itself.  The constraints are really just a safety net, and a nice to have at best - the collection which manages emails is well tested, implements all business rules properly, and everything which deals with emails uses it.  Getting all emails of a particular type is a legitamate requirement, but can be gotten around in several ways: selecting the entire email column in a sql query, and doing additional filtering client side for the specific email types, or if you are using a database which supports json querying (such as postgres), using that to narrow the results down.

The final point is the most interesting, as it could be resolved with a few different designs.  The current design has one additional table:

{% highlight sql %}
table rules
id serial primary key
person_id int => people.id
target_type int --e.g 1=email, 2=phone, 3=address etc
target_id int
active bool
{% endhighlight %}

And the `Person` object has a method like this:

{% highlight c# %}
public bool HasRuleInForce(Entity target)
{
	return Rules
		.Where(rule => rule.TargetType == target.Type)
		.Where(rule => rule.TargetID == target.ID)
		.Where(rule => rule.Active)
		.Any();
}
{% endhighlight %}

While this works, the design has a few problems:

* There is no foreign keying of `rules.target_id` available
* So you have to remember to delete rules when deleting any entity
* You have to remember if an entity is valid for rules to be attached to
* If normalisation was your argument for an `emails` table, explain this table relationship...

There are two solutions to this problem:

The first is to change the rules table to just have a `target` column, and put the unique data in there e.g. a rule for an email would have the email address in the `target` column, a rule for a phone number would have the actual phone number in the `target` column.  While this works, it doesn't really improve the design of the system; we still have the existing joins and "remember to also" problems of before.

The second solution is to remove the `rules` table entirely and implement rules as small collections on each target entity, and make the `person.Rules` property a readonly aggregate.  This has a few advantages: each entity explicitly has a rule collection if applicable, and we no longer need to remember to check another collection for updates/deletes.

The implementation of a `.Rules` property on each entity is trivial - just a standard list property:

{% highlight c# %}
public class Email
{
	public int ID { get; private set; }
	public int PersonID { get; set; }
	public string Address { get; set; }
	public EmailTypes Type { get; set; }
	public List<Rule> Rules { get; set; }
}
{% endhighlight %}

As we don't wish to repeat the logic on each collection of rules, we can add an extension method for checking if rules are in force:

{% highlight c# %}
public static class RulesExtensions
{
	public static bool HasRuleInForce(this IEnumerable<Rule> self)
	{
		return self.Any(rule => rule.Active);
	}
}
{% endhighlight %}

And finally on the `Person` object itself, we can make a simple aggregate property for all child entity's rules:

{% highlight c# %}
public IEnumerable<Rule> Rules
{
	get
	{
		var all = new[]
		{
			Emails.SelectMany(e => e.Rules),
			Phones.SelectMany(p => p.Rules),
		};

		return all.SelectMany(r => r);
	}
}
{% endhighlight %}

Personally I prefer the 2nd form of this, as it makes domain modelling a lot more straight forward - however like all things, you should consider all your requirements carefully - and don't let the database (sql or nosql variety) dictate your model.