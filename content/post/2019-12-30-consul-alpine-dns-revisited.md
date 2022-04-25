---
date: "2019-12-30T00:00:00Z"
tags: ["consul", "dns", "infrastructure"]
title: Consul DNS Fowarding in Alpine, revisited
---

I noticed when running an Alpine based virtual machine with Consul DNS forwarding set up, that sometimes the machine couldn't resolve `*.consul` domains, but not in a consistent manner. Inspecting the logs looked like the request was being made and responded to successfully, but the result was being ignored.

After a lot of googling and frustration, I was able to track down that it's down to a difference (or optimisation) in musl libc, which glibc doesn't do. From Musl libc's [Functional differences from glibc](https://wiki.musl-libc.org/functional-differences-from-glibc.html) page, we can see under the Name Resolver/DNS section the relevant information:

> Traditional resolvers, including glibc's, make use of multiple nameserver lines in resolv.conf by trying each one in sequence and falling to the next after one times out. musl's resolver queries them all in parallel and accepts whichever response arrives first.

The machine's `/etc/resolv.conf` file has two `nameserver` specified:

```
nameserver 127.0.0.1
nameserver 192.168.121.1
```

The first is our `Unbound` instance which handles the forwarding to Consul, and the second is the DHCP set DNS server, in this case, libvirt/qemu's dnsmasq instance.

When running in a glibc based system, queries go to the first `nameserver`, and then if that can't resolve the request, it is then sent to the next `nameserver`, and so forth. As Alpine Linux uses muslc, it makes the requests in parallel and uses the response from whichever response comes back first.

![sequence diagram, showing parallel DNS requests](/images/muslc-dns.png)

When the DHCP DNS server is a network hop away, the latency involved means our resolution usually works, as the queries will hit the local DNS and get a response first. However, when the DHCP DNS is not that far away, for example when it is the DNS server that libvirt runs in the virtual network the machine is attached to, it becomes much more likely to get a response from that DNS server first, causing the failures I was seeing.

The solution to this is to change the setup so that all requests go to Unbound, which can then decide where to send them on to.  This also has the additional benefits of making all DNS requests work the same on all systems; regardless of glibc or muslc being used.

![sequence diagram, showing all DNS requests going through unbound](/images/unbound-dns.png)

## Rebuilding DNS Resolution

You can follow the same instructions in my previous [Consul DNS forwarding](/2019/05/31/consul-dns-forwarding-alpine/#run-consul) post to setup Consul, as that is already in the right state for us.

Once Consul is up and running, it's time to fix the rest of our pipeline.

### Unbound

First, install `unbound` and configure it to start on boot:

```bash
apk add unbound
rc-update add unbound
```

The unbound config file (`/etc/unbound/unbound.conf`) is almost the same as the previous version, except we also have an `include` statement, pointing to a second config file, which we will generate shortly:

```yaml
server:
 verbosity: 1
 do-not-query-localhost: no
 domain-insecure: "consul"
stub-zone:
 name: "consul"
 stub-addr: 127.0.0.1@8600
include: "/etc/unbound/forward.conf"
```

### Dhclient

Next, we install `dhclient` so that we can make use of it's hooks feature to generate our additional unbound config file.

```bash
apk add dhclient
```

Create a config file for dhclient (`/etc/dhcp/dhclient.conf`), which again is almost the same as the previous post, but this time doesn't specify `prepend domain-name-servers`:

```conf
option rfc3442-classless-static-routes code 121 = array of unsigned integer 8;
send host-name = gethostname();
request subnet-mask, broadcast-address, time-offset, routers,
 domain-name, domain-name-servers, domain-search, host-name,
 dhcp6.name-servers, dhcp6.domain-search, dhcp6.fqdn, dhcp6.sntp-servers,
 netbios-name-servers, netbios-scope, interface-mtu,
 rfc3442-classless-static-routes, ntp-servers;
```

Now we can write two hooks. The first is an enter hook, which we can use to write the `forward.conf` file out.

```bash
touch /etc/dhclient-enter-hooks
chmod +x /etc/dhclient-enter-hooks
```

The content is a single statement to write the `new_domain_name_servers` value into a `forward-zone` for unbound:

```bash
#!/bin/sh

(
cat <<-EOF
forward-zone:
 name: "."
 forward-addr: ${new_domain_name_servers}
EOF
) | sudo tee /etc/unbound/forward.conf
```

The second hook is an exit ook, which runs after dhclient has finished writing out all the files it controls (such as `/etc/resolv.conf`):

```bash
touch /etc/dhclient-exit-hooks
chmod +x /etc/dhclient-exit-hooks
```

The content is a single `sed` statement to replace the address of `nameserver` directives written to the `/etc/resolv.conf` with the unbound address:

```bash
#!/bin/sh
sudo sed -i 's/nameserver.*/nameserver 127.0.0.1/g' /etc/resolv.conf
```

It's worth noting; we could put the content of the `enter` hook into the `exit` hook if you would rather.

Finally, we can delete our current `resolv.conf` and restart the networking service:

```bash
rm /etc/resolv.conf # hack due to it dhclient making an invalid `chown` call.
rc-service networking restart
```

## Testing

We can now test that we can resolve the three kinds of address we care about:

* `dig consul.service.consul` - should return the `eth0` ip of the machine
* `dig alpinetest.karhu.xyz` - should be resolved by libvirt's dnsmasq instance
* `dig example.com` - should be resolved by an upstream DNS server

## Conculsion

This was an interesting and somewhat annoying problem to solve, but it means I have a more robust setup in my virtual machines now. It's interesting to note that if the DNS server from DHCP were not a local instance, the network latency added would make all the system function properly most of the time, as the local instance would answer before the remote instance could.
