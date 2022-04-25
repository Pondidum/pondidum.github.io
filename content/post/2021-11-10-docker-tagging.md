---
date: "2021-11-10T00:00:00Z"
tags: ["docker", "architecture"]
title: How do you tag docker images?
---

An interesting question came up at work today: how do you tag your Docker images?  In previous projects, I've always used a short git sha, or sometimes a semver, but with no great consistency.

As luck would have it, I had pushed for a change in tagging format at a client not so long ago as the method we were using didn't make a lot of sense and, worst of all, it was a _manual_ process.  One of the things that I push at all clients is documenting all architectural decisions made, in the form of [Architecture Decision Records](/2019/06/29/architecture-decision-records), so I'm reproducing it here, with a few details changed to mask where this happened.

One of the most interesting points of this is that I went in with an idea on the right way to do this, and over the course of discussion and review of the document, _changed my mind_.

---

## Change Versioning Scheme

### Status

Accepted

### Context

Currently, the UI uses a [SemVer](https://semver.org/) style version number. However, we have no convention for what kind of modifications constitute a major, minor, or patch change.  We also have no processes or people who care specifically about what kind of change it is, just that a new version was deployed.

The other problem with using SemVer is that people wait until a branch has been approved, and then make an additional commit with the version number change (as another prod deployment might have happened in the meantime), meaning they need to wait for an additional build before they can deploy.

Not to mention, it's possible to accidentally go backwards in numbers if a value was misread or if someone forgets to update the version number in their branch.

### Considered Options

#### 1. Auto-incrementing integer version

On production deployment, we would write a version number to the application.  The negative of this approach is not having a version number in pre-production environments, such as test environments.

We could generate the number on the build phase (when the container is created), but this means that we might not release versions "in order", as the order of what feature is deployed to production is not guaranteed, although the need to merge `master` into your branch would mean a rebuild, so a new version could be generated.

This method would also mean gaps in version numbers, as not all builds hit production, which might be a touch confusing.

Another issue with this method is that we build multiple containers from the same commit in separate pipelines, so we would need some way to generate a version in both pipelines which would match, which would mean either a function deriving from the commit hash or a service which would calculate and cache version numbers so they could be generated and looked up by multiple pipelines.

Example Version:
```
1870
```

#### 2. Git (short) sha of the commit

On build, write the short (7 char) SHA as the version number.  The negative of this approach is not having an easy to understand order of version numbers.  However, this scheme means we can easily see exactly which commit is currently running in production (or any environment, for that matter.)

Example Version:
```
84d33bb
```

#### 3. Build ID from CI System

On build, embed the buildID as the version number.  The pipeline id is a 24 character string consisting of numbers and letters, so this is functionally similar to [Option 2](#2-git-short-sha-of-the-commit), but with a longer number that doesn't tie back to a commit.

As with [Option 1](#1-auto-incrementing-integer-version), we would need to decide if this number comes from the build pipeline, or from the deployment pipeline.  This also has the same multi-pipeline problem too.

Example Version:
```
611a0be261ddea19dab67c22
```

#### 4. Datestamp

On build, use the current commit's datestamp as the tag.

As long as we keep the resolution of the datestamp large enough, the multiple pipelines needing to generate the same ID shouldn't be a problem.  I guess 1-minute resolution would be enough, although if a rebuild is needed (e.g. flakey internet connection), we would end up with a different datestamp.

Example Version:
```
2021-08-16.13-07
```

#### 5. Commit Datestamp

Similar to [Option 4](#4-datestamp), except we use the commit's commit date to build the version number.  This solves multiple pipelines needing to generate the same tag in parallel, as well as being unique and ordered.  The timestamps can also be higher precision than [Option 4](#4-datestamp), as we don't need to hope that pipelines start at a close enough time.

This is how we would generate it:

```shell
timestamp=$(git show -s --format=%cd --date="format:%Y-%m-%d.%H-%M-%S")
```

Example Version:
```
2021-08-16.13-07-34
```

#### 6. Automatic SemVer

On build, calculate the version number using [Semantic-Release](https://github.com/semantic-release/semantic-release).

This method means that we would need to start enforcing commit message styles, and I am not sure the format that Semantic Release is ideal for us, so it might be better to cover the commit message formatting outside this process.

The commit format would be as follows:

```
<type>(<scope>): <short summary>
│       │             │
│       │             └─⫸ Summary in the present tense. Not capitalized. No period at the end.
│       │
│       └─⫸ Commit Scope: animations|bazel|benchpress|common|compiler|compiler-cli|core|
│                          elements|forms|http|language-service|localize|platform-browser|
│                          platform-browser-dynamic|platform-server|router|service-worker|
│                          upgrade|zone.js|packaging|changelog|dev-infra|docs-infra|migrations|
│                          ngcc|ve
│
└─⫸ Commit Type: build|ci|docs|feat|fix|perf|refactor|test
```

Having worked in repositories with this enforced, I would recommend against it, as it causes a lot of frustration ("omg _why_ has my commit been rejected again?!") and as mentioned in other options, I am not sure semver itself makes sense for our UI (or UI projects in general.)

We will still need developers to decide if a given commit is a major/minor/patch.

Example Version:
```
13.4.17
```

#### 6. Combination: Datestamp + Git

On build, use a combination of [Option 5](#5-commit-datestamp) and [Option 2](#2-git-short-sha-of-the-commit) to generate a unique build number.

This method had the advantage of the meaning of the date, with the uniqueness of the git commit, but the likelihood of us needing to distinguish two commits made at identical times by their commit sha is unlikely, especially as we require clean merges to master.

Example Version:
```
2021-08-16.13-07-34.84d33bb
```

### Chosen Decision

[Option 5](#5-commit-datestamp)

We will also embed other build information as labels in the docker container, such as:

- branch name
- pipeline/build number
- git hash
- git commit timestamp

### Consequences

- No need to tag commits as a released version, but we could automate this if we wanted
- No need to rebuild for changing the version number
- No need to remember to change the version number
- No need to decide on major/minor/patch semantics
- Gain an understandable version number, with meaning

---

## Summary

As I said earlier, I went into this process (which I drove) wanting to pick the 2nd option - Short Git Sha, and I came away agreeing that the commit datestamp was the best thing to use.

Not only was my mind changed in the course of this, but also people who join the project later can check out the `./docs/adr/` and see what options we considered for everything about this project, and how we arrived at the conclusions.  It also means I have examples to refer back to when people ask interesting questions at work.

How do _you_ tag your containers?
