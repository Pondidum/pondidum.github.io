---
date: "2019-11-21T00:00:00Z"
tags: ["nomad", "infrastructure", "kubernetes"]
title: Nomad Good, Kubernetes Bad
---

I will update this post as I learn more (both positive and negative), and is here to be linked to when people ask me why I don't like Kubernetes, and why I would pick Nomad in most situations if I chose to use an orchestrator *at all*.

TLDR: I don't like complexity, and Kubernetes has more complexity than benefits.

### Operational Complexity

Operating Nomad is very straight forward.  There are very few moving parts, so the number of things which can go wrong is significantly reduced.  No external dependencies are required to run it, and there is only one binary to use.  You run 3-5 copies in Server mode to manage the cluster and as many as you want running in Client mode to do the actual work.  You can add Consul if you want service discovery, but it's optional.  More on that later.

Compare this to operating a Kubernetes cluster.  There are multiple Kubernetes orchestration projects, tools, and companies to get clusters up and running, which should be an indication of the level of complexity involved.  Once you have the cluster set up, you need to keep it running.  There are so many moving parts (Controller Manager, Scheduler, API Server, Etcd, Kubelets) that it quickly becomes a full-time job to keep the cluster up and running.  Use a cloud service to run Kubernetes, and if you must use your own infrastructure, pay someone else to manage it.  It's cheaper in the long run. Trust me.

### Deployment

Nomad, being a single binary, is easy to deploy.  If you want to use [Terraform](https://www.terraform.io/) to create a cluster, Hashicorp provides modules for both [AWS](https://github.com/hashicorp/terraform-aws-nomad) and [Azure](https://github.com/hashicorp/terraform-azurerm-nomad).  Alternatively, you can do everything yourself, as it's just keeping one binary running on hosts, and a bit of network/DNS config to get them talking to each other.

By comparison, Kubernetes has a multitude of tools to help you deploy a cluster. Still, while it gives you a lot of flexibility in choice, you also have to hope that the tool continues to exist and that there is enough community/company/documentation about that specific tool to help you when something goes wrong.

### Upgrading The Cluster

Upgrading Nomad involves doing a rolling deployment of the Servers and Clients.  If you are using the Hashicorp Terraform module, you re-apply the module with the new AMI ID to use, and then delete nodes (gracefully!) from the cluster and let the AutoScaleGroup take care of bringing new nodes up.  If you need to revert to an older version of Nomad, you follow the same process.

When it comes to Kubernetes, please pay someone else to do it.  It's not a fun process.  The process will differ depending on which cluster management tool you are using, and you also need to think about updates to etcd and managing state in the process.  There is a [nice long document](https://kubernetes.io/docs/tasks/administer-cluster/configure-upgrade-etcd/) on how to upgrade etcd.

### Debugging a Cluster

As mentioned earlier, Nomad has a small number of moving parts.  There are three ports involved (HTTP, RPC and Gossip), so as long as those ports are open and reachable, Nomad should be operable.  Then you need to keep the Nomad agents alive.  That's pretty much it.

Where to start for Kubernetes? As many [Kubernetes Failure Stories](https://github.com/hjacobs/kubernetes-failure-stories) point out: it's always DNS. Or etcd. Or Istio. Or networking. Or Kubelets. Or all of these.

### Local Development

To run Nomad locally, you use the same binary as the production clusters, but in dev mode: `nomad agent -dev`.  To get a local cluster, you can spin up some Vagrant boxes instead.  I use my [Hashibox](https://github.com/pondidum/hashibox) Vagrant box to do this when I do conference talks and don't trust the wifi to work.

To run Kubernetes locally to test things, you need to install/deploy MiniKube, K3S, etc.  The downside to this approach is that the environment is significantly different to your real Kubernetes cluster, and you can end up where a deployment works in one, but not the other, which makes debugging issues much harder.

### Features & Choice

Nomad is relatively light on built-in features, which allows you the choice of what features to add, and what implementations of the features to use.  For example, it is pretty popular to use Consul for service discovery, but if you would rather use [Eureka](https://github.com/Netflix/eureka), or Zookeeper, or even etcd, that is fine, but you lose out on the seamless integration with Nomad that other Hashicorp tools have.  Nomad also supports [Plugins](https://www.nomadproject.io/docs/internals/plugins/index.html) if you want to add support for your favourite tool.

By comparison, Kubernetes does everything, but like the phrase "Jack of all trades, master of none", often you will have to supplement the inbuilt features.  The downside to this is that you can't switch off Kubernetes features you are not using, or don't want.  So if you add Vault for secret management, the Kubernetes Secrets are still available, and you have to be careful that people don't use them accidentally.  The same goes for all other features, such as Load Balancing, Feature Toggles, Service Discovery, DNS, etc.

### Secret Management

Nomad doesn't provide a Secret Management solution out of the box, but it does have seamless Vault integration, and you are also free to use any other Secrets As A Service tool you like.  If you do choose Vault, you can either use it directly from your tasks or use Nomad's integration to provide the secrets to your application.  It can even send a signal (e.g. `SIGINT` etc.) to your process when the secrets need re-reading.

Kubernetes, on the other hand, provides "Secrets".  I put the word "secrets" in quotes because they are not secrets at all. The values are stored encoded in base64 in etcd, so anyone who has access to the etcd cluster has access to *all* the secrets.  The [official documentation](https://kubernetes.io/docs/concepts/configuration/secret/#risks) suggests making sure only administrators have access to the etcd cluster to solve this.  Oh, and if you can deploy a container to the same namespace as a secret, you can reveal it by writing it to stdout.

> Kubernetes secrets are not secret, just "slightly obscured."

If you want real Secrets, you will almost certainly use Vault.  You can either run it inside or outside of Kubernetes, and either use it directly from containers via it's HTTPS API or use it to populate Kubernetes Secrets.  I'd avoid populating Kubernetes Secrets if I were you.

### Support

If Nomad breaks, you can either use community support or if you are using the Enterprise version, you have Hashicorp's support.

When Kubernetes breaks, you can either use community support or find and buy support from a Kubernetes management company.

The main difference here is "when Kubernetes breaks" vs "if Nomad breaks".  The level of complexity in Kubernetes makes it far more likely to break, and that much harder to debug.
