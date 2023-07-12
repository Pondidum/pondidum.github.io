+++
tags = ['communication', 'cd', 'feature flags', 'process', 'productivity']
title = 'QA and Continuous Delivery'

+++

When migrating to a continuous delivery process, it is often the case that a QA team are worried about what their role is going to be, and how the changes will affect the quality of the software in question.

While doing continuous delivery does change the QA process, when done well, it improves everyone's lives and makes the software _better_ quality.  Are silver bullets incoming?  Not quite, but we don't have to make someone's life worse to improve other people's lives.

This article is going to rely heavily on [Feature Flags][feature-flags], so a passing familiarity is useful.  In summary, feature flags are the ability to switch features on and off at runtime of the application without requiring re-deployment.  Feature flags can also be used to switch on features for specific users or groups of users.

> Aside; this post is a bit different from my usual style.  This time I have written a story about a fictional dev team and QA team and how they move towards continuous delivery together.

## TLDR

Move your QA Engineers inside the dev teams; DevOps is a way of working in a cross-functional team; this should include everyone who can contribute.

Test things early.  Involve QA with features hidden behind flags.  De-couple your deployments from your releases.

## Setting The Scene

A team has gotten to the point where they want to switch from deploying at the end of each sprint to deployments happening as often as needed, be that 5 minutes for a text change or a few days for a bigger feature.

When they happily announce this to the rest of their organisation, the QA Team reacts with dismay; how are they going to manage to do full testing before every deployment if the team is constantly deploying?  They object; this is ludicrous.

Being level-headed people, everyone decides to sit down and talk through their concerns and what to do next.  The key points are written down:

- The development team wants to ship faster
- The QA team wants to test everything before it is deployed
- The management team doesn't want to hire 10 more QAs to try and keep up

So what to do?

## The first step

It is important to realise that while we want quality, not all changes are created equal; some need much closer scrutiny than others.  For example, fixing some spelling mistakes probably needs no one else's input (other than a spell-checking tool, perhaps) other than the person doing it.

The teams agree on this; after some discussion, they write down the following:

- Small fixes can be released without a QA approval

This raises a few further questions however:

1. How big is small?
2. If a small fix can be deployed without QA, what about a small feature?
3. Why is QA the final authority on what can be released?

## Changing Perspective

While we could try and answer these questions (and spend countless hours deciding how many lines of code "small" is.  Does it depend on line length too?), a better tactic is to investigate why the QA process is happening _so late in the process_.

We agree that features need QA testing, but what happens if features can be hidden?  What happens if we can move the testing from "before deployment" to "before release"?  Because as I have written before [`deploy doesn't mean release`](deploy-not-release).

The team realises that they have a Feature Flagging tool available.  Currently, they are not really using it, but they have been meaning to for a while.  What if new features were developed behind flags?  It could be deployed to production without affecting anyone, and QA could test at their leisure by enabling the flag for just one tester or for the whole team.

The QA team thinks this could work in principle, but how do they know a change is _really_ isolated behind a flag?  What happens if it escapes?

Let's look at the process they came up with, with an example.

## The New Feature

The current web application has a notification system.  It's nothing glamorous; it's an icon in the app which gets a small dot when there is a new notification.  Currently, only notifications from the system itself are supported, but there has been a request to have other parts of the system send notifications there too, along with feature requests for being able to remove read notifications and mark notifications to trigger again later.

This seems like the ideal candidate for a feature flag, so the development team writes down their next steps:

1. create a flag `enable-rich-notifications`
2. develop all the capabilities (API, UI)
3. deploy
4. QA can test it with the flag
5. release it to the world

Someone points out that Step 2 looks like several weeks of work on its own, and that isn't very continuous.  They break down the tasks a bit further:

1. create a flag `enable-rich-notifications`
2. update the API with a new `/rich` endpoint, which can only be queried if you have the flag.
3. create some fake data for the `/rich` endpoint to return
4. create a new UI component which uses the new endpoint
5. update the application to use the new component if you have the flag and the old component otherwise

