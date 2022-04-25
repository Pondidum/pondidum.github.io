---
date: "2014-05-04T00:00:00Z"
tags: design c# domain ddd
title: Writing Rich Domain Models
---

The term Rich Domain Model is used to describe a domain model which really shows you how you should be using and manipulating the model, rather than letting you do anything with it.  It is the opposite of an Anaemic Domain Model, which provides a very low abstraction over the data storage (generally), but with little to no enforcing of rules.

## The Anaemic Domain Model

To take the standard model of a person who has addresses and phone numbers etc seems a little contrite, so lets run through an example using timesheets (bear in mind I don't know what really goes into a timesheet system, this just seems reasonable).  The current model looks something like the following:

```csharp
public class TimeSheet : DbEntity
{
	public DateTime WeekDate { get; set; }
	public TimeSheetStates State { get; set; }
	public TimeSheetLineCollection Lines { get; set; }

	//...
}

public class TimeSheetLineCollection : DbEntityCollection<TimeSheetLine>
{
}

public class TimeSheetLine : DbEntity
{
	public DateTime Day { get; set;}
	public LineTypes LineType { get; set; }
	public decimal HourlyRate { get; set; }
	public decimal Hours { get; set; }
}

public enum TimeSheetStates
{
	New,
	Saved,
	Submitted,
	Approved,
	Rejected
}

public enum LineTypes
{
	Normal,
	Holiday,
	Sick
}
```

The first problem with this model is that the domain entities are inheriting directly from a `DbEntity` which is coupling our logic directly to our data access, which amongst other things is a violation of [SRP][blog-solid-srp].  Putting this aside for the time being, the next issue is that the domain model lets you do anything with the objects and collections.

The model implies that there are rules governing its usage somewhere, but gives no hint as to what these rules are, or where they are located.  Rules such as 'only allow hours to be entered in increments of half an hour' and 'no more than 5 lines in a given week' really should be in the domain model itself, as a Rich Domain Model should not allow itself to get into an invalid state.

The model also is leaking what kind of data store it is built on - after all, if you had an Event Sourcing pattern for storage, a `Delete` operation on the `TimeSheetLineCollection` would not make a lot of sense.

## The Rich Domain Model

A better version of this model is to make all the behaviour explicit, rather than just exposing the collections for external modification:

```csharp
public class TimeSheet
{
	public DateTime WeekDate { get; private set; }
	public TimeSheetStates State { get; private set; }
	public IEnumerable<TimeSheetLine> Lines { get { return _lines; } }

	private readonly List<TimeSheetLine> _lines;
	private readonly TimeSheetRules _rules;

	public TimeSheet(TimeSheetRules rules, DateTime weekDate)
	{
		_lines = new List<TimeSheetLine>();
		_rules = rules;
		WeekDate = weekDate
	}

	public void AddLine(DayOfWeek day, LineTypes lineType, decimal hours, decimal hourlyRate)
	{
		var line = new TimeSheetLine {
			Day = WeekDate.AddDays(day),
			LineType = lineType,
			Hours = hours,
			HourlyRate = hourlyRate
		};

		_rules.ValidateAdd(Lines, line);	//throws a descriptive error message if you can't do add.
		_lines.Add(line);
	}

}
```

The Rich model does a number of interesting things.  The first is that all the properties of the `TimeSheet` class are now `private set`.  This allows us to enforce rules on when and how they get set.  For example, the `WeekDate` property value gets passed in via the constructor, as our domain says that for a week to be valid it must have a weekdate.

The major improvement is in adding lines to the `TimeSheet`.  In the Anaemic version of the model, you could have just created a `TimeSheetLine` object and set the `Day` property to an arbitrary date, rather than one in the given week's range.  The Rich model forces the caller to pass in a `DayOfWeek` to the function, which ensures that a valid datetime will get stored for the line.  The `AddLine` method also calls `_rules.ValidateAdd()` which gives us a central place for putting rules on line actions.

Now that the user has been able to fill out all the lines in their timesheet, the next likely action they want to perform is to submit it for authorization.  We can do this by adding the following method:

```csharp
public void SubmitForApproval(User approver)
{
	_rules.ValidateTimeSheetIsComplete(this);

	approver.AddWaitingTimeSheet(this);
	State = TimeSheetStates.Submitted;
}
```

Note this method only validates if the timesheet is complete enough to be approved - validation for whether the approver can actually approve this timesheet is held within the `apperover.AddWaitingTimeSheet` method.

The next thing to consider is when the approver rejects the timesheet because the user filled out the wrong weekdate.  Rather than just exposing Weekdate to be publicly setable, we can capture the intent of the adjustment with a set of methods:

```csharp
public void UserEnteredIncorrectWeek(DateTime newDate)
{
	var delta = WeekDate - newDate;

	WeekDate = newDate;
	_lines.ForEach(line => line.Day = line.Day.AddDays(-delta));
}
```

Note how the method is named to capture the reason for the change.  Although we are not actively storing the reason, if we were using an EventStream for the backing store, or maintaining a separate log of changes we would now have a reason as to why the change was made.  This helps guide UI elements - rather then just having an "Edit Week Date" button, there could be a UI element which says "Change Incorrect Week" or similar.

The function also has some logic baked into it - each of the `TimeSheetLine`s needs its `Day` property re-calculating.

Hopefully this helps demonstrate why Rich Domain Models are better solutions to complex domain problems than Anaemic Domain Models are.

For a really good video on this subject, check out Jimmy Bogard's [Crafting Wicked Domain Models][wicked-domains] talk.

[blog-solid-srp]: http://andydote.co.uk/solid-principles-srp
[wicked-domains]: http://vimeo.com/43598193
