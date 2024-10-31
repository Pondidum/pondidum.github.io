+++
title = 'Too Much Configuration'
tags = ['configuration']
+++

When writing software, you will come across many questions which don't always have a clear answer, and its tempting to not answer the question, and provide it as a configuration option to the user of your software.  You might go as far as setting a default value though.  This seems good for everyone; you don't have to make a decision, users can change their minds whenever they want.

However, too much configuration is a bad thing in general.  There are two ways I want to view configuration: internally, from the developer of the software's perspective, and externally, from the user of the software's persepective (who might also be writing software.)

Most of the categories I will cover exist in both internal and external sections, as the effects they have are bi-directional.

## Internal Issues

While internal issues will be more felt by you, the developer of software, I think that they are actually less important than the external issues.  How a user experiences your software is siginficantly important, and in general I would rather take on a little burden to allow my users to have a better experience - to a point though.

### Cognitive Load

One of the most underlooked aspect of configuration is the cognative load it puts on developers.  Every configuration value brings in additional questions, some of which we will touch on later.  For now I want to look at how a value is actually used.  Every piece of indirection in a codebase adds up over time - some indirection is useful and needed for sure, but having too much makes it harder to follow flow of the program.

Often times, configuration values interact with eachother in unexpected ways; a retry-count and a backoff-strategy when configured together can cause your program to retry forever very fast, or retry forever with days between attempts.

### Changing Values

A configuration value is part of your API contract; changing anyhthing about it can break your users, and might keep you from making needed changes.  For example, if a configuration has a default value, and you want to change that default, how do you handle all the usages of your software which are expecting the default to remain the same?  will it impact their usage?  The same goes for removing a default value, renaming a configuration setting, or changing its type.

### Validation

Configuration values need validating, and not just checking they are the expected type (type validation) - they also need semantic validation; checking wether a value makes sense in the given context.  For example, setting an HTTP timeout to 3 days is probably not a useful thing to do.

This adds to the complexity of your software: constants (or even magic values) don't need this extra validation and the more code you have to maintain, the higher the burden of maintenance.

### Complexity

This area in general applies to applications over libraries, but does apply to both, especially when libraries can read configuration from multiple places.

First off, how many configuration sources does your software have?  For a CLI, there are usually at least three sources: Command arguments and flags, environment variables, and configuration files.  These sources need to have their values merged somehow, and even if that strategy is just "cli arguments win", you still have to implement a hierarchy and all the complexities that come with that.  Boolean configurations are fairly straightforward, but how do arrays or nested objects work?  Do you merge the array values?  Do you replace the entire array?  And is the answer the same for all arrays in your configuration, or does it differ?

If you are lucky, all the complexity is encapsulated into one place, such that the rest of you software sees a single `Configuration` object of some form, and doesn't need to worry about how the values got into that structure.

However, even libraries end up doing things with the environment.  For example, the OpenTelemetry libraries will read the environment for default values; this is nice to start with as you have to write less code to get things up and running, and (at least for OpenTelemetry) the environment variables are documented, and the same across languages and versions.  It does mean, though, that the configuration for your OpenTelemetry setup is separate from the rest of your application.  What if you also need to do something that depends on a value that is used for OpenTelemetry?

The final complexity to talk about is the extra code paths introduced by each configuration value.  This applies to things like which storage method to use in the application, rather than only supporting one, or which transports to use.  I have had experience with a vendor whose product supported multiple central logging backends, but it turned out they primarily used Firebase, and all other backends weren't tested so thoroughly.  We saw so many performance problems with the other "supported" backends.  Which leads us nicely to Testing.

### Testing

Testing configuration is tricky; generally its not really done, which leaves you open to problems like "did this value here really used to work?"

Frequently when developing software, I have multiple implementations of a cache in my programs:  there is the real implementation (such as Redis or a filesystem), a testing cache (in memory), and a bunch of decorators (statistics about the cache, tracing of what is happening etc.)  When testing most of my applications, I use the `InMemoryCache` as it is the fastest and eliminates shared state from my tests (or even between test runs), but I have to make sure there are also sufficient integration tests for the application using the real cache - not just testing its implementation.

