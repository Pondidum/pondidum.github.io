---
date: "2017-12-09T00:00:00Z"
tags: design process
title: Tweaking Processes to Remove Errors
---

When we are developing (internal) Nuget packages at work, the process used is the following:

1. Get latest of master
2. New branch `feature-SomethingDescriptive`
3. Implement feature
4. Push to GitHub
5. TeamCity builds
6. Publish package to the nuget feed
7. Pull request
8. Merge to master

Obviously `3` to `6` can repeat many times if something doesn't work out quite right.

There are a number of problems with this process:

### Pull-request after publishing

Pull requests are a great tool which we use extensively, but in this case, they are being done too late. By the time another developer has reviewed something, possibly requesting changes, the package is published.

### Potentially broken packages published

As packages are test-consumed from the main package feed, there is the chance that someone else is working on another code base, and decides to update the nuget which you have just published. Now they are pulling in a potentially broken, or unreviewed package.

### Published package is not nessacarily what is on master

Assuming the pull-request is approved with no changes, then the code is going to make it to master. However there is nothing to stop another developer's changes getting to master first, and now you have a merge...and the published package doesn't match what the source says it contains.

### Feature/version conflicts with multiple developers

A few of our packages get updated fairly frequently, and there is a strong likelyhood that two developers are adding things to the same package. Both publish their package off their feature branch, and now someone's changes have been "lost" as the latest package doesn't have bother developer's changes.

## Soltuon: Continuous Delivery / Master Based Development

We can solve all of these issues by changing the process to be more "Trunk Based":

1. Get latest of master
2. New branch `feature-SomethingDescriptive`
3. Implement feature
4. Push to GitHub
5. Pull request
6. TeamCity builds branch
7. Merge to master
8. TeamCity builds & publishes the package

All we have really changed here is to publish from master, rather than your feature branch. Now a pull-request has to happen (master branch is Protected in GitHub) before you can publish a package, meaning we have elimnated all of the issues with our previous process.

Except one, kind of.

How do developers test their new version of the package is correct from a different project? There are two solutions to this (and you could implement both):

* Publish package to a local nuget feed
* Publish packages from feature branches as `-pre` versions

The local nuget feed is super simple to implement: just use a directory e.g. I have `/d/dev/local-packages/` defined in my machine's nuget.config file. We use Gulp for our builds, so modifying our `gulp publish` task to publish locally when no arguments are specified would be trivial.

The publishing of Pre-release packages can also be implemented through our gulp scripts: we just need to adjust TeamCity to pass in the branch name to the gulp command (`gulp ci --mode=Release --barnch "%vcsroot.branch%"`), and we can modify the script to add the `-pre` flag to the version number if the branch parameter is not `master`.

Personally, I would use local publishing only, and implement the feature branch publishing if the package in question is consumed by multiple teams, and you would want an external team to be able to verify the changes made before a proper release.

Now our developers can still test their package works from a consuming application, and not clutter the nuget feed with potentially broken packages.
