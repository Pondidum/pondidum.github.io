+++
date = '2019-05-31T00:00:00Z'
tags = ['infrastructure', 'consul', 'alpine', 'dns']
title = 'Configuring Consul DNS Forwarding in Alpine Linux'

+++

# DEPRECATED - This has a race condition!

[Please see this post for an updated version which works!](/2019/12/30/consul-alpine-dns-revisited/)

Following on from the post the other day on setting up [DNS forwarding to Consul with SystemD](/2019/05/29/consul-dns-forwarding/), I wanted also to show how to get Consul up and running under [Alpine Linux](https://www.alpinelinux.org/), as it's a little more awkward in some respects.

To start with, I am going to setup Consul as a service - I didn't do this in the Ubuntu version, as there are plenty of useful articles about that already, but that is not the case with Alpine.

## Run Consul

First, we need to get a version of Consul and install it into our system.  This script downloads `1.5.1` from Hashicorp's releases site, installs it to `/usr/bin/consul`, and creates a `consul` user and group to run the daemon with:

```shell
CONSUL_VERSION=1.5.1

curl -sSL https://releases.hashicorp.com/consul/${CONSUL_VERSION}/consul_${CONSUL_VERSION}_linux_amd64.zip -o /tmp/consul.zip

unzip /tmp/consul.zip
sudo install consul /usr/bin/consul

sudo addgroup -S consul
sudo adduser -S -D -h /var/consul -s /sbin/nologin -G consul -g consul consul

```

Next, we need to create the directories for the configuration and data to live in, and copy the init script and configuration file to those directories:

```shell
consul_dir=/etc/consul
data_dir=/srv/consul

sudo mkdir $consul_dir
sudo mkdir $data_dir
sudo chown consul:consul $data_dir

sudo mv /tmp/consul.sh /etc/init.d/consul
sudo chmod +x /etc/init.d/consul

sudo mv /tmp/consul.json $consul_dir/consul.json
```

The init script is pretty straight forward, but note that I am running the agent in this example in `dev` mode; **don't do this in production**:

```shell
#!/sbin/openrc-run
CONSUL_LOG_FILE="/var/log/${SVCNAME}.log"

name=consul
description="A tool for service discovery, monitoring and configuration"
description_checkconfig="Verify configuration file"
daemon=/usr/bin/$name
daemon_user=$name
daemon_group=$name
consul_dir=/etc/consul
extra_commands="checkconfig"

start_pre() {
    checkpath -f -m 0644 -o ${SVCNAME}:${SVCNAME} "$CONSUL_LOG_FILE"
}

depend() {
    need net
    after firewall
}

checkconfig() {
    consul validate $consul_dir
}

start() {
    checkconfig || return 1

    ebegin "Starting ${name}"
        start-stop-daemon --start --quiet \
            -m --pidfile /var/run/${name}.pid \
            --user ${daemon_user} --group ${daemon_group} \
            -b --stdout $CONSUL_LOG_FILE --stderr $CONSUL_LOG_FILE \
            -k 027 --exec ${daemon} -- agent -dev -config-dir=$consul_dir
    eend $?
}

stop() {
    ebegin "Stopping ${name}"
        start-stop-daemon --stop --quiet \
            --pidfile /var/run/${name}.pid \
            --exec ${daemon}
    eend $?
}
```

Finally, a basic config file to launch consul is as follows:

```json
{
    "data_dir": "/srv/consul/data",
    "client_addr": "0.0.0.0"
}
```

Now that all our scripts are in place, we can register Consul into the service manager, and start it:

```shell
sudo rc-update add consul
sudo rc-service consul start
```

You can check consul is up and running by using `dig` to get the address of the consul service itself:

```bash
dig @localhost -p 8600 consul.service.consul
```

## Setup Local DNS with Unbound

Now that Consul is running, we need to configure a local DNS resolver to forward requests for the `.consul` domain to Consul.  We will use [Unbound](https://nlnetlabs.nl/projects/unbound/about/) as it works nicely on Alpine.  It also has the wonderful feature of being able to send queries to a specific port, so no `iptables` rules needed this time!

The config file (`/etc/unbound/unbound.conf`) is all default values, with the exception of the last 5 lines, which let us forward DNS requests to a custom, and insecure, location:

```shell
#! /bin/bash

sudo apk add unbound

(
cat <<-EOF
server:
    verbosity: 1
    root-hints: /etc/unbound/root.hints
    trust-anchor-file: "/usr/share/dnssec-root/trusted-key.key"
    do-not-query-localhost: no
    domain-insecure: "consul"
stub-zone:
    name: "consul"
    stub-addr: 127.0.0.1@8600
EOF
) | sudo tee /etc/unbound/unbound.conf

sudo rc-update add unbound
sudo rc-service unbound start
```

We can validate this works again by using `dig`, but this time removing the port specification to hit `53` instead:

```bash
dig @localhost consul.service.consul
```

## Configure DNS Resolution

Finally, we need to update `/etc/resolv.conf` so that other system tools such as `ping` and `curl` can resolve `.consul` addresses.  This is a little more hassle on Alpine, as there are no `head` files we can push our nameserver entry into.  Instead, we use `dhclient` which will let us prepend a custom nameserver (or multiple) when the interface is brought up, even when using DHCP:

```shell
#! /bin/bash

sudo apk add dhclient

(
cat <<-EOF
option rfc3442-classless-static-routes code 121 = array of unsigned integer 8;
send host-name = gethostname();
request subnet-mask, broadcast-address, time-offset, routers,
        domain-name, domain-name-servers, domain-search, host-name,
        dhcp6.name-servers, dhcp6.domain-search, dhcp6.fqdn, dhcp6.sntp-servers,
        netbios-name-servers, netbios-scope, interface-mtu,
        rfc3442-classless-static-routes, ntp-servers;
prepend domain-name-servers 127.0.0.1;
EOF
) | sudo tee /etc/dhcp/dhclient.conf

sudo rm /etc/resolv.conf # hack due to it dhclient making an invalid `chown` call.
sudo rc-service networking restart
```

The only thing of interest here is the little hack: we delete the `/etc/resolv.conf` before restarting the networking service, as if you don't do this, you get errors about "chmod invalid option resource=...".

We can varify everything works in the same way we did on Ubuntu; `curl` to both a `.consul` and a public address:

```bash
$ curl -s -o /dev/null -w "%{http_code}\n" http://consul.service.consul:8500/ui/
200

$ curl -s -o /dev/null -w "%{http_code}\n" http://google.com
301
```

## End

This was a bit easier to get started with than the Ubuntu version as I knew what I was trying to accomplish this time - however making a good `init.d` script was a bit more hassle, and the error from `chmod` took some time to track down.