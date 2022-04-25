---
date: "2017-11-17T00:00:00Z"
tags: design architecture process
title: Evolutionary Development
---

Having recently finished reading the [Building Evolutionary Architectures: Support Constant Change book](https://www.goodreads.com/book/show/35755822-building-evolutionary-architectures), I got to thinking about a system which was fairly representative of an architecture which was fine for it's initial version, but it's usage had outgrown the architecture.

## Example System: Document Storage

The system in question was a file store for a multi user, internal, desktop based CRM system.  The number of users was very small, and the first implementation was just a network file share.  This was a fine solution to start with, but as the number of CRM users grew, cracks started to appear in the system.

A few examples of problems seen were:

* Concurrent writes to the same files
* Finding files for a specific record in the CRM
* Response time
* Files "going missing"
* Storage size
* Data retention rules

Most of this was caused by the number of file stored, which was well past the 5 million mark.  For example, queries for "all files for x record" got slower and slower over time.

Samba shares can't be listed in date-modified order (you actually get all the file names, then sorting is applied), which means you can't auto delete old files, or auto index (e.g. export text to elasticsearch) updated files easily.

The key to dealing with this problem is to take small steps - if you have a large throughput to support, the last thing you want to do is break it for everyone at once, by doing a "big bang" release.

Not only can we take small steps in deploying our software, but we can also utilise Feature Toggles to make things safer.  We can switch on a small part of the new system for a small percentage of users, and slowly ramp up usage while monitoring for errors.

## Incremental Replacement

To replace this in an incremental manner, we are going to do the following 4 actions for every feature, until all features are done:

1. Implement new feature in API and client
2. Deploy client (toggle: off)
3. Deploy API
4. Start toggle roll out

Now that we know how each feature is going to be delivered, we can write out our list of features, in a rough implementation order:

* Create API, build scripts, CI and deployment pipeline
* Implement authentication on the API
* Implement fetching a list of files for a record
* Implement fetching a single file's content for a record
* Implement storing a single file for a record
* Implement deletion of a single file for a record

The development and deployment of our features can be overlapped too: we can be deploying the next version of the client with the next feature off while we are still rolling out the previous feature(s).  This all assumes that your features are nice and isolated however!

Once this list of features is done, and all the toggles are on, from the client perspective we are feature complete.

We are free to change how the backend of the API works.  As long as we don't change the API's contract, the client doesn't need any more changes.

Our next set of features could be:

* Implement audit log of API actions
* Publish store and delete events to a queue
* Change our indexing process to consume the store and delete events
* Make the samba hidden (except to the API)
* Implement background delete of old documents
* Move storage backend (to S3, for example)

This list of features doesn't impact the front end (client) system, but the backend systems can now have a more efficient usage of the file store.  As with the client and initial API development, we would do this with a quick, iterative process.

## But we can't do iterative because...

This is a common reaction when an iterative approach is suggested, and thankfully can be countered in a number of ways.

First off, if this is an absolute requirement, we can do our iterations an feature toggling rollouts to another environment, such a Pre-Production, or QA.  While this reduces some of the benefits (we loose out on live data ramp up), it does at least keep small chunks of work.

Another work around is to use feature toggles anyway, but only have a couple of "trusted" users use the new functionality.  Depending on what you are releasing, this could mean a couple of users you know, or giving a few users a non-visible change (i.e. they're not aware they've been selected!)  You could also use NDA (Non Disclosure Agreements) if you need to keep them quiet, although this is quite an extreme measure.

A final option is to use experiments, using an experimentation library (such as [Github's Scientist](https://github.com/github/scientist)) which continues to use the existing features, but in parallel runs and records the results of the replacement feature.  This obviously has to be done with care, as you don't want to cause side effects.

How do you replace old software? Big bang, iterative, experimentation, or some other process?