### Documentation

Configuration values need documenting: what each value does, whether it is required or not, what it's default value is, and any interactions it has (i.e. configuration it cannot work with or requires to also be set.)  Documentation is generally not the strongest of points for developers, and keeping that documentation up to date and accurate again adds to the maintenance burden.

## External Issues

The issues we face internally are also faced externally by the users of our software.  Not all of them are exactly the same, and the problems can be easier or harder than the internal problems.

### Cognative Load

When you encounter a piece of software which has _so many_ options, how do you go about figuring out what settings you need for your usage, and what properties need to be set?  If you are lucky, the [Documentation](#documentation) is up to date and easily discoverable.

One interesting problem occurs with default values; I have seen software where not specifying a property means something very different from specifying it as null/empty/etc.

The interaction of different properties can also be hard to follow; properties which only work when other properties are set (or not set) is one issue, and the other is how the values you specify interact with each other, such as with timeouts and retries.

If there are many configuration properties, trying to discern the right one to change is also a burden, and this gets even worse when there are many sources of configuration:  where is the appropriate place to set the value?  is something else overriding it?  do you even know where all possible places the configuration can come from are?

### Changing Values

This is pretty much the same as the Internal Issues Changing Values section; if I have configured something and it is working, but then at some point the default values of _something_ change, and it stops working, I now have to spend a lot of effort figuring out which value change was the culprit, and possibly try and work out what the old value was so that I can put it back to how it was.

This can be especially difficult if the changelog only says "changed the default value of $thingy to 17" - what it was before is a mystery, but maybe you can figure it out by looking through the commits.  Assuming its opensource (or source available.)

What happens when a new feature is added?  Should it be disabled by default or enabled by default?  What if it suddenly starts doing something you don't want, or its configuration starts affecting your existing configuration?

### Validation

Like the previous validation section, we are trying to think of not only what values can go here but also what format they should be in.  Is the value specified in seconds?  Milliseconds?  Epoch?  or is it a string of "3m20s"?  Hopefully, the software will tell you the value is invalid, but it might just ignore invalid values, making you think you're setting something, but nothing is actually happening.

An experience I had somewhat recently with the [ALB Controller](https://kubernetes-sigs.github.io/aws-load-balancer-controller/) was related to a configuration value.  We had the following set in an annotation, which worked fine:

```yaml
alb.ingress.kubernetes.io/success-codes: 200, 204
```

But someone decided that `204` was no longer a valid success-code, so removed the `, 204`:

```yaml
alb.ingress.kubernetes.io/success-codes: 200
```

...which broke silently when deployed - the problem was that the ALB controller is expecting a string here, when the value changed it suddenly parsed as an integer.  What made this harder to pick up is that the deployment itself worked, but the ALB controller started throwing errors, and that wasn't noticed for a while.  The fix, by the way, was to add quotes around the value.

```yaml
alb.ingress.kubernetes.io/success-codes: "200"
```

### Testing

How often do you test that configuration of software is correct?  I would hazzard a guess at "not often" - and when it is done, it tends to be just testing of the configuration values themselves; checking a timeout is set or not set for example.

What is harder to manage is to test the the given configuration has the desired effect on the software itself.  How do you go about testing that timeouts and retries work as expected?  or authentication parameters are having the desired effect?

This kind of testing is important too, especially when updating software; minor, or even patch versions of software can have breaking changes, and if you're not testing it still does what you want, how will you know that everything is configured correctly?

## Suggestions for better configuration

1.  Don't make something configurable unless it needs to be
2.  Clearly specify what the default values are, what format values are in, and what the property actually does
3.  Ideally, use a system that keeps the documentation for the property with the property in code.  It might stay in sync then.
4.  If there are conventions, follow them.  This applies to all aspects: source of configuration, format, property names, and values.

## Wrapping Up

We haven't even gotten into the debate about the right configuration file language!

In a future post, I want to go over how these thoughts have affected how I design software, and the configuration for it, but this post is quite long enough already so I will wait for another time.