With implicit "Deploy" steps after each step.  This seems reasonable to the development team, but the QA team still have questions: when should they test the UI?  once it is fully complete?  And how do they know it is working?

The development team also realises that the new notifications system will be using the same data model as the old system, and they need to make sure the old system continues to work correctly.  Come to think of it, QA involvement would be useful here too...

## Moving QA Earlier

> As an aside, I find it much better to have a QA Engineer be part of the development team.  The whole DevOps thing is about working in one cross-functional team, and why should QA, Security, or anyone else be excluded from this?  Regrettably, this is a slow organisational change to make, so we come up with ways to make it work as best we can and iterate towards the embedded QA model.

When the new notifications feature is being designed, the development team requests someone from QA be involved from the start; there are things which they should be aware of, and have useful input on.


1. Update the data model in place with the new design
2. QA to test it in an isolated environment; no changes expected
3. Deploy

The QA points out that as far as they are aware, there aren't any tests for the old notifications system; it was so barebones and unused that no one bothered.  The QA also points out that they have been evaluating switching to a more code-first UI automation tool, and this might be the ideal candidate to start with, and could they put the UI testing code in the repo alongside the feature?

This is well received by the dev team; this might help the regressions they keep causing when a selector is updated, and the UI tests break; if it's all in the same repository, `grep` can find all the instances at once!  It's win-win again.

The again updated list of actions is now:

1. QA creates UI tests for the current system (and verifies against isolated environment)
2. Devs Update the data model
3. QA verifies nothing has changed in a staging environment
4. Deploy

Note there are no flags involved yet!

The team goes ahead and makes all the discussed changes; however, when the new UI tests are run against the environment with the new data model, they break, and it isn't apparent why.  The QA and the developer sit down and dig around until they find the problem; the format of a field has changed slightly, and the UI tests are catching the problem.

They fix the issue, test again, and this time deploy into production.

## The New API and UI

Now that involving QA earlier has been tried and seems to work, the team decide to move forward with the API changes and the feature flag for the original version and the rich version of notifications.

The flag is created, the API is wrapped with a check for the flag, the developers test it works, and deployment is done.  No problems so far.

The UI is up next; as this is early on in the process, the dev team, designer, and QA engineer are all sitting together to figure out exactly how it will work.  As the QA is present, they can start writing outlines for UI testing.  As the code for tests is alongside the application code, the developers can help keep the tests working as they flesh out the UI, and they might even write a test themselves too.

The interesting realisation comes that with a feature flag, two QAs can be involved at once; one is running tests for the flag off, and one is running the tests for the flag on.  It isn't required to be like this of course, but it does mean you can spread the work further.

Features are developed, tests are written, and deployments are done.

## Ready for Release

The team, which now includes the QA by default, is getting close to being ready to release their new rich notifications to the world.  They have one more test they would like to conduct: what is the load like when users are re-notifying themselves?  How do they even go about testing this?

The answer, perhaps unsurprisingly, is a feature flag.  In this case, a new feature flag called `load-generator-rich-notifications`.  When this flag is enabled, the rich notifications system is still hidden, but a small piece of code randomly activates notifications for re-notifying and varying intervals.  The team can switch it on for a few percent of users and then watch their traces and monitoring systems to keep an eye on the health of the system.

They can add more and more users to the test until they are happy.  Then disable the load generator and clean up all the mess it has left.

> Aside; this is how Facebook Messenger was load tested before the public saw anything!

## Wrapping Up

The key takeaway from this is that QA is an important part of the delivery lifecycle.  Your QA Engineers are smart people who want to make things better, so involve them early and see what conversations and ideas can happen when you put smart people together and task them with making things better.

This was a lot longer than it sounded in my head when I thought this up while cycling home, but I like how it's gone.  I might even turn this into a talk to give to clients if it is well received.

[feature-flags]: /tags/feature-flags/
[deploy-not-release]: /2022/11/02/deploy-doesnt-mean-release/