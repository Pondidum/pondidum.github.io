---
layout: post
title: Alarm Fatigue
tags: support alarms oncall
---

I've been on-call for work over the last week for the first time, and while it wasn't as alarming (heh) as I thought it might be, I have had a few thoughts on it.

## Non-action Alarms

We have an alarm periodically about an MVC View not getting passed the right kind of model.  The resolution is to mark the bug as completed/ignored in YouTrack.  Reading the stack trace, I can see that the page is expecting a particular model, but is being given a `HandleErrorInfo` model, which is an in built type.  After some investigation and a quick pull-request, we no longer get that error message.  Turns out the controller was missing an attribute which would allow custom error handling.

## Un-aggregated Alarms

If there is one time out in a system...I don't care that much. If there are multiple in a short space of time then I want an alarm. Otherwise, it should be logged and reviewed when I am next at the office, so I can look  for a pattern, such as it happening hourly on the hour, or every 28 hours.

## Bad Error Messages

`No or more than one usable entry found` - this exception makes sense, if you have the context it is thrown from.  However, when reading the stacktrace in YouTrack, it's not idea.  Some investigation shows that all of the data required to write some good exception messages is available.

This gets harder when the exception is thrown from a library, especially when most of the code required to generate the exception is marked as internal, and that the library only throws one kind of error, differing only by message.  They way I will solve this one is to catch the error, and if the message is the one I care about, throw a new version with a better message.  Unfortunately to build that message, I will have to regex the first exception's message.  Sad times.

## Run Books

We have some! Not as many as I would like, but hopefully I can expand on them as I learn things.  One thing I noticed about them is they are named after the business purpose, which is great from a business perspective...but is not so great when I am looking to see if there is a run book for an exception in YouTrack.  A few things can be done to fix this.

First off, we can rename the run books to resemble the exception messages to make them easier to find.  The business description can be included as a sub-title, or in the body of the run book.

Next is to update the exceptions thrown to have a link to the relevant run book in them, so that when someone opens a ticket in YouTrack, they can see the link to the how to solve it.

Third, and my favourite, is to get rid of the run book entirely, by automating it.  If the run book contains a fix like "run x query, and if it has rows, change this column to mark the job as processed", then we can just make the code itself handle this case, or modify the code to prevent this case from happening at all.

## Overall

Overall, I enjoyed being on call this week.  Nothing caught fire too much, and I have learnt quite a bit about various parts of our system which I don't often interact with.  But on-call (and thus on-boarding) can be improved - so that's what I am going to do.  Hopefully when a new person enters the on-call schedule, they will have an even easier time getting up to speed.

