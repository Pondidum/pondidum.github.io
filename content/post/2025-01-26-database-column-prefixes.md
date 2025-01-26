+++
title = 'Database Column Prefixes'
tags = [ "dotnet", "story", "database" ]
+++

Back in a previous company (the same place as where [Debugging GDI Handle Leaks](/2025/01/11/debugging-gdi-handles/) happened), there was an interesting convention in the database: all tables had a unique 3-letter prefix assigned to them, and all columns in that table **must** start with the prefix, which I've written about [before](/2014/03/29/using-a-micro-orm-to-decouple-your-db-access/).

For example, a `person` table would have the prefix `PEO`, and the columns would be `PEO_PersonID`, `PEO_FirstName`, `PEO_DateOfBirth`, etc.  When you wanted to create a new table, you opened the shared Excel sheet, added your table to the bottom, and made up a prefix that wasn't already in the sheet.  Even link tables (for many-to-many relationships) were not immune to this rule.

With over 100 tables in the database, finding a prefix which was vaguely related to the table's purpose became harder and harder, especially for common letters, such as `C` which off the top of my head had tables like `Companies`, `Candidates`, `Contacts`, `Categories`,`Contracts`, `ContractAttachments`, `ContractExceptions`, and a bunch of link tables to go with them all.

When asked, the DBA said that the reason the convention existed was to prevent column name conflicts when joining tables; all columns would be globally unique!  This made some level of sense for simple queries:

```sql
select  CAN_CandidateID,
        PEO_FirstName,
        PEO_LastName
from    candidates
join    people on CAN_PersonID = PEO_PersonID
where   CAN_CandidateID = @candidateID
```

The problem was that this didn't really solve the issue of columns not being ambiguous; queries often needed to join to the person table mulitple times, often via another table:

```sql
select  CAN_CandidateID,
        peo.PEO_FirstName + ' ' + peo.PEO_LastName as 'name',
        creatorperson.PEO_FirstName + ' ' + creatorperson.PEO_LastName as 'creator',
        modifierperson.PEO_FirstName + ' ' + modifierperson.PEO_LastName as 'modifier'
from    candidates
join    people peo             on CAN_PersonID = peo.PEO_PersonID
join    users creator          on creator.USR_UserID = CAN_CreatedBy
join    people creatorperson   on creatorperson.PEO_PersonID = creator.USR_PersonID
join    users modifier         on modifier.USR_UserID = CAN_ModifiedBy
join    people modifierperson  on modifierperson.PEO_PersonID = creator.USR_PersonID
where   CAN_CandidateID = @candidateID
```

A thing of pure beauty, as you can see.  Not only were all the sql statements far longer than they needed to be, and you often needed to figure out some obscure prefixes, you end up "stuttering" with things like `person.PEO_PersonID` - how many times do I need to know this is a PersonID in a single sentence?

The primary key of each table had to include the table name too.  I'm not actually convinced this is a bad idea; having a bunch of columns called `id` doesn't really make things clear when joining 6 tables.

The interesting part of this is that it's all useless; the database server we used supported table aliases (as seen above), so we could use those and drop the prefixes entirely:

```sql
select  CandidateID,
        p.FirstName + ' ' + p.LastName as 'name',
        creatorperson.FirstName + ' ' + creatorperson.LastName as 'creator',
        modifierperson.FirstName + ' ' + modifierperson.LastName as 'modifier'
from    candidates c
join    people p               on c.PersonID = p.PersonID
join    users creator          on creator.UserID = c.CreatedBy
join    people creatorperson   on creatorperson.PersonID = creator.PersonID
join    users modifier         on modifier.UserID = c.ModifiedBy
join    people modifierperson  on modifierperson.PersonID = creator.PersonID
where   CandidateID = @candidateID
```

Shorter, at least.

The table relationships didn't always help matters either; the idea of the `Person` table was that a person could exist as multiple entities in our system; they could be a user, a candidate, and a contact.  Not that this actually happened; they had unique person records for each of their entities.

The table prefixes also meant that when we wanted to use a microORM ([Dapper](https://www.learndapper.com/), which **only** maps queries into objects), we had to make every query alias every column, otherwise our property names would have to also include the prefixes, and we really didn't want the column prefixes polluting the rest of the domain!

We never got rid of this scheme in the primary database; the change wasn't worth doing.  If we had started making new tables without prefixes, we would still have had 100+ old tables with the prefixes, and no chance of fixing them as usually it involved dropping and recreating the tables.  Definitely not worth the hassle.

However, when we started creating separate databases for services which didn't rely on any data in the main database, the column prefix was not used.  In those services, everything felt a little smoother and a little less noisy.
