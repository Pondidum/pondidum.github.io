# Merging and Unmerging Records with Event Sourcing

Merging records in a system is a common requirement, and one which tends to be discovered once the system has been in use for a while.  Typical reasons for record merging are: accidental creations (such as a user registering twice for a website, with differing email addresses, or an automated system creating a new record rather than updating an existing one), and bulk data imports from other systems.

How to detect that two (or more) records can be merged is a large subject in of itself, as it is almost entirely very domain specific.  An example which is not clear cut is user accounts; do you consider two accounts with the same emails listed as the same account, or is it a couple who share an email address, but want different accounts?

The technical details on *how* to merge though are discussable however, although as with all things development, there are some trade-offs to choose among.

When merging, we have two methods which can be used: we can either write a combination of data from both source records to a new record, or copy the data from one record to the other.  In both cases the old records will need either deleting, or marking as "dead".

This so far is fairly straight forward, but it gets much more interesting once the logical next business requirement comes along

> I need to be able to undo a merge of two records

This makes sense - but is a fairly difficult operation, once you consider the following:

* undo the merge itself
* do the old records exist still?
* what about changes since the merge?
	* and if we are keeping then, applied to both or one record?

This is where the relational system falls down, as unless you logged old and new values for every value which changed during the merge, how do you know what to undo?

## Enter Event Sourcing

When it comes to merging two records with event sourcing, we start off with a choice when constructing the merge code:

* Write one `RecordMerged` event, with all the changes contained within
* Write multiple `xxxxMerged` event, one for each change required.

This decision is down to personal taste:  One event is easier to handle, but means it changes whenever we need to add/modify/remove values the merge handles.  Multiple event has better Single Responsibility, but means our undo-merge code will have to watch for multiple events, which need specifying somehow.

Personally I opt for the multiple events system, and mark the events with an interface such as `IMergeEvent` and build a projection/stream processor to look for all events implementing it.


## Undoing the Merge

We have the same choice to make when undoing a merge as we did when making the merge - do we create one single undo event, or multiple?

If we start with two source aggregates:

* AggregateCreated id: "source_1"
* EmailAdded email: "source.1@example.com"
* AggregatesMerged otherId: "source_2"

* AggregateCreated id: "source_2"
* EmailAdded email: "source.2@example.com"
* AggregatesMerged otherId: "Source 1"

And merge them:

* AggregateCreatedByMerge id: "merged_1", from: [ "source_1", "source_2" ], ...
* EmailChanged email: "merged.1@example.com"

And "undo" the merge:

* AggregateCreated id: "source_1"
* EmailAdded email: "source.1@example.com"
* AggregatesMerged otherId: "source_2"
* **EmailChanged email: "merged.1@example.com"**

* AggregateCreated id: "source_2"
* EmailAdded email: "source.2@example.com"
* AggregatesMerged otherId: "Source 1"
* **EmailChanged email: "merged.1@example.com"**

Is this valid?  The only real answer here is "it depends", as this is entirely domain specific.

When undoing a merge, frequently the best option is to give the user the choice of what to do:

* Apply all subsequent events to "source_1"
* Apply all subsequent events to "source_2"
* Apply all subsequent events to both
* Choose which events to apply to each aggregate

The last option sounds easy, but also entails validity, what if `EventY` can only happen after `EventX`, but the user selects to apply `EventX` to "source_1" and `EventY` to "source_2"?
