---
layout: post
title: Noticing Changes
Tags: design
permalink: noticing-changes
---

I work on a piece of software that has been around for about 6 years now, which looks something like this:

![Control][1]

The textboxes are validating that their contents, some as decimal, and some as integer.  All the textboxes consider no-value to be invalid.

I made a slight change to the control, which was to add a new row.  Since adding that row, many users have sent in requests to have the validation changed on the textboxes, so that no-value is considered to be zero.

Now while I have no problem in making the change, I do however wonder what caused all the requests.  Is it because users noticed the control was changed, so a developer is paying attention somewhere, so maybe they can fix a problem?  Had they just not noticed that no-value is invalid, and now it has changed slightly, they have?

Another thing is that while it is a very minor change, it must have been causing user friction for the last 6 years or so, and no one has mentioned it?  Maybe they just didn't think it was changeable, or that it just wasn't that bothersome compared to some other issue they had?

[1]: /images/form-validation.jpg