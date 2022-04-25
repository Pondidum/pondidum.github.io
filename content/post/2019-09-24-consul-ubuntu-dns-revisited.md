---
date: "2019-09-24T00:00:00Z"
tags: ["consul", "dns", "infrastructure"]
title: Consul DNS Fowarding in Ubuntu, revisited
---

I was recently using my [Hashibox](https://github.com/pondidum/hashibox) for a test, and I noticed the DNS resolution didn't seem to work.  This was a bit worrying, as I have written about how to do [DNS resolution with Consul forwarding in Ubuntu](/2019/05/29/consul-dns-forwarding/), and apparently something is wrong with how I do it.  Interestingly, the [Alpine version](/2019/05/31/consul-dns-forwarding-alpine/) works fine, so it appears there is something not quite working with how I am configuring Systemd-resolved.

So this post is how I figured out what was wrong, and how to do DNS resolution with Consul forwarding on Ubuntu properly!

## The Problem

If Consul is running on the host, I can only resolve `.consul` domains, and if Consul is not running, I can resolve anything else.  Clearly I have configured something wrong!

To summarise, I want to be able to resolve 3 kinds of address:

* `*.consul` addresses should be handled by the local Consul instance
* `$HOSTNAME.mshome.net` should be handled by the Hyper-V DNS server (running on the Host machine)
* `reddit.com` public DNS should be resolved properly

## Discovery

To make sure that hostname resolution even works by default, I create a blank Ubuntu box in Hyper-V, using [Vagrant](https://www.vagrantup.com/).

```ruby
Vagrant.configure(2) do |config|
  config.vm.box = "bento/ubuntu-16.04"
  config.vm.hostname = "test"
end
```

I set the hostname so that I can test that dns resolution works from the host machine to the guest machines too.  I next bring up the machine, SSH into it, and try to `dig` my hostmachine's DNS name (`spectre.mshome.net`):

```bash
> vagrant up
> vagrant ssh
> dig spectre.mshome.net

; <<>> DiG 9.10.3-P4-Ubuntu <<>> spectre.mshome.net
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 12333
;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 0

;; QUESTION SECTION:
;spectre.mshome.net.            IN      A

;; ANSWER SECTION:
Spectre.mshome.net.     0       IN      A       192.168.181.161

;; Query time: 0 msec
;; SERVER: 192.168.181.161#53(192.168.181.161)
;; WHEN: Mon Sep 23 21:57:26 UTC 2019
;; MSG SIZE  rcvd: 70

> exit
> vagrant destroy -f
```

As you can see, the host machine's DNS server responds with the right address.  Now that I know that this should work, we can tweak the `Vagrantfile` to start an instance of my Hashibox:

```ruby
Vagrant.configure(2) do |config|
  config.vm.box = "pondidum/hashibox"
  config.vm.hostname = "test"
end
```

When I run the same command sin this box, I get a slighty different response:

```bash
; <<>> DiG 9.10.3-P4-Ubuntu <<>> spectre.mshome.net
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NXDOMAIN, id: 57216
;; flags: qr aa rd; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 1
;; WARNING: recursion requested but not available

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 4096
;; QUESTION SECTION:
;spectre.mshome.net.            IN      A

;; AUTHORITY SECTION:
consul.                 0       IN      SOA     ns.consul. hostmaster.consul. 1569276784 3600 600 86400 0

;; Query time: 1 msec
;; SERVER: 127.0.0.1#53(127.0.0.1)
;; WHEN: Mon Sep 23 22:13:04 UTC 2019
;; MSG SIZE  rcvd: 103
```

As intended, the DNS server on localhost responded...but it looks like Consul answered, not the inbuilt dns server (`systemd-resolved`), as I intended.

The reason for this is that I am running Consul's DNS endpoint on `8600`, and Systemd-Resolved cannot send requests to anything other than port `53`, so I use `iptables` to redirect the traffic from port `53` to `8600`, which means any local use of DNS will always be sent to Consul.

The reason it works when Consul is not running is that we have both `127.0.0.1` specified as a nameserver, and a fallback set to be the `eth0`'s Gateway, so when Consul doesn't respond, the request hits the default DNS instead.

## The Solution: Dnsmasq.

Basically, stop using `systemd-resolved` and use something that has a more flexible configuration.  Enter Dnsmasq.

Starting from the blank Ubuntu box, I install dnsmasq, and disable systemd-resolved.  Doing this might prevent any DNS resolutio working for a while...

```bash
sudo apt-get install -yq dnsmasq
sudo systemctl disable systemd-resolved.service
```

If you would rather not disable `systemd-resolved` entirely, you can use these two lines instead to just switch off the local DNS stub:

```bash
echo "DNSStubListener=no" | sudo tee --append /etc/systemd/resolved.conf
sudo systemctl restart systemd-resolved
```

Next I update `/etc/resolv.conf` to not be managed by Systemd, and point to where dnsmasq will be running:

```bash
sudo rm /etc/resolv.conf
echo "nameserver 127.0.0.1" | sudo tee /etc/resolv.conf
```

The reason for deleting the file is that it was symlinked to the Systemd-Resolved managed file, so that link needed to be broken first to prevent Systemd interfering.

Lastly a minimal configuration for dnsmasq:

```bash
echo '
port=53
resolv-file=/var/run/dnsmasq/resolv.conf
bind-interfaces
listen-address=127.0.0.1
server=/consul/127.0.0.1#8600
' | sudo tee /etc/dnsmasq.d/default

sudo systemctl restart dnsmasq
```

This config does a few things, the two most important lines are:

* `resolv-file=/var/run/dnsmasq/resolv.conf` which is pointing to the default `resolv.conf` written by dnsmasq.  This file contains the default nameserver supplied by the default network connection, and I want to use this as a fallback for anything dnsmasq cannot resolve directly (which will be everything, except `.consul`).  In my case, the content of this file is just `nameserver 192.168.181.161`.

* `server=/consul/127.0.0.1#8600` specifies that any address ending in `.consul` should be forwarded to Consul, running at `127.0.0.1` on port `8600`.  No more `iptables` rules!

## Testing

Now that I have a (probably) working DNS system, let's look at testing it properly this time.  There are 3 kinds of address I want to test:

* Consul resolution, e.g. `consul.service.consul` should return the current Consul instance address.
* Hostname resolution, e.g. `spectre.mshome.net` should resolve to the machine hosting the VM.
* Public resolution, e.g. `reddit.com` should resolve to...reddit.

I also want to test that the latter two cases work when Consul is **not** running too.

So let's write a simple script to make sure these all work.  This way I can reuse the same script on other machines, and also with other VM providers to check DNS works as it should.  The entire script is here:

{% raw %}
```bash
local_domain=${1:-mshome.net}
host_machine=${2:-spectre}

consul agent -dev -client 0.0.0.0 -bind '{{ GetInterfaceIP "eth0" }}' > /dev/null &
sleep 1

consul_ip=$(dig consul.service.consul +short)
self_ip=$(dig $HOSTNAME.$local_domain +short | tail -n 1)
host_ip=$(dig $host_machine.$local_domain +short | tail -n 1)
reddit_ip=$(dig reddit.com +short | tail -n 1)

kill %1

[ "$consul_ip" == "" ] && echo "Didn't get consul ip" >&2 && exit 1
[ "$self_ip" == "" ] && echo "Didn't get self ip" >&2 && exit 1
[ "$host_ip" == "" ] && echo "Didn't get host ip" >&2 && exit 1
[ "$reddit_ip" == "" ] && echo "Didn't get reddit ip" >&2 && exit 1

echo "==> Consul Running: Success!"

consul_ip=$(dig consul.service.consul +short | tail -n 1)
self_ip=$(dig $HOSTNAME.$local_domain +short | tail -n 1)
host_ip=$(dig $host_machine.$local_domain +short | tail -n 1)
reddit_ip=$(dig reddit.com +short | tail -n 1)

[[ "$consul_ip" != *";; connection timed out;"* ]] && echo "Got a consul ip ($consul_ip)" >&2 && exit 1
[ "$self_ip" == "" ] && echo "Didn't get self ip" >&2 && exit 1
[ "$host_ip" == "" ] && echo "Didn't get host ip" >&2 && exit 1
[ "$reddit_ip" == "" ] && echo "Didn't get reddit ip" >&2 && exit 1

echo "==> Consul Stopped: Success!"

exit 0
```
{% endraw %}

What this does is:

1. Read two command line arguments, or use defaults if not specified
1. Start Consul as a background job
1. Query 4 domains, storing the results
1. Stop Consul (`kill %1`)
1. Check an IP address came back for each domain
1. Query the same 4 domains, storing the results
1. Check that a timeout was received for `consul.service.consul`
1. Check an IP address came back for the other domains


To further prove that dnsmasq is forwarding requests correctly, I can include two more lines to `/etc/dnsmasq.d/default` to enable logging, and restart dnsmasq

```bash
echo "log-queries" | sudo tee /etc/dnsmasq.d/default
echo "log-facility=/var/log/dnsmasq.log" | sudo tee /etc/dnsmasq.d/default
sudo systemctl restart dnsmasq
dig consul.service.consul
```

Now I can view the log file and check that it received the DNS query and did the right thing.  In this case, it recieved the `consul.service.consul` query, and forwarded it to the local Consul instance:

```
Sep 24 06:30:50 dnsmasq[13635]: query[A] consul.service.consul from 127.0.0.1
Sep 24 06:30:50 dnsmasq[13635]: forwarded consul.service.consul to 127.0.0.1
Sep 24 06:30:50 dnsmasq[13635]: reply consul.service.consul is 192.168.181.172
```

I don't tend to keep DNS logging on in my Hashibox as the log files can grow very quickly.

## Wrapping Up

Now that I have proven my DNS resolution works (I think), I have rolled it back into my Hashibox, and can now use machine names for setting up clusters, rather than having to specify IP addresses initially.
