---
date: "2019-12-22T00:00:00Z"
tags: libvirt vagrant dns
title: Libvirt Hostname Resolution
---

I use [Vagrant](http://vagrantup.com/) when testing new machines and experimenting locally with clusters, and since moving (mostly) to Linux, I have been using the [LibVirt Plugin](https://github.com/vagrant-libvirt/vagrant-libvirt) to create the virtual machines. Not only is it significantly faster than Hyper-V was on windows, but it also means I don't need to use Oracle products, so it's win-win really.

The only configuration challenge I have had with it is setting up VM hostname resolution, and as I forget how to do it each time, I figured I should write about it.

## Setup

First I install the plugin so Vagrant can talk to Libvirt.

```bash
vagrant plugin install vagrant-libvirt
```

I also created a single `vagrantfile` with two virtual machines defined in it, so that I can check that the machines can resolve each other, as well as the host being able to resolve the guests.

```ruby
Vagrant.configure("2") do |config|
 config.vm.box = "elastic/ubuntu-16.04-x86_64"

 config.vm.define "one" do |n1|
 n1.vm.hostname = "one"
 end

 config.vm.define "two" do |n1|
 n1.vm.hostname = "two"
 end
end
```

Once running `vagrant up` has finished (either with `--provider libvirt` or setting ` VAGRANT_DEFAULT_PROVIDER=libvirt`), connect to one of the machines, and try to ping the other:

```bash
andy@karhu$ vagrant ssh one
vagrant@one$ ping two
ping: unknown host two
vagrant@one$ exit
```

Now that we can see they can't resolve each other let's move on to fixing it.


## Custom Domain

The solution is to configure the libvirt network to have a domain name, and then to set the host machine to send requests for that domain to the virtual network.

First, I picked a domain. It doesn't matter what it is, but I gather using `.local` will cause problems with other services, so instead, I picked `$HOSTNAME.xyz`, which is `karhu.xyz` in this case.

Vagrant-libvirt by default creates a network called `vagrant-libvirt`, so we can edit it to include the domain name configuration by running the following command:

```bash
virsh net-edit --network vagrant-libvirt
```

And adding the `<domain name='karhu.xyz' localOnly='yes' /> line to the xml which is displayed:

```diff
<network ipv6='yes'>
 <name>vagrant-libvirt</name>
 <uuid>d265a837-96fd-41fc-b114-d9e076462051</uuid>
 <forward mode='nat'/>
 <bridge name='virbr1' stp='on' delay='0'/>
 <mac address='52:54:00:a0:ae:fd'/>
+ <domain name='karhu.xyz' localOnly='yes'/>
 <ip address='192.168.121.1' netmask='255.255.255.0'>
 <dhcp>
 <range start='192.168.121.1' end='192.168.121.254'/>
 </dhcp>
 </ip>
</network>
```

To make the changes take effect, we need to destroy and re-create the network, so first I destroy the vagrant machines, then destroy and restart the network:

```bash
vagrant destroy -f
virsh net-destroy --network vagrant-libvirt
virsh net-start --network vagrant-libvirt
```

Finally, we can re-create the machines, and log in to one to check that they can resolve each other:

```bash
andy@karhu$ vagrant up
andy@karhu$ vagrant ssh one
vagrant@one$ ping two
PING two.karhu.xyz (192.168.121.243) 56(84) bytes of data.
vagrant@one$ exit
```

You can also check that the host can resolve the machine names when querying the virtual network's DNS server:

```bash
andy@karhu$ dig @192.168.121.1 +short one
> 192.168.121.50
```

## Host DNS Forwarding

The host cant talk to the machines by name still, so we need to tweak the host's DNS, which means fighting with SystemD. Luckily, we only need to forward requests to a DNS server running on port `53` - if it was on another port then replacing systemd-resolved like [my post on Consul DNS forwarding](/2019/09/24/consul-ubuntu-dns-revisited/) would be necessary.

Edit `/etc/systemd/resolved.conf` on the host, to add two lines which instruct it to send DNS requests for the domain picked earlier to the DNS server run by libvirt (dnsmasq):


```diff
[Resolve]
-#DNS=
+DNS=192.168.121.1
#FallbackDNS=
-#Domains=
+Domains=~karhu.xyz
#LLMNR=no
#MulticastDNS=no
#DNSSEC=no
#DNSOverTLS=no
#Cache=yes
#DNSStubListener=yes
#ReadEtcHosts=yes
```

Lastly, restart systemd-resolved for the changes to take effect:

```bash
systemctl restart systemd-resolved
```

Now we can resolve the guest machines by hostname at the domain we picked earlier:

```bash
andy@karhu$ ping one.karhu.xyz
PING one.karhu.xyz (192.168.121.50) 56(84) bytes of data.
```

Done!