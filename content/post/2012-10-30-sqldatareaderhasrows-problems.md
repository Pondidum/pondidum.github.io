---
date: "2012-10-30T00:00:00Z"
tags: bug sql
title: SqlDataReader.HasRows Problems
---

For the last 6 years or so at work, we have had an intermittent bug.  In this case, intermittent means around once in 6 months or so.  A little background to the problem first:

Our data access is done via what was originally Microsoft's SQLHelper class, passing in a stored procedure (and parameters), and our entities use the reader to load all their properties.  Pretty straight forward stuff.

The problemis, on the live system, every few months a sproc will stop returning results, for no apparent reason.  Calling the sproc from Sql Management Studio works fine.  We have tried many different fixes: re-applying the sproc, calling the sproc from a different database login, re-pointing to the dev or test systems.  None of it makes any difference, and then as suddenly as it stopped working, it starts working again.

A few days ago, I was attempting to track down some inconsistent search results, this time based around an fts index.  Now this index is pretty large (at least, in my books it is) at around 1.5 million rows, and the column itself being a good few thousand words on average.

The code used for this problem boils down to the following

Sproc "ftsSearch":

```sql
	Select	id
	from	ftsTable
	where	contains(@query, searchColumn)
```

Reader Class:

```csharp
public class FtsSearch : List<int>
{
	public void Search(String input)
	{
		Clear();

		var param = new SqlParameter("@query", SqlDbType.VarChar);
		param.Value = input;

		using (var reader = SqlHelper.ExecuteReader(DbConnection, "ftsSearch", param))
		{
			if (reader.HasRows)
			{
				while (reader.Read())
				{
					Add(reader.GetInt32(0));
				}
			}
		}
	}
}
```

Calling this function while the error is occurring, yields the following results:

	Query:						Results:
	"Project Management"		20,000
	"Project Manager"			15,000
	"Project Manage*"			0

The first two queries are fine, the last however I would expect to bring back between 20,000 and ~35,000 results, and when we ran the sql from Management Studio, it brought back 29,000 results.

Now when debugging the function, we double checked everything was being called correctly - correct DB, correct sproc, correct login, correct (parsed) parameter.

Inspecting HasRows returns False.  So we forced a call to Read() anyway, just to see what happened.  An what do you know? Results, all there.

The reason that HasRows was returning false was that the sproc was triggering sql server to also send back a warning - in this case one about too many results (afraid I have lost the exact error code).  Sadly this behaviour does not seem to be documented anywhere.
