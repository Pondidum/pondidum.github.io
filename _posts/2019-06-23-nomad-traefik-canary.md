---
layout: post
title: Canary Routing with Traefik in Nomad
tags: infrastructure vagrant nomad consul traefik
---

I wanted to implement canary routing for some HTTP services deployed via [Nomad](https://www.nomadproject.io/) the other day, but rather than having the traffic split by weighting to the containers, I wanted to direct the traffic based on a header.

My first choice of tech was to use [Fabio](https://fabiolb.net/), but it only supports routing by URL prefix, and additionally with a route weight.  While I was at [JustDevOps](https://justdevops.org/) in Poland, I heard about another router/loadbalancer which worked in a similar way to Fabio: [Traefik](https://traefik.io/).

While Traefik also doesn't directly support canary routing, it is much more flexible than Fabio, also allowing request filtering based on HTTP headers.  Traefik integrates with a number of container schedulers directly, but Nomad is not one of them.  It does however also support using the Consul Service Catalog so that you can use it as an almost drop-in replacement for Fabio.

So let's get to the setup.  As usual, there is a complete repository on GitHub: [Nomad Traefik Canary Routing](https://github.com/Pondidum/nomad-traefik-canary-demo).

## Nomad

As usual, I am using my [Hashibox](https://github.com/Pondidum/hashibox) [Vagrant](https://vagrantup.com/) base image, and provisioning it as a single Nomad server and client node, using [this script](https://github.com/Pondidum/nomad-traefik-canary-demo/blob/master/scripts/server.sh).  I won't dig into all the setup in that, as I've written it a few times now.

## Consul

Consul is already running on the Hashibox base, so we have no further configuration to do.

## Traefik

Traefik can be deployed as a Docker container, and either configured through a TOML file (yay, [not yaml!](https://noyaml.com/)) or with command line switches.  As we only need a minimal configuration, I opted to use the command line.

The container exposes two ports we need to care about: `80` for incoming traffic to be routed, and `8080` for the UI, which are statically allocated to the host as `8000` and `8080` for this demo.

The command line configuration used is as follows:

* `--api` - enable the UI.
* `--consulcatalog` - Traefik has two ways to use Consul - `--consul` uses the KV store for service definitions, and `--consulcatalog` makes use Consul's service catalogue.
* `--consulcatalog.endpoint=consul.service.consul:8500` as Consul is not running in the same container as Traefik, we need to tell it where Consul is listening, and as we have [DNS Forwarding for `*.consul` domains](), we use the address `consul.service.consul`.  If DNS forwarding was not available, you could use the Nomad variable `${attr.unique.network.ip-address}` to get the current task's host's IP.
* `--consulcatalog.frontEndRule` disable the default rule - each service needs to specify `traefik.frontend.rule`.
* `--consulcatalog.exposedByDefault=false` - lastly, we stop Traefik showing all services registered into consul, the will need to have the `traefik.enable=true` tag to be processed.

The entire job file is listed below:

```bash
job "traefik" {
  datacenters = ["dc1"]
  type = "service"

  group "loadbalancers" {
    count = 1

    task "traefik" {
      driver = "docker"

      config {
        image = "traefik:latest"

        args = [
          "--api",
          "--consulcatalog",
          "--consulcatalog.endpoint=consul.service.consul:8500",
          "--consulcatalog.frontEndRule=''",
          "--consulcatalog.exposedByDefault=false"
        ]

        port_map {
          http = 80
          ui = 8080
        }
      }

      resources {
        network {
          port "http" { static = 8000 }
          port "ui" { static = 8080 }
        }

        memory = 50
      }

    }
  }
}
```

We register the job into Nomad, and then start on the backend services we will route to:

```bash
nomad job run jobs/traefik.nomad
```

## The Backend Services

To demonstrate the services can be routed to correctly, we can use the `containersol/k8s-deployment-strategies` docker container.  This image exposes an HTTP service which responds with the container's hostname and the content of the `VERSION` environment variable, something like this:

```bash
$ curl http://echo.service.consul:8080
# Host: 23351e48dc98, Version: 1.0.0
```

We'll start by making a standard nomad job for this container, and then update it to support canarying.  The entire job is listed below:

```ruby
job "echo" {
  datacenters = ["dc1"]
  type = "service"

  group "apis" {
    count = 3

    task "echo" {
      driver = "docker"

      config {
        image = "containersol/k8s-deployment-strategies"

        port_map {
          http = 8080
        }
      }

      env {
        VERSION = "1.0.0"
      }

      resources {
        network {
          port "http" { }
        }
      }

      service {
        name = "echo"
        port = "http"

        tags = [
          "traefik.enable=true",
          "traefik.frontend.rule=Host:api.localhost"
        ]

        check {
          type = "http"
          path = "/"
          interval = "5s"
          timeout = "1s"
        }
      }
    }
  }
}
```

The only part of interest in this version of the job is the `service` stanza, which is registering our echo service into consul, with a few tags to control how it is routed by Traefik:

```ruby
service {
  name = "echo"
  port = "http"

  tags = [
    "traefik.enable=true",
    "traefik.frontend.rule=Host:api.localhost"
  ]

  check {
    type = "http"
    path = "/"
    interval = "5s"
    timeout = "1s"
  }
}
```

The `traefik.enabled=true` tag allows this service to be handled by Traefik (as we set `exposedByDefault=false` in Traefik), and `traefik.frontend.rule=Host:api.localhost` the rule means that any traffic with the `Host` header set to `api.localhost` will be routed to the service.

Which we can now run the job in Nomad:

```bash
nomad job run jobs/echo.nomad
```

Once it is up and running, we'll get 3 instances of `echo` running which will be round-robin routed by Traefik:

```bash
$ curl http://traefik.service.consul:8080 -H 'Host: api.localhost'
#Host: 1ac8a49cbaee, Version: 1.0.0
$ curl http://traefik.service.consul:8080 -H 'Host: api.localhost'
#Host: 23351e48dc98, Version: 1.0.0
$ curl http://traefik.service.consul:8080 -H 'Host: api.localhost'
#Host: c2f8a9dcab95, Version: 1.0.0
```

Now that we have working routing for the Echo service let's make it canaryable.

## Canaries

To show canary routing, we will create a second version of the service to respond to HTTP traffic with a `Canary` header.

The first change to make is to add in the `update` stanza, which controls how the containers get updated when Nomad pushes a new version.  The `canary` parameter controls how many instances of the task will be created for canary purposes (and must be less than the total number of containers).  Likewise, the `max_parallel` parameter controls how many containers will be replaced at a time when a deployment happens.

```diff
group "apis" {
  count = 3

+  update {
+    max_parallel = 1
+    canary = 1
+  }

  task "echo" {
```

Next, we need to modify the `service` stanza to write different tags to Consul when a task is a canary instance so that it does not get included in the "normal" backend routing group.

If we don't specify at least 1 value in `canary_tags`, Nomad will use the `tags` even in the canary version - an empty `canary_tags = []` declaration is not enough!

```diff
service {
  name = "echo"
  port = "http"
  tags = [
    "traefik.enable=true",
    "traefik.frontend.rule=Host:api.localhost"
  ]
+  canary_tags = [
+    "traefik.enable=false"
+  ]
  check {
```

Finally, we need to add a separate `service` stanza to create a second backend group which will contain the canary versions.  Note how this group has a different name, and has no `tags`, but does have a set of `canary_tags`.

```ruby
service {
  name = "echo-canary"
  port = "http"
  tags = []
  canary_tags = [
    "traefik.enable=true",
    "traefik.frontend.rule=Host:api.localhost;Headers: Canary,true"
  ]
  check {
    type = "http"
    path = "/"
    interval = "5s"
    timeout = "1s"
  }
}
```

The reason we need two `service` stanzas is that Traefik can only create backends based on the name of the service registered to Consul and not from a tag in that registration.  If we just used one `service` stanza, then the canary version of the container would be added to both the canary backend and standard backend.  I was hoping for `traefik.backend=echo-canary` to work, but alas no.


The entire updated jobfile is as follows:

```ruby
job "echo" {
  datacenters = ["dc1"]
  type = "service"

  group "apis" {
    count = 3

    update {
      max_parallel = 1
      canary = 1
    }

    task "echo" {
      driver = "docker"

      config {
        image = "containersol/k8s-deployment-strategies"

        port_map {
          http = 8080
        }
      }

      env {
        VERSION = "1.0.0"
      }

      resources {
        network {
          port "http" { }
        }

        memory = 50
      }

      service {
        name = "echo-canary"
        port = "http"

        tags = []
        canary_tags = [
          "traefik.enable=true",
          "traefik.frontend.rule=Host:api.localhost;Headers: Canary,true"
        ]

        check {
          type = "http"
          path = "/"
          interval = "5s"
          timeout = "1s"
        }
      }

      service {
        name = "echo"
        port = "http"

        tags = [
          "traefik.enable=true",
          "traefik.frontend.rule=Host:api.localhost"
        ]
        canary_tags = [
          "traefik.enable=false"
        ]

        check {
          type = "http"
          path = "/"
          interval = "5s"
          timeout = "1s"
        }
      }
    }
  }
}
```

## Testing

First, we will change the `VERSION` environment variable so that Nomad sees the job as changed, and we get a different response from HTTP calls to the canary:

```diff
env {
-  VERSION = "1.0.0"
+  VERSION = "2.0.0"
}
```

Now we will update the job in Nomad:

```
nomad job run jobs/echo.nomad
```

If we run the status command, we can see that the deployment has started, and there is one canary instance running.  Nothing further will happen until we promote it:

```bash
$ nomad status echo
ID            = echo
Status        = running

Latest Deployment
ID          = 330216b9
Status      = running
Description = Deployment is running but requires promotion

Deployed
Task Group  Promoted  Desired  Canaries  Placed  Healthy  Unhealthy  Progress Deadline
apis        false     3        1         1       1        0          2019-06-19T11:19:31Z

Allocations
ID        Node ID   Task Group  Version  Desired  Status   Created    Modified
dcff2555  82f6ea8b  apis        1        run      running  18s ago    2s ago
5b2710ed  82f6ea8b  apis        0        run      running  6m52s ago  6m26s ago
698bd8a7  82f6ea8b  apis        0        run      running  6m52s ago  6m27s ago
b315bcd3  82f6ea8b  apis        0        run      running  6m52s ago  6m25s ago
```

We can now test that the original containers still work, and that the canary version works:

```bash
$ curl http://traefik.service.consul:8080 -H 'Host: api.localhost'
#Host: 1ac8a49cbaee, Version: 1.0.0
$ curl http://traefik.service.consul:8080 -H 'Host: api.localhost'
#Host: 23351e48dc98, Version: 1.0.0
$ curl http://traefik.service.consul:8080 -H 'Host: api.localhost'
#Host: c2f8a9dcab95, Version: 1.0.0
$ curl http://traefik.service.consul:8080 -H 'Host: api.localhost' -H 'Canary: true'
#Host: 496840b438f2, Version: 2.0.0
```

Assuming we are happy with our new version, we can tell Nomad to promote the deployment, which will remove the canary and start a rolling update of the three tasks, one at a time:

```bash
nomad deployment promote 330216b9
```

## End

My hope is that the next version of Traefik will have better support for canary by header, meaning I could simplify the Nomad jobs a little, but as it stands, this doesn't add much complexity to the jobs, and can be easily put into an Architecture Decision Record (or documented in a wiki page, never to be seen or read from again!)
