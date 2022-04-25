---
date: "2018-08-10T00:00:00Z"
tags: ["rant", "git", "ci", "process", "productivity", "testing"]
title: Branching and Red Builds
---

So this is a bit of a rant...but hopefully with some solutions and workarounds too. So let's kick things off with a nice statement:

I hate broken builds.

So everyone basically agrees on this point I think.  The problem is that I mean *all* builds, including ones on shared feature branches.

Currently, I work on a number of projects which uses small(ish) feature branches.  The way this works is that the team agrees on a new feature to work on creates a branch, and then each developer works on tasks, committing on their own branches, and Pull-Requesting to the feature branch.  Once the feature branch is completed, it's deployed and merged to master.  We'll ignore the fact that Trunk Based Development is just better for now.

![branching, developers working on small tasks being merged into a feature branch](/images/branching-features.png)

The problem occurs when one of the first tasks to be completed is writing behaviour (or acceptance) tests.  These are written in something like SpecFlow, and call out to stubbed methods which throw `NotImplementedException` s.  When this gets merged, the feature branch build goes red and stays red until all other tasks are done.  And probably for a little while afterwards too.  Nothing like "red-green-refactor" when your light can't change away from red!

## The Problems

* Local tests are failing, no matter how much you implement
* PullRequests to the feature branch don't have passing build checks
* The failing build is failing because:
  * Not everything is implemented yet
  * A developer has introduced an error, and no one has noticed yet
  * The build machine is playing up

![branching, developers working on small tasks being merged into a feature branch showing everything as failed builds](/images/branching-features-builds.png)

## Bad Solutions

The first thing we could do is to not run the acceptance tests on a Task branch's build, and only when a feature branch build runs.  This is a bad idea, as someone will have forgotten to check if their task's acceptance tests pass, and will require effort later to fix the broken acceptance tests.

We could also implement the acceptance file and not call any stubbed methods, making the file a text file and non-executable.  This is also a pretty bad idea - how much would you like to bet that it stays non-executable?

## The Solution

Don't have the acceptance tests as a separate task.  Instead, split the criteria among the implementation tasks.  This does mean that your other tasks should be Vertical Slices rather than Horizontal, which can be difficult to do depending on the application's architecture.

## An Example

So let's dream up a super simple Acceptance Criteria:

* When a user signs up with a valid email which has not been used, they receive a welcome email with an activation link.
* When a user signs up with an invalid email, they get a validation error.
* When a user signs up with an in-use email, they get an error

Note how this is already pretty close to being the tasks for the feature?  Our tasks are pretty much:

* implement the happy path
* implement other scenarios

Of course, this means that not everything can be done in parallel - I imagine you'd want the happy path task to be done first, and then the other scenarios are probably parallelisable.

So our trade-off here is that we lose some parallelisation, but gain feedback. While this may seem insignificant, it has a significant impact on the overall delivery rate - everyone knows if their tasks are complete or not, and when the build goes red, you can be sure of what introduced the problem.

Not to mention that features are rarely this small - you probably have various separate acceptance criteria, such as being able to view an account page.

Oh, and once you can split your tasks correctly, there is only a small step to getting to do Trunk Based Development.  Which would make me happy.

And developer happiness is important.