+++
title = 'Outside In Design'
tags = ['configuration', 'architecture', 'design']
+++

Following on from my previous post about not having [too much configuration](/2024/10/31/too-much-configuration/), I want to talk about how I design software.

I try and follow what I call "outside in design"; I try and make something that requires the bare minimum amount of configuration to cover the most common of use-cases.  Once this functionality is working, further configuration can be added to cover the next most common use cases.

## API Reduction As A Feature

The first example I want to go through is how I removed options from an HTTP rate limiter we use.  There are many teams using rate limiters, and we have noticed that there are often similar mistakes made in how they work and duplication of domain-specific functionality.

In order to make life easier for _most_ users, a new rate limiter was made which reduced the API surface area, only exposing the bare minimum options.


**Algorithm.** Instead of offering many different types of algorithm (for example `Token Bucket`, `Leaky Bucket`, `Fixed Window Counter`, and `Sliding Window`), the rate limiter only uses `Sliding Window`.

**Sizes.**: By forcing the use of a specific algorithm, we eliminate a lot of algorithm specific options, such as bucket capacity, refill/drain rate, and window overlap.  We expose a single option of `WindowSeconds` with a default vault of `60`.

**Penalties.** We also decided to not expose how long a ban is, and instead make it a multiple of the `WindowSeconds` option, in our case `WindowSeconds * 3`.

**Selection Criteria.** Rate limiters we observed could filter by many different properties, such as IP Address, Headers, Cookies, Path, Query, Status Code, etc.  Our rate limiter has the following rules:

- Account and Path
- Account and (bad) Status
- IP Address and Path
- IP Address and (bad) Status
- Anonymous IP

The way the account is detected is the same for all services in our domain, so we can centralise the checking code to be identical in all instances.

**Triggering.** For triggering, we went with the simplest thing possible: if any counter's value goes over a threshold, then a ban is issued for the Account or IP address triggering the ban.  The threshold value is exposed as `MaxRequests` with a default value of `100`

Finally, we expose one additional configuration: `Storage`.  This is an optional field which you can set to a Valkey or Redis-compatible client so that if you have many instances of your application, the rate-limiting is shared amongst all instances.

For most of our teams, using a rate limiter is now this:

```go
limiter := &org.NewRateLimiter(
  org.WithStorage(valkey),
  org.WithMaxRequests(80),
)

mux := http.NewServeMux()
mux.Handle("/", limiter(apiRootHandler))
```

For teams that need more customisation, we recommend they reach out to us to see what their needs are; the outcome would usually be embedding and customising the rate-limiter, forking the library, or using an off-the-shelf library directly.  So far very few teams have needed extra customisation.

The downside to this approach is if we want to change some detail about how the rate limiter works; we now have to find all the teams using it to make sure we don't break their workflow.  For Go projects, we typically bump the major version of the library, and have explicit messaging about the differences in the readme.

## Organisation Conventions


The next example is trying to show off what we can achive when using conventions;  some of these conventions were in place before we wrote this code, and some have become conventions since.

When it comes to building docker containers, there are a few things that people, in general, want:

- the container to be built
- tests to be run, preventing publishing broken containers
- the (working) container to be published somewhere so it can be used
- extra artifacts from the build to be collected (test reports, coverage, etc.)
- it to be fast

The problem with all these things is in the details; building itself is fairly straightforward, but publishing requires knowing where to publish, any credentials required, and how to name and version the container.  Likewise, artifact collection requires knowing where the artifacts are to be collected from, and where to publish them to (along with authentication etc.)

The even bigger issue is "to be fast"; people don't care about how its fast, they just want fast.  This means not only making a cacheable dockerfile but doing that caching somehow; with ephemeral build agents, that caching becomes harder.

We can go through our requirements and see what ones we know the answers to already and what we need to get from users:

**Docker Registry.** The internal Secret Management Service (SMS) has a convention for where your docker registry is, and what the credentials are: read from `/teams/$team_name/docker/registry`.

**Container Path.** We always publish to `$registry/$team_name/$repo_name/$container_name`.

**Container Name.** A repository can have multiple containers, or the name of the container can differ from the repository.  So for this property, we need the users to supply something.

