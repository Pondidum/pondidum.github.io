+++
tags = ['make', 'ci']
title = 'Content based change detection with Make'

+++

On several occasions when building complex projects, I have been tempted to set up [Bazel] to help speed up the build process; after all, it has a lot to offer: only building what has changed, caching built artifacts, and sharing that cache between machines for even more speed.

## TLDR

We can use [Make] and a couple of short shell scripts to implement file content-based caching and read/write that cache to remote storage, such as S3.  The [demo repository][make-cas] contains a version using [minio] for ease of demonstration.

## Bazel

However, Bazel has quite a high barrier to entry; there are two drawbacks: a specialised build language and the need to host extra components.  While the specialised language is not much of a drawback, the hosting side is more of an issue.  If you wish to have a shared cache (which is required to get fast builds), you need to either run `bazel-remote`, which is not actually part of the Bazel project, and requires some shared storage such as S3, or Nginx, which again requires some shared storage somewhere.

It boils down to not wanting to have to maintain a lot of infrastructure on top of all the usual CI bits just to have fast builds.

## So what about Make?

Whereas Bazel's caching method is based on a hash of the input artifacts, [Make]'s is based on the input sources and output artifacts' `lastModified` times.

I tried adding distributed caching to Make by copying the output artifacts to S3, and on the next build (on a different agent), restoring them to the working directory, and seeing what would happen.

As both Git and S3 set the file `lastModified` dates to the time they ran, the build process either never ran (artifacts are newer than source), or always ran (sources are newer than artifacts).

This sent me on a relatively short journey to see if I could add hash-based change detection to Make, without recompiling Make.

Spoiler: it is!

## Hashing

The first question is how to hash all our source files reliably.  It turns out you can do all of this with `sha256sum` in a one-liner:

```shell  {linenos=table}
find src -iname "*.ts" -print0 \
  | xargs -0 sha256sum \
  | LC_ALL=C sort \
  | sha256sum \
  | cut -d" " -f 1
```

This does the following:

1.  Find all typescript files (for example)
2.  Sort all the files [using the "simple" locale](https://unix.stackexchange.com/a/87763)
3.  generate a hash of the content of each file
4.  generate a hash of all the path+hash pairs
5.  trim the output to only the hash

Now that I have a hash for the files, its time to figure out how to use that with Make.

We'll be trying to make this totally legitimate build target in a `Makefile` run only when content changes, regardless of file edit dates:


{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="40e26dde9973479a861e4521e2a55d8222451b28"
  file="makefile"
  start=10
  finish=15
>}}

All this build step does is write the current date to a file called `dist/index.js`.  To make this more realistic, you could change the `sleep 3s` to `sleep 10m` ;)


The idea I have to make this hashing work is to use a file that I control and mess with its edit date:

1.  Check if a file called `${current_hash}` exists
2.  If it doesn't exist, write the current timestamp to a new file called `${current_hash}`
3.  If it does exist, set the file `${current_hash}`'s modified date to the timestamp stored inside the file
4.  `echo` the filename so that it can be consumed by Make

This way, the file's edit date will change whenever the hash changes,  and if the hash doesn't change, we leave the edit date as is (which fixes the S3 file edit date being wrong.)

Code wise, it's a few lines of shell script:

{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="1f88779e38c44f4b3af4155de5d864354863e05a"
  file="build/cas.sh"
  start=3
>}}

And the usage inside the `makefile` is only adding an extra `$(shell ./build/cas.sh .... )` around our dependency list:

{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="1f88779e38c44f4b3af4155de5d864354863e05a"
  file="makefile"
  start=10
  finish=15
>}}

## Testing

We have a few test cases to cover:

1.  Entirely blank repository; after all, it should work when you first run `git clone`
    ```shell
    $ git clean -dxf
    $ make build
      ==> Building
      ==> Done
    ```
2.  Files have not changed at all; it should have the same behaviour as normal make, i.e. nothing happens
    ```shell
    $ git clean -dxf
    $ make build
      ==> Building
      ==> Done
    $ make build
      make: Nothing to be done for 'build'.
    ```
3.  File `lastModified` date has changed; this should cause nothing to happen also, as the content of the files hasn't changed:
    ```shell
    $ git clean -dxf
    $ make build
      ==> Building
      ==> Done
    $ touch src/index.ts
    $ make build
      make: Nothing to be done for 'build'.
    ```
4.  File content has changed (but `lastModified` hasn't); forcing a file to have different content with the same `lastModified` to show that its only content that matters:
    ```shell
    $ git clean -dxf
    $ make build
      ==> Building
      ==> Done
    $ set old_date (date -r src/index.ts "+%s")
    $ echo "// change" >> src/index.ts
    $ touch -d @$old_date src/index.ts
    $ make build
      ==> Building
      ==> Done
    ```

## Collecting Assets

Before we can implement remote caching, we need to be able to mark what assets should be included for the given source hash.

I initially tried to achieve this by passing the name of the make target into the `cas.sh` script, but this involves a lot of repetition as the special "target name" make variable (`$@`) doesn't work if it's included in the source list:

```make
dist/index.js: $(shell ./build/cas.sh $@ $(shell find src -iname "*.ts" -not -iname "*.test.ts"))
  @echo "==> Building"
```

Besides not working, this is also not very flexible; what happens if you have other artifacts to store, other than the one acting as your make target?  What happens if you are using a sentinel file instead of actual output as a make target?  or a `.PHONY` target?

The answer to these questions is an extra script to store artifacts, called `artifact.sh`, which writes the path of an artifact to the hash file with a prefix of `artifact: `:

{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="e0889484bd55e1098a85f4b15f2c81a8798321e9"
  file="build/artifact.sh"
>}}


