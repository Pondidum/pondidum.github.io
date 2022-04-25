---
layout: post
title: Service Mesh with Consul Connect (and Nomad)
tags: microservices consul nomad
---

When it comes to implementing a new feature in an application's ecosystem, I [don't like spending my innovation tokens](https://mcfunley.com/choose-boring-technology) unless I have to, so I try not to add new tools to my infrastructure unless I _really_ need them.

This same approach comes when I either want, need, or have been told, to implement a Service Mesh.  This means I don't instantly setup [Istio](https://istio.io/).  Not because it's bad - far from it - but because it's extra complexity I would rather avoid, unless I need it.

But what alternatives are there?

In most large systems I have been involved with [Consul](https://www.consul.io) has been deployed;  usually for Service Discovery, [Feature Toggles](/2018/09/06/consul-feature-toggles/), it's key-value store, or distributed locking.  As Consul has Service Mesh functionality built in, why not use that?

So let's dive into setting up a [Consul Connect](https://www.consul.io/docs/connect/index.html) based Service Mesh.

## Implementing

The demo for this is made up of two parts (taken from HashiCorp's consul demo repo): a counter and a dashboard.  The counter listens for HTTP requests and will return the number of requests it's handled.  The dashboard polls the counter and displays the current count.

All the source code for the demo is in the [Consul Connect Example Repository](https://github.com/Pondidum/consul-connect-nomad-demo).

Clone the repository, and run the build script to create the apps:

```bash
git clone https://github.com/Pondidum/consul-connect-nomad-demo
cd consul-connect-nomad-demo
./apps/build.sh
```

### Local Run

Run the apps locally to prove they work, in two separate terminals:

```bash
PORT=9001 ./apps/bin/counter
```

```bash
PORT=9002 ./apps/bin/dashboard
```

Open `http://localhost:9002` to see the counter running.

### Start A Cluster

Now we have established our apps actually start, we can create a small Consul cluster.  I am using my Hashibox to do this, so you'll need libvirt and Vagrant installed to do this.

Running `vagrant up` will spawn three machines, which will form a Consul cluster, which we can now experiment in.  Once it is up and running, we can manually register the two applications into Consul's service mesh to check that our in cluster communication works.

First, the counter service.  The script writes a service definition into consul, which, by specifying the `connect` stanza, indicates this service is to be included in the service mesh.  Once this is done, the counter is started (and sent to the background), and a consul connect proxy is started for this service:

```bash
curl --request PUT --url http://localhost:8500/v1/agent/service/register \
  --data '{
    "name": "counter",
    "port": 9001,
    "connect": {
      "sidecar_service": {}
    }
  }'

PORT=9001 /vagrant/apps/bin/counter &

consul connect proxy -sidecar-for counter
```

We can run this script in a new terminal by running this command:

```bash
vagrant ssh one -c '/vagrant/scripts/counter.sh'
```

Finally, we start the dashboard.  The script is very similar, in that we write a service definiton into consul, start the service and run a proxy.  The only notable difference is the service registation payload itself:

```json
{
  "name": "dashboard",
  "port": 9002,
  "connect": {
    "sidecar_service": {
      "proxy": {
        "upstreams": [
          { "destination_name": "counter", "local_bind_port": 8080 }
        ]
      }
    }
  }
}
```

As before, it registers a service, and on what port it will be listening on, but in the `connect` stanza, we specify that we want to connect to the `counter`, and we want to talk to it on `localhost:8080`.

In a new terminal, you can run this script like so:

```bash
vagrant ssh two -c '/vagrant/scripts/dashboard.sh'
```

Now that both are up and running, you can open a browser to the dashboard and see it working: `http://two.karhu.xyz:9002`.  Once you are satisfied, you can stop the services by hitting `ctrl+c` in both terminals...or try running a second counter or dashboard on the third vagrant machine (`vagrant ssh three -c '/vagrant/scripts/dashboard.sh'`)

### Nomad

Now that we have seen how to run the services manually let's see how easy it is to use the service mesh using [Nomad](https://nomadproject.io).

There are two nomad job definitions in the included project, so let's look at the counter's first:

```go
job "counter" {
  datacenters = ["dc1"]

  group "api" {
    count = 3

    network {
      mode = "bridge"
    }

    service {
      name = "count-api"
      port = "9001"

      connect {
        sidecar_service {}
      }
    }

    task "counter" {
      driver = "exec"

      config {
        command = "/vagrant/apps/bin/counter"
      }

      env {
        PORT = 9001
      }
    }
  }
}
```

The `network` stanza is set to `bridge` mode, which creates us an isolated network between all the services in the group only.  In our case, we will have a single `counter` service and the proxy.

The `service` stanza is replicating the same functionality we had by writing a service registration into Consul.  By specifying the `connect` part, Nomad knows that it also needs to start a proxy-based on the service stanza's settings, and will handle starting and stopping this proxy for us.

The `task "counter"` block uses the `exec` driver to run the counter app natively on the host, but `docker`, `java`, and others are available too.

To run this into our Nomad cluster, we can use the nomad CLI:

```bash
export NOMAD_ADDR="http://one.karhu.xyz:4646"

nomad job run apps/counter/counter.nomad
```

The dashboard's Nomad job is very similar:

```go
job "dashboard" {
  datacenters = ["dc1"]

  group "dashboard" {
    network {
      mode = "bridge"

      port "http" {
        to     = 9002
      }
    }

    service {
      name = "count-dashboard"
      port = 9002

      connect {
        sidecar_service {
          proxy {
            upstreams {
              destination_name = "count-api"
              local_bind_port  = 8080
            }
          }
        }
      }
    }

    task "dashboard" {
      driver = "exec"

      config {
        command = "/vagrant/apps/bin/dashboard"
      }

      env {
        PORT = "${NOMAD_PORT_http}"
        COUNTING_SERVICE_URL = "http://${NOMAD_UPSTREAM_ADDR_count_api}"
      }
    }
  }
}
```

The `network` block this time also specifies that we want to expose our service to the public.  As we don't have a `static = 9002` in the port definition, Nomad will assign one at random (this is better! You can avoid port clashes with multiple tasks on the same node), we do however specify that we will map to `9002`.  The rest of the file can use the Nomad variable `NOMAD_PORT_http` to get this port number, so we don't have to copy-paste the number everywhere.  Similarly, the `sidecar_service` stanza exposes a variable called `NOMAD_UPSTREAM_ADDR_<destination_name>`, so we can use that too for our dashboard task's environment variable values. This means we should only ever need to specify ports in 1 location in a Nomad file.

As with the counter, we can run the job using the CLI:

```bash
nomad job run apps/counter/dashboard.nomad
```

If we want to get the address and port the dashboard is actually running at, it is easiest to go through the UI, but you can also get the information from the console using the Nomad CLI and jq:

```bash
allocation_id=$(nomad alloc status -json | jq -r '.[] | select(.JobID == "dashboard") | .ID')

nomad alloc status -json "$allocation_id" \
  | jq -r '.AllocatedResources.Shared.Networks[0] | ( "http://" + .IP + ":" + (.DynamicPorts[] | select(.Label == "http") | .Value | tostring))'
```

## Wrapping Up

With Consul Connect's supported APIs, there is great flexibility in how you can implement your service mesh; through definition files, through API requests, or through the container orchestrator directly.  Couple this with Consul already being in use in most systems I have been involved with, and hopefully you can see why it makes a great way of having a Service Mesh.