**Container Version.** We decided that a short git SHA is enough for versioning.

**Caching.** The registry has a second path convention for storing cache contents: `$registry/cache/$team_name/$repo_name/$container_name`.

**Artifacts.** Artifacts are published to the Github Actions artifacts, so no extra authentication or settings are needed.  We decided that if the `.artifacts` folder exists and has contents, that is what will get stored.

Given the above analysis, we decided on 4 configuration options:

1.  `team_name`: no default.  We will use this value to find the registry information and build container and cache paths.
2.  `container_name`: no default.  You need to tell us what the name of your container should be.
3.  `build_args`: default empty.  Supply extra arguments to the `docker build` command.  Some teams need to inject extra information from the host.
4. `dockerfile`: default `./Dockerfile`.  Some teams have multiple dockerfiles in their repository, or keep the files in subfolders.

By relying on the `team_name` parameter, so many other options can be eliminated, and it turns out most people don't care what exact path their containers are uploaded to, as long as they are accessible when it comes to being used in a deployment environment.  This is foreshadowing!

For most teams, their build workflow becomes just two steps: checkout sourcecode, and build the container:

```yaml
steps:
- uses: actions/checkout@v4

- uses: org/docker-build@v1
  with:
    team_name: "team-one"
    container_name: api
```

## Organisation Conventions Two


Now that we have a shared way to build docker containers with low configuration, the next logical step was figuring out if we could do the same for deployment.  It turns out a lot of the conventions used to build the container can be applied to deployment: docker registry, container path, container name, and container version are all the same between the two.  In addition, we need to add a few more: the name of the environment being deployed to and the path to your deployment definition file (for example, a Nomad job).


```yaml
steps:
- uses: org/nomad-docker@v1
  with:
    team_name: "team-one"
    container_name: api
    environment: live
```

## The Pit of Success

We also like to leverage The Pit of Success, which seems to originate from [Rico Mariani](https://learn.microsoft.com/en-us/archive/blogs/brada/the-pit-of-success); we want to make doing the easiest thing to be the correct thing.

To that end, we provide a library to populate an app's secrets.  This library handles multiple forms of authentication for different runtime locations (developer machine, nomad cluster, lambda, etc.), and handles where the secrets themselves are located.

The library's usage boils down to two things.  A single `struct` to represent their secrets:

```go
type Secrets struct {
  ClientID string
  ClientSecret string
  ApiToken string
  // etc
}
```

And a single function call to populate it:
```go
err := org.ReadSecrets(ctx, "management-api", secrets)
```

This function call does a lot behind the scenes:

**Authentication.** This varies based on where the app is running: on a developer machine, it uses the local cached secret manager credentials and triggers authentication flows if needed.  When deployed, it uses the relevant secret authentication system for that environment (e.g. Nomad's Vault integration or AWS Secret Manager in Lambda).

**Secret Location.** It reads all the secrets for from a conventional path: `/teams/$team_name/apps/$app_name/$env/*`, where the values come from different places:

- `team_name` comes from a common environment variable, and `ReadSecrets` errors if its not populated
- `env` comes from either an environment variable when the app is deployed somewhere or is set to `local` on a developer's machine.
- `app_name` is supplied in code (`management-api` in this case)


While teams can roll their own secret management integration, our library is so easy to use that almost no teams choose to do anything different.

## The Golden Path

Our tools form what we call our Golden Path, a term which seems to originate from [spotify](https://engineering.atspotify.com/2020/08/how-we-use-golden-paths-to-solve-fragmentation-in-our-software-ecosystem/#:~:text=The%20Golden%20Path%20%E2%80%94%20as%20we,this%20opinionated%20and%20supported%20path.).  We use it to define a way to develop and deploy software in a tried and tested manner.  Teams are always free to choose their own path by changing what parts of the system they see fit.

The trade off teams are making is between maintenance burden and configuration;  choose our tools, and you don't need to worry about them working, but you need to follow our conventions and opinions.

## How Do You Design Software?

While this is working really well for me and my teams, there has to be other opinions too; I'd be interested in hearing how people do this for their teams and projects.