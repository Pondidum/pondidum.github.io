+++
date = '2014-02-18T00:00:00Z'
tags = ['design', 'c#', 'solid']
title = 'SOLID Principles - SRP'

+++

## Single Responsibility Principle

[Single Responsibility][blog-solid-srp] | [Open Closed][blog-solid-ocp] | [Liskov Substitution][blog-solid-lsp] | [Interface Segregation][blog-solid-isp] | [Dependency Inversion][blog-solid-dip]

SRP (Single Responsibility Principle) is something I hear a lot of developers agree is a good thing, but when I read their code, they violate it without realising, or don't see the use in their particular case.

A particularly prominent example I find in our code bases is Permissioning and Caching.  These two requirements can often slip into classes slowly - especially if requirements are not clear, or change as the task progresses.  A slightly contrived example is this:

```csharp
public class JobPostingService
{
	private static readonly TimeSpan Timeout = new TimeSpan(0, 10, 0);

	private readonly JobWebService _jobService;

	private List<Job> _jobs;
	private DateTime _lastLoaded;

	public JobPostingService()
	{
		_jobService = new JobWebService();
		_lastLoaded = DateTime.MinValue;
	}

	public IEnumerable<Job> GetCurrentJobs()
	{
		if (_lastLoaded - DateTime.Now > Timeout)
		{
			_jobs = _jobService.GetLiveJobs().ToList();
			_lastLoaded = DateTime.Now;
		}

		return _jobs;
	}

	public void PostToFreeBoards(Job job)
	{
		var jobs = GetCurrentJobs();

		if (jobs.Any(j => j.ID == job.ID))
			return;

		_jobService.Post(job, Boards.FreeBoard1 | Boards.FreeBoard2);
	}

	public void PostToAllBoards(Job job)
	{
		var jobs = GetCurrentJobs();

		if (jobs.Any(j => j.ID == job.ID))
			return;

		_jobService.Post(job, Boards.PaidBoard1 | Boards.PaidBoard2);
	}
}
```

This class is fairly small, but it is already showing the symptoms of doing too many things; it is dealing with caching, as well as posting jobs.  While this is not a major problem at the moment, it is also easier to nip the problem in the bud - before a load of new requirements/changes arrive and complicate things.

## The Solution

We start off by changing our class to take it's dependencies in via constructor parameters (Dependency Injection, the 'D' in SOLID):

```csharp
public JobPostingService(JobWebService jobService)
{
	_jobService = jobService;
	_lastLoaded = DateTime.MinValue;
}
```

So the usage of the `JobPostingService` goes from this:

```csharp
var poster = new JobPostingService();
```

To this:

```csharp
var poster = new JobPostingService(new JobWebService());
```

Next, we take the `JobWebService` class and extract & implement an interface of it's methods:

```csharp
public interface IJobService
{
	IEnumerable<Job> GetLiveJobs();
	bool Post(Job job, Boards boards);
}

public class JobWebService : IJobService
{
	//...
}
```

And finally, create a new class which only deals with caching the results of a JobService, by wrapping calls to another instance:

```csharp
public class CachedJobService : IJobService
{
	private List<Job> _jobs;
	private DateTime _lastLoaded;
	private readonly TimeSpan _timeout;
	private readonly IJobService _other;

	public CachedJobService(IJobService otherService)
		: this(otherService, new TimeSpan(0, 10, 0))
	{
	}

	public CachedJobService(IJobService otherService, TimeSpan timeout)
	{
		_other = otherService;
		_timeout = timeout;
		_lastLoaded = DateTime.MinValue;
	}

	public IEnumerable<Job> GetLiveJobs()
	{
		if (_lastLoaded - DateTime.Now > _timeout)
		{
			_jobs = _other.GetLiveJobs().ToList();
			_lastLoaded = DateTime.Now;
		}

		return _jobs;
	}

	public bool Post(Job job, Boards boards)
	{
		return _other.Post(job, boards);
	}
}
```

This class passes all `Post()` calls to the other implementation, but caches the results of calls to `GetLiveJobs()`, and we have added a time-out as an optional constructor parameter.  This wrapping calls to another implementation is called [The Decorator Pattern][pattern-decorator].

