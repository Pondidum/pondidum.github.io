---
layout: post
title: Database Integrations for MicroServices
tags: code net microservices integration eventsourcing
---

This is a follow up post after seeing [Michal Franc][michal-franc-ndc]'s NDC talk on migrating from Monolithic architectures.

One point raised was that Database Integration points are a terrible idea - and I wholeheartedly agree.  However, there can be a number of situations where a Database Integration is the best or only way to achieve the end goal.  This can be either technical; say a tool does not support API querying (looking at you SSRS), or cultural; the other team either don't have the willingness, time, or power to learn how to query an API.

One common situation is a reporting team, who either cannot query an API (e.g. they are stuck using SSRS), or don't want/have time to learn how to query an API.

There are two ways which can make a Database Integration an altogether less painful prospect, both with a common starting point: A separate login to the Database, with only readonly access to a very small set of tables and views.

Views can be used to create a representation of the service's data in a manner which makes sense to external systems, for example de-normalising tables, or converting integer based enumerations into their string counterparts.

Tables can be used to expose a transformed version of the service's data, for example a readmodel from an event stream.

## Event Sourcing source data

For example, one of our services uses Event Sourcing.  It uses projections to construct readmodels as events are stored (we use the [Ledger][ledger] library, and a SqlServer backend for this.)  To provide a Database Integeration point, we have a second set of projections which populate a set of tables specifically for external querying.

If the following event was committed to the store:
```
{ "eventType": "phoneNumberAdded", "aggregateID": 231231, "number": "01230 232323", "type": "home" }
```

The readmodel table, which is just two columns: `id:int` and `json:varchar(max)`, would get updated to look like this:
```
id      | json
----------------------------------------------------------
231231  | {
            "id": 231231,
            "name": "Andy Dote",
            "phones": [
              { "type": "mobile", "number": "0712345646" },
              { "type": "home", "number": "01230 232323" }
            ]
          }
```

The external integration table, which is a denormalised view of the data would get updated to look like this:
```
id      | name      | home_phone    | mobile_phone
----------------------------------------------------------
231231  | Andy Dote | 01230 232 323 | 07123 456 456
```

### Non-SQL Systems

While I have not needed to implement this yet, there is a plan for how to do it:  a simple regular job which will pull the data from the service's main store, transform it, and insert it into the SQL store.

### Relational Systems

A relational system can be done in a number of ways:
* In the same manner as the Non-SQL system: with a periodical job
* In a similar manner to the Event Sourced system: Updating a second table at the same time as the primary tables
* Using SQL triggers: on insert, add a row to the integration table etc.

I wouldn't recommend the 3rd option, as you will start ending up with more and more logic living in larger and larger triggers.
The important point on all these methods is that the Integration tables are separate from the main tables: you do not want to expose your internal implementation to external consumers.



[ssrs-sources]: https://msdn.microsoft.com/en-us/library/ms159219.aspx
[ledger]: https://www.nuget.org/packages/ledger
[michal-franc-ndc]: https://twitter.com/francmichal
