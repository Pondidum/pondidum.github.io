---
layout: post
title: The problems with and solutions to Repositories 
tags: design code cqrs architecture
---


Repositories are a design pattern which I have never been a huge fan of.  I can see the use of them as a good layer boundary, but too often I see them being used all over the place instead of at an infrastructure level in a code base.

A particularly prevalent version of this misuse I see is self populating collections.  These generally inherit `List<TEntity>` or `Dictionary<TID, TEntity>`, and provide a set of methods such as `.LoadByParentID(TID id)`.  The problem with this is that the collection still exposes methods such as `.Add()` and `.Remove()` - but these operations only run on the in-memory entities, and don't effect the data source itself.

## The Alternative

The technique I prefer for reads are Query objects.  These are simple classes which expose a single public method to return some data.  For example:

```csharp
public class GetDocumentsWaitingQuery : IDocumentsQuery
{
	private readonly IDataStore _dataStore;

	public GetDocumentsWaitingQuery(IDataStore datastore)
	{
		_dataStore = datastore;
	}

	public IEnumerable<Document> Execute()
	{
		using (var connection = _dataStore.Open())
		{
			return connection
				.Query<Document>(
					"select * from documents where status == @status",
					new { status = DocumentStatuses.Waiting})
				.ToList();
		}
	}
}
```

The code using this class might look something like this:

```csharp
public class DocumentProcessor
{
	private readonly IDocumentsQuery _query;

	public DocumentProcessor(IDocumentsQuery waitingDocumentsQuery)
	{
		_query = waitingDocumentsQuery;
	}

	public void Run()
	{
		foreach (var document in _query.Execute())
		{
			//some operation on document...
		}
	}
}
```

This class is almost too simple, but resembles a system's processor which I wrote.  They key here is that the `DocumentProcessor` only relies on an `IDocumentsQuery`, not a specific query.

Normal usage of the system looks like this:

```csharp
public void ProcessAll()
{
	var query = new GetDocumentsWaitingQuery(_dataStore);
	var saveCommand = new SaveDocumentCommand(_dataStore);

	var processor = new DocumentProcessor(query, saveCommand);

	processor.Run();
}
```

When the user requests a single document get reprocessed, we just substitute in a different Query:

```csharp
var query = new GetDocumentByIDQuery(_dataStore, id: 123123);
var saveCommand = new SaveDocumentCommand(_dataStore);

var processor = new DocumentProcessor(query, saveCommand);

processor.Run();
```

And finally, when the system is under test, we can pass in completely fake commands:

```csharp
[Fact]
public void When_multiple_documents_for_the_same_user()
{
	var first = new Document { .UserID = 1234, .Name = "Document One" };
	var second = new Document { .UserID = 1234, .Name = "Document Two" };

	var query = Substitute.For<IDocumentsQuery>();
	query.Execute().Returns(new[] {first, second});

	var processor = new DocumentProcessor(query, Substitute.For<ISaveDocumentCommand>());
	processor.Run();

	first.Primary.ShouldBe(true);
	second.Primary.ShouldBe(false);
}
```

This means that in the standard usage, it gets passed an instance of `GetDocumentsWaitingQuery`, but when under test gets a `Substitute.For<IDocumentsQuery>()`, and for debugging a problem with a specific document, it gets given `new GetSingleDocumentQuery(id: 234234)` for example.

## Commands

What about saving?  Well it's pretty much the same story:

```csharp
public class SaveDocumentCommand
{
	private readonly IDataStore datastore;

	public SaveDocumentCommand(IDataStore datastore)
	{
		_dataStore = datastore
	}

	public void Execute(Document document)
	{
		using (var connection = _dataStore.Open())
		{
			connection.Execute("update documents set status = @status where id = @id", document);
		}
	}
}
```

Obviously the sql in the save command would be a bit more complete...

## But Repositories...

Well yes, you can create methods on your repositories to do all of this, like so:

```csharp
public IDocumentRepository
{
	public void SaveDocument(Document document) { /* ... */ }
	public IEnumerable<Document> GetDocumentsWaiting() { /* ... */ }
}
```

But now your classes utilising this repository are tied to the methods it implements - you cannot just swap out the workings of `.GetDocumentsWaiting` for a single document query any more.

This is why I like to use Command and Query objects - the not only provide good encapsulation (all your sql is contained within), but they also provide a large level of flexibility in your system, and make it very easy to test to boot too!
