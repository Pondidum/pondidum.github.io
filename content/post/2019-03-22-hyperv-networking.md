---
date: "2019-03-22T00:00:00Z"
tags: vagrant docker hyperv networking
title: Hyper-V, Docker, and Networking Drama
---

I had a major problem a few hours before giving my [Nomad: Kubernetes Without the Complexity](https://andydote.co.uk/presentations/index.html?nomad) talk this morning: the demo stopped working.

Now, the first thing to note is the entire setup of the demo [is scripted](https://github.com/pondidum/nomad-demo), and the scripts hadn't changed.  The only thing I had done was restart the machine, and now things were breaking.

## The Symptoms

A docker container started inside the guest VMs with a port mapped to the machine's public IP wasn't resolvable outside the host.

For example, using a machine based off the `bento/ubuntu-16.04` base box, provisioned with docker, running this from inside an SSH connection to the machine would work:


```bash
vagrant ssh

# launch a container which can respond to a http get
docker run -d --rm -p 172.127.48.105:5000:5000 registry:latest

# curl by public ip
curl http://172.127.48.105:5000 --silent -w "%{http_code}"   # 200
```

But running the same `curl` command on the host would fail:

```bash
# container is still running
curl http://172.127.48.105:5000 --silent -w "%{http_code}"   # timeout
```


## Investigation

So it's 5 hours before the demo (thankfully it's not 10 minutes before), so let's start digging into what could be causing this.

## Docker Networking

I also was searching for Nomad and Docker networking issues - as I figured I could change the Nomad job to bind the container to all interfaces (e.g. `-p 5000:5000`) instead of just the one IP.  [This reply](https://github.com/hashicorp/nomad/issues/209#issuecomment-145313928) mentioned the `docker0` network, and when I checked the guest machines, I saw that this network is also in the `172.*` range.

So my guest machines had public addresses which happened to fall in the same range as a separate network adaptor on that machine.

## Hyper-V IP Addresses

While I was checking the Windows Firewall to see if anything was weird in there, I stumbled across a rule I'd added to allow exposure of a NodeJS service from my host to Hyper-v guests (but not anywhere else).  I noticed that the IP range it defined was `192.168.*`, and I now had machines with `172.*` addresses.

So the IP address range for guest machines had changed.


## The Solution

Luckily, there is a straightforward solution to this:

**Reboot until you get the range you want**

Really.

The other solution is to use an External Switch in Hyper-V and bridge it with your host's internet connection, which doesn't really help me, as I am on a laptop, on different WiFi networks, and sometimes I use a thunderbolt based network adaptor too.  And having to update/rebuild machines on every network change would be an absolute pain.

So I rebooted — a lot.

So if anyone from Microsoft is reading this: Please let us configure the Default Switch.  Or have a way to recreate it without rebooting at least.
