---
date: "2021-06-02T00:00:00Z"
tags: kubernetes nodejs infrastructure
title: The Problem with CPUs and Kubernetes
---

## Key Takeaway:

> `os .cpus()` returns the number of cores on a Kubernetes host, not the number of cores assigned to a pod.


## Investigating excessive memory usage

Recently, when I was looking through a cluster health dashboard for a Kubernetes cluster, I noticed that one of the applications deployed was using a considerable amount of RAM - way more than I thought could be reasonable.  Each instance (pod) of the application used approximately 8 GB of RAM, which was definitely excessive for a reasonably simple NodeJS webserver.  Combined with the application running 20-30 replicas or so, it makes the total RAM usage between 160 GB and 240 GB.

One of the first things I noticed was that the deployment manifest in Kubernetes had the `NODE_MAX_MEM` environment variable specified and set to 250 MB:

```yaml
environment:
  NODE_MAX_MEM: 250
```

_Interesting_.  So how is a single container using more RAM than that?

The application used to be deployed to EC2 machines and to fully utilise the multiple cores in the machines, the [cluster](https://www.npmjs.com/package/cluster) library was used.

This library essentially forks the node process into `n` child processes, and in this case, `n` was set to `os.cpus()`, which returns the number of cores available on the machine in NodeJS.

While this works for direct virtual machine usage, when the application was containerised and deployed to Kubernetes, it used about the same amount of ram as before, so no one realised there was a problem.

## os.cpus() and Kubernetes

The interesting thing about `os.cpus()` when called in a container in Kubernetes is that it reports the number of cores available on the host machine, not the amount of CPU assigned to the container (e.g. through resource requests and limits).

So every replica for the application spawns 32 child processes, as our EC2 hosts have that many cores.  As they had a limited per-pod CPU budget, was there any benefit to doing this?

So I did what seemed natural - I replaced `os.cpus()` with `1`, and deployed the application to production, and watched the performance metrics to see what happened.

And what do you know? No difference in request performance _at all_ - and the memory usage dropped by 7.75 GB per pod.

This means overall, we have saved 155 GB to 232.5 GB of RAM, with no performance difference!