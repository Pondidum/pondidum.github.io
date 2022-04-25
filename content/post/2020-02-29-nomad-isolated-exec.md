---
date: "2020-02-29T00:00:00Z"
tags: ["infrastructure", "nomad", "docker"]
title: Nomad Isolated Exec
---

One of the many features of [Nomad](https://nomadproject.io) that I like is the ability to run things other than Docker containers.  It has built-in support for Java, QEMU, and Rkt, although the latter is deprecated.  Besides these inbuilt "Task Drivers" there are community maintained ones too, covering Podman, LXC, Firecraker and BSD Jails, amongst others.

The one I want to talk about today, however, is called `exec`.  This Task Driver runs any given executable, so if you have an application which you don't want (or can't) put into a container, you can still schedule it with Nomad.  When I run demos (particularly at conferences), I try to have everything runnable without an internet connection, which means I have to make sure all the Docker containers I wish to run are within a local Docker Registry already, and, well, sometimes I forget.  By using `exec`, I can serve a binary off my machine with no container overheads involved.


## Insecurity?

Until recently, I had always considered `exec` as a tradeoff: I don't need a docker container, but I lose the isolation of the container, and the application I run has full access to everything on this host.

What I hadn't realised, is that `exec` actually uses the host operating system's isolation features via the [libcontainer](https://pkg.go.dev/github.com/opencontainers/runc/libcontainer?tab=doc) package to contain the application.  On Linux, this means using `cgroups` and a `chroot`, making the level of isolation roughly the same as a docker container provides.


When you specify a binary to run, it must meet a few criteria:

- An absolute path within Nomad's `chroot`
- A relative path within the Allocation Directory

For instance, to run a dotnet core application consists of invoking `/usr/bin/dotnet` with the relative path of the dll extracted from the artifact:

```ruby
task "consumer" {
    driver = "exec"

    config {
        command = "/usr/bin/dotnet"
        args = [ "local/Consumer.dll" ]
    }

    artifact {
        source = "http://s3.internal.net/consumer-dotnet.zip"
    }
}
```

Whereas running a go binary can be done with a path relative to the allocation directory:

```ruby
task "consumer" {
    driver = "exec"

    config {
        command = "local/consumer"
    }

    artifact {
        source = "http://s3.internal.net/consumer-go.zip"
    }
}
```

But what happens if we want to run a binary which is not within the default chroot environment used by `exec`?

## Configuring The chroot Environment

By default, Nomad links the following paths into the task's chroot:

```json
[
    "/bin",
    "/etc",
    "/lib",
    "/lib32",
    "/lib64",
    "/run/resolvconf",
    "/sbin",
    "/usr"
]
```

We can configure the `chroot` per Nomad client, meaning we can provision nodes with different capabilities if necessary.  This is done with the `chroot_env` setting in the client's configuration file:

```ruby
client {
  chroot_env {
    "/bin"            = "/bin"
    "/etc"            = "/etc"
    "/lib"            = "/lib"
    "/lib32"          = "/lib32"
    "/lib64"          = "/lib64"
    "/run/resolvconf" = "/run/resolvconf"
    "/sbin"           = "/sbin"
    "/usr"            = "/usr"
    "/vagrant"        = "/vagrant"
  }
}
```

In this case, I have added in the `/vagrant` path, which is useful as I usually provision a Nomad cluster using [Vagrant](https://vagrantup.com), and thus have all my binaries etc. available in `/vagrant`.  It means that my `.nomad` files for the demo have something like this for their tasks:

```ruby
task "dashboard" {
    driver = "exec"

    config {
        command = "/vagrant/apps/bin/dashboard"
    }
}
```

Meaning I don't need to host a Docker Registry, or HTTP server to expose my applications to the Nomad cluster.

## Need Full Access?

If you need full access to the host machine, you can use the non-isolating version of `exec`, called `raw_exec`.  `raw_exec` works in the same way as `exec`, but without using `cgroups` and `chroot`.  As this would be a security risk, it must be enabled on each Nomad client:

```ruby
client {
    enabled = true
}

plugin "raw_exec" {
    config {
        enabled = true
    }
}
```

## Wrapping Up

One of the many reasons I like Nomad is its simplicity, especially when compared to something as big and complex as Kubernetes.  Whenever I look into how Nomad works, I always seem to come away with the feeling that it has been well thought out, and how flexible it is because of this.

Being able to configure the chroot used by the Nomad clients means I can simplify my various demos further, as I can remove the need to have a webserver for an artifact source. As always, the less accidental complexity you have in your system, the better.
