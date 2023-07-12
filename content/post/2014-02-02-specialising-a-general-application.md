+++
date = '2014-02-02T00:00:00Z'
tags = ['design']
title = 'Specialising a General Application'

+++

Currently our application at work is used by all employees - sales staff, legal team, marketing, accounts etc.  This means we have one very large, and general fit application.  It covers everyone's needs *just*, and the largest group of users (sales in this case) have an application which closely matches what they need.  This is at the expense of the other teams having an application that is not quite right - close, but could be better.

For example, the main UI might look something like this:

![Sales UI][ui-sales]

This is fine for a sales person, who just needs the details on a single person on the system at a time.  However the legal team might only be interested in new contracts and ones which will expire soon.

Adding a report to the existing application which they can then use to find the people with new contracts is one solution, but it still presents them with the same UI - if a person has multiple contracts or many other documents, it won't be particularly helpful to the user.

A better solution would be to give them a separate UI entirely for viewing contracts:

![Legal UI][ui-legal]

This UI has a much closer fit to the Legal team's usage - it only shows the information which is relevant to them, and the actions they can perform are visible and easy to get to.

Implementing UIs like this is a straight forward task - each application has its own data model to display with, which can be a performance increase - each model can be optimized to be as efficient as possible, without having knock-on effects on the other applications.

If this was a web application, you could even make it work out which UI to display based on which user has logged in, rather than deploying one application to some users, and a different application to other users.

Mockups were made using [Moqups.com][mocking-tool]
Names generated with [Behind THe Name][name-gen]

[ui-sales]: /images/specialised-sales.png
[ui-legal]: /images/specialised-legal.png
[name-gen]: http://www.behindthename.com/random/
[mocking-tool]: https://moqups.com/
