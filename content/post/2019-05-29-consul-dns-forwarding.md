---
layout: post
title: Configuring Consul DNS Forwarding in Ubuntu 16.04
tags: infrastructure dns consul
---

# DEPRECATED - This doesn't work properly

[Please see this post for an updated version which works!](/2019/09/24/consul-ubuntu-dns-revisited/)

---

One of the advantages of using [Consul](https://www.consul.io/) for service discovery is that besides an HTTP API, you can also query it by DNS.

The DNS server is listening on port `8600` by default, and you can query both A records or SRV records from it.  [SRV](https://en.wikipedia.org/wiki/SRV_record) records are useful as they contain additional properties (`priority`, `weight` and `port`), and you can get multiple records back from a single query, letting you do load balancing client side:

```bash
$ dig @localhost -p 8600 consul.service.consul SRV +short

1 10 8300 vagrant1.node.dc1.consul.
1 14 8300 vagrant2.node.dc1.consul.
2 100 8300 vagrant3.node.dc1.consul.
```

A Records are also useful, as it means we should be able to treat services registered to Consul like any other domain - but it doesn't work:

```bash
$ curl http://consul.service.consul:8500
curl: (6) Could not resolve host: consul.service.consul
```

The reason for this is that the system's built-in DNS resolver doesn't know how to query Consul.  We can, however, configure it to forward any `*.consul` requests to Consul.


## Solution - Forward DNS queries to Consul

As I usually target Ubuntu based machines, this means configuring `systemd-resolved` to forward to Consul.  However, we want to keep Consul listening on it's default port (`8600`), and `systemd-resolved` can only forward requests to port `53`, so we need also to configure `iptables` to redirect the requests.

The steps are as follows:

1. Configure `systemd-resolved` to forward `.consul` TLD queries to the local consul agent
1. Configure `iptables` to redirect `53` to `8600`

So let's get to it!

### 1. Make iptables persistent

IPTables configuration changes don't persist through reboots, so the easiest way to solve this is with the `iptables-persistent` package.

Typically I am scripting machines (using [Packer] or [Vagrant]), so I configure the install to be non-interactive:

```bash
echo iptables-persistent iptables-persistent/autosave_v4 boolean false | sudo debconf-set-selections
echo iptables-persistent iptables-persistent/autosave_v6 boolean false | sudo debconf-set-selections

sudo DEBIAN_FRONTEND=noninteractive apt install -yq iptables-persistent
```

### 2. Update Systemd-Resolved

The file to change is `/etc/systemd/resolved.conf`.  By default it looks like this:

```conf
[Resolve]
#DNS=
#FallbackDNS=8.8.8.8 8.8.4.4 2001:4860:4860::8888 2001:4860:4860::8844
#Domains=
#LLMNR=yes
#DNSSEC=no
```

We need to change the `DNS` and `Domains` lines - either editing the file by hand, or scripting a replacement with `sed`:

```bash
sudo sed -i 's/#DNS=/DNS=127.0.0.1/g; s/#Domains=/Domains=~consul/g' /etc/systemd/resolved.conf
```

The result of which is the file now reading like this:

```conf
[Resolve]
DNS=127.0.0.1
#FallbackDNS=8.8.8.8 8.8.4.4 2001:4860:4860::8888 2001:4860:4860::8844
Domains=~consul
#LLMNR=yes
#DNSSEC=no
```

By specifying the `Domains` as `~consul`, we are telling resolvd to forward requests for the `consul` TLD to the server specified in the `DNS` line.

### 3. Configure Resolvconf too

For compatibility with some applications (e.g. `curl` and `ping`), we also need to update `/etc/resolv.conf` to specify our local nameserver.  You do this **not** by editing the file directly!

Instead, we need to add `nameserver 127.0.0.1` to `/etc/resolvconf/resolv.conf.d/head`.  Again, I will script this, and as we need `sudo` to write to the file, the easiest way is to use `tee` to append the line and then run `resolvconf -u` to apply the change:

```bash
echo "nameserver 127.0.0.1" | sudo tee --append /etc/resolvconf/resolv.conf.d/head
sudo resolvconf -u
```

### Configure iptables

Finally, we need to configure iptables so that when `systemd-resolved` sends a DNS query to localhost on port `53`, it gets redirected to port `8600`.  We'll do this for both TCP and UDP requests, and then use `netfilter-persistent` to make the rules persistent:

```bash
sudo iptables -t nat -A OUTPUT -d localhost -p udp -m udp --dport 53 -j REDIRECT --to-ports 8600
sudo iptables -t nat -A OUTPUT -d localhost -p tcp -m tcp --dport 53 -j REDIRECT --to-ports 8600

sudo netfilter-persistent save
```

## Verification

First, we can test that both Consul and Systemd-Resolved return an address for a consul service:

```bash
$ dig @localhost -p 8600 consul.service.consul +short
10.0.2.15

$ dig @localhost consul.service.consul +short
10.0.2.15
```

And now we can try using `curl` to verify that we can resolve consul domains and normal domains still:

```bash
$ curl -s -o /dev/null -w "%{http_code}\n" http://consul.service.consul:8500/ui/
200

$ curl -s -o /dev/null -w "%{http_code}\n" http://google.com
301
```

## End

There are also guides available on how to do this on [Hashicorp's website](https://learn.hashicorp.com/consul/security-networking/forwarding), covering other DNS resolvers too (such as BIND, Dnsmasq, Unbound).