As the JobPostingService class no longer has to cache the results of calls to `JobService` itself, we can delete all the caching related code:

```csharp
public class JobPostingService
{
	private readonly IJobService _jobService;

	public JobPostingService(IJobService jobService)
	{
		_jobService = jobService;
	}

	public IEnumerable<Job> GetCurrentJobs()
	{
		return _jobService.GetLiveJobs();
	}

	public void PostToFreeBoards(Job job)
	{
		var jobs = GetCurrentJobs();

		if (jobs.Any(j => j.ID == job.ID))
			return;

		_jobService.Post(job, Boards.FreeBoard1 | Boards.FreeBoard2);
	}

	public void PostToAllBoards(Job job)
	{
		var jobs = GetCurrentJobs();

		if (jobs.Any(j => j.ID == job.ID))
			return;

		_jobService.Post(job, Boards.PaidBoard1 | Boards.PaidBoard2);
	}
}
```

And our usage changes again, from this:

```csharp
var poster = new JobPostingService(new JobWebService());
```

To this:

```csharp
var webService = new CachedJobService(new JobWebService());
var poster = new JobPostingService(webService);
```

We have now successfully extracted all the various pieces of functionality into separate classes, which has gained us the ability to test individual features (caching can be tested with a fake `IJobService` and checked to see when calls go through to the service), and the ability to adapt more easily to new requirements.  Talking of which...

> New Requirement:  The third party webservice is not always available, allow use of a fallback webservice.

Now you could go and modify the `JobPostingService` class to have a second webservice parameter:

```csharp
var primaryService = new CachedJobService(new JobWebService());
var secondaryService = new CachedJobService(new BackupWebService());

var poster = new JobPostingService(primaryService, secondaryService);
```

But what happens when a third service is added? and a fourth? Surely there is another way?

As luck would have it, we can use the `IJobService` interface to create a single class which handles all the logic for switching between the two services:

```csharp
public class FailoverJobService : IJobService
{
	private readonly List<IJobService> _services;

	public FailoverJobService(params IJobService[] services)
	{
		_services = services.ToList();
	}

	public IEnumerable<Job> GetLiveJobs()
	{
		return _services.SelectMany(s => s.GetLiveJobs());
	}

	public bool Post(Job job, Boards boards)
	{
		return _services.Any(service => service.Post(job, boards));
	}
}
```

This class takes in a number of `IJobService`s and will try each one in turn to post jobs, and when listing jobs, gets the results from all services.  In the same manner as the `CachedJobService`, we have a single class which can easily be tested without effecting any of the other functionality.

The really interesting point comes when we decide when to use caching? do you cache each service passed to the `FailoverJobService`:

```csharp
var primaryService = new CachedJobService(new JobWebService());
var secondaryService = new CachedJobService(new BackupWebService());

var failover = new FailoverJobService(primaryService, secondaryService);

var poster = new JobPostingService(failover);
```

Or do you cache the `FailoverJobService` itself:

```csharp
var primaryService = new JobWebService();
var secondaryService = new BackupWebService();

var failover = new CachedJobService(new FailoverJobService(primaryService, secondaryService));

var poster = new JobPostingService(failover);
```

Or both?

Hopefully this article has explained 1/5th (maybe a little more, we did do Dependency Injection after all!) of the SOLID principles, and how it can be useful to keep your code as small and modular as possible.

All source code is available on my Github: [Solid.Demo Source Code][solid-demo-repo]

[blog-solid-srp]: http://andydote.co.uk/solid-principles-srp
[blog-solid-ocp]: http://andydote.co.uk/solid-principles-ocp
[blog-solid-lsp]: http://andydote.co.uk/solid-principles-lsp
[blog-solid-isp]: http://andydote.co.uk/solid-principles-isp
[blog-solid-dip]: http://andydote.co.uk/solid-principles-dip
[solid-demo-repo]: https://github.com/Pondidum/Solid.Demo
[pattern-decorator]: http://en.wikipedia.org/wiki/Decorator_pattern
