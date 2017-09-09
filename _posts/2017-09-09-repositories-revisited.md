---
layout: post
title: Repositories Revisited (and why CQRS is better)
tags: design code cqrs architecture
---

**TLDR:** I still don't like Repositories!

Recently I had a discussion with a commenter on my [The problems with, and solutions to Repositories](2015/03/28/problems-with-and-solutions-to-repositories/) post, and felt it was worth expanding on how I don't use repositories.

My applications tend to use the mediator pattern to keep things decoupled (using the [Mediatr](https://github.com/jbogard/MediatR) library), and this means that I end up with "handler" classes which process messages; they load something from storage, call domain methods, and then write it back to storage, possibly returning some or all the data.

For example you could implement a handler to update the tags on a toggle class like so:

```csharp
public class UpdateToggleTagsHandler : IAsyncRequestHandler<UpdateToggleTagsRequest, UpdateToggleTagsResponse>
{
    private readonly GetToggleQuery _getToggle;
    private readonly SaveToggleCommand _saveToggle;

    public UpdateToggleTagsHandler(GetToggleQuery getToggle, SaveToggleCommand saveToggle)
    {
        _getToggle = getToggle;
        _saveToggle = saveToggle;
    }

    public async Task<UpdateToggleTagsResponse> Handle(UpdateToggleTagsRequest message)
    {
        var toggle = await _getToggle.Execute(message.ToggleID);

        toggle.AddTag(message.Tag);

        await _saveToggle(toggle);

        return new UpdateToggleTagsResponse
        {
            Tags = toggle.Tags.ToArray()
        };
    }
}
```

Note how we use constructor injection to get a single command and a single query, and that  the business logic is contained within the `Toggle` class itself, not the `Handler`.

By depending on commands and queries rather than using a repository, we can see at a glance what the `UpdateToggleTagsHandler` requires in the way of data, rather than having to pick through the code and figure out which of 20 methods on a repository is actually being called.

The actual domain classes (in this case, the `Toggle` class) know nothing of storage concerns.  As I use EventSourcing a lot, the domain classes just need a few methods to facilitate storage: applying events, fetching pending events, and clearing pending events.  For non EventSourced classes, I tend to use the Memento pattern: each class implements two methods, one to load from a plain object, one to write to the same plain object.

If your handler starts needing many commands or many queries passed in, it's a pretty good indication that your design has a weakness which will probably need refactoring.  This is harder to notice when using repositories as you might still only have a single constructor parameter, but be calling tens of methods on it.

Hopefully this provides a bit more reasoning behind my dislike of repositories, and how I try to implement alternatives.