Which is used in the `makefile`, utilising some of Make's magic variables: the `$<` is the filepath to the first dependency (which is the hash file produced by `cas.sh`), and usually, we use `$@`, which is the name of the target being built.  In this example, a second invocation marks another file as an artifact of the make rule:

{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="e0889484bd55e1098a85f4b15f2c81a8798321e9"
  file="makefile"
  start=31
>}}

## Remote Caching

As mentioned earlier, I want to manage as little infrastructure for this as possible, so cloud object storage such as S3 is ideal.  For local testing, we'll use a [minio] docker container.

First up, as I want this to be reasonably extensible, rather than hardcode s3 logic into the scripts, I check for an environment variable `CAS_REMOTE`, and execute that with specific arguments if it exists, both in `cas.sh` and `artifact.sh`:

{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="e3c00c7924d24a1aba6bdd2fad7996a3428ee530"
  file="build/cas.sh"
  options="hl_lines=12-14 22-29 35-37"
>}}


{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="e3c00c7924d24a1aba6bdd2fad7996a3428ee530"
  file="build/artifact.sh"
  options="hl_lines=12-16"
>}}

The main point is keeping how state and artifacts are copied around separate from the logic of how their `lasModified` dates are manipulated.  In the case of the `fetch-artifacts` call, we first pull all the artifacts using the remote script, and then update their `lastModified` dates to match the state's `lastModified` date:

{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="e3c00c7924d24a1aba6bdd2fad7996a3428ee530"
  file="build/cas.sh"
  start=22
  finish=27
>}}

## S3 Remote Cache

The S3 remote script implements four functions: `fetch-state`, `fetch-artifacts`, `store-state`, and `store-artifact`, with the convention that the first parameter is always the key - e.g. the state file name.

In this demo, the actual S3 command is defaulted to use the local minio endpoint, unless `CAS_S3_CMD` is specified, as I cannot find a way to set the `--endpoint-url` via an environment variable directly:

{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="e3c00c7924d24a1aba6bdd2fad7996a3428ee530"
  file="build/remote_s3.sh"
  start=5
  finish=5
>}}

This is used in each of the four functions to interact with S3.  For example, to fetch the state; note how we use both `--quiet` and `>&2` to redirect all output to `stderr`, as anything on `stdout` make will pick up as a filename, causing issues.  We also use `|| true` for fetching state, as it might not exist:

{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="e3c00c7924d24a1aba6bdd2fad7996a3428ee530"
  file="build/remote_s3.sh"
  start=13
  finish=20
>}}

## Testing Remote Caching

First, we need to start our minio container and configure the environment:

```shell
docker-compose up -d
export "AWS_ACCESS_KEY_ID=minio"
export "AWS_SECRET_ACCESS_KEY=password"
export "CAS_REMOTE=./build/remote_s3.sh"
export "CAS_S3_BUCKET_PATH=makestate/cas-demo/"
export "CAS_READ_ONLY=0"
export "CAS_VERBOSE=1"
```

Also, we need to create the S3 bucket using the AWS cli:

```shell
aws --endpoint-url http://localhost:9000 s3 mb s3://makestate
```

We're now ready to try a build:

```shell
$ git clean -dxf
$ make build
  c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256: Fetching remote state to .state/c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256
  c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256: Storing state from .state/c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256
  977e50e9421f0a2749587de6a887ba63f2ddf9109d27ab7cae895a6664b2711a.sha256: Fetching remote state to .state/977e50e9421f0a2749587de6a887ba63f2ddf9109d27ab7cae895a6664b2711a.sha256
  977e50e9421f0a2749587de6a887ba63f2ddf9109d27ab7cae895a6664b2711a.sha256: Storing state from .state/977e50e9421f0a2749587de6a887ba63f2ddf9109d27ab7cae895a6664b2711a.sha256
  ==> Building
  ==> Done
  Storing dist/index.js
  c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256: Storing artifact dist/index.js
  c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256: Storing state from .state/c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256
$ cat dist/index.js
  compiled at la 17.9.2022 13.16.48 +0300
```

If we now clean the repository and build again, we should end up with all the artifacts from the original build but no build process actually running:

```shell
$ git clean -dxf
$ make build
  c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256: Fetching remote state to .state/c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256
  c2bac686e507434398d9bf4e33f63f275dfd3bfecfe851d698f8f17672eeccbe.sha256: Fetching dist/index.js
  977e50e9421f0a2749587de6a887ba63f2ddf9109d27ab7cae895a6664b2711a.sha256: Fetching remote state to .state/977e50e9421f0a2749587de6a887ba63f2ddf9109d27ab7cae895a6664b2711a.sha256
  make: Nothing to be done for 'build'.
$ cat dist/index.js
  compiled at la 17.9.2022 13.16.48 +0300
```

## Extra Features

I added a `CAS_READ_ONLY` environment variable, which by default prevents the scripts from pushing state and artifacts to remote storage but does allow fetching from storage.  The idea of this is that local development can make use of the caches, but only CI machines can write to the cache:

{{< git-embed
  user="Pondidum"
  repo="make-cas"
  ref="b40aabf2affa88a4d8f143ac5895354d4e932bad"
  file="build/artifact.sh"
  start=12
  finish=12
>}}

## Wrapping Up

Overall, I am very happy with how this has gone; it all works, and hopefully I'll be testing it in parallel to normal build processes over the coming weeks.

[bazel]: https://bazel.build/
[make]: https://www.gnu.org/software/make/
[minio]: https://min.io/
[make-cas]: https://github.com/Pondidum/make-cas
