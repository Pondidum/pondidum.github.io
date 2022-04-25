---
layout: post
title: Vagrant in the world of Docker
tags: docker vagrant testing
---

I gave a little talk at work recently on my use of Vagrant, what it is, and why it is still useful in a world full of Docker containers.

## So, What is Vagrant?

[Vagrant]() is a product by Hashicorp, and is for scripting the creation of (temporary) virtual machines.  It's pretty fast to create a virtual machine with too, as it creates them from a base image (known as a "box".)

It also supports multiple virtualisation tools, such as VirtualBox and HyperV.  If you are already using [Packer](https://www.packer.io) to create AMIs for your Amazon infrastructure, you can modify your packerfile to also output a Vagrant box.

As an example, this is a really basic VagrantFile for creating a basic Ubuntu box:

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "hashicorp/precise64"

  config.vm.provider "hyperv" do |h|
    h.vmname = "UbuntuPrecise"
    h.cpus = 2
    h.memory = 2048
  end
end
```

To create the vm, on your command line just run

```bash
vagrant up # creates the virtual machine
vagrant ssh # ssh into the virtual machine
vagrant destroy -f # destroy the virtual machine
```

## What can I use it for?

Personally I have three main uses for a Vagrant boxes; Performance/Environment Testing, Cluster Testing, and Complete Environment Setup.

### Performance and Environment Testing

When I am developing a service which will be deployed to AWS, we tend to know rougly what kind of instance it will be deployed to, for example `T2.Small`.  The code we develop local performs well...but that is on a development machine with anywhere from 4 to 16 CPU cores, and 8 to 32 GB of memory, and SSD storage.  How do you know what performance will be like when running on a 2 core, 2048 MB machine in AWS?

While you can't emulate AWS exactly, it has certainly helped us tune applications - for example modifying how many parallel messages to handle when receiving from RabbitMQ (you can see about how to configure this in my previous post [Concurrency in RabbitMQ](2017/10/11/masstransit-rabbitmq-concurrency-testing/).)

### Cluster Testing

When you want to test a service which will operate in a cluster, Vagrant comes to the rescue again - you can use the `define` block to setup multiple copies of the machine, and provide common provisioning:

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "hashicorp/precise64"
  config.vm.provision "shell", inline: <<SCRIPT
  # a bash script to setup your service might go here
SCRIPT

  config.vm.provider "hyperv" do |h|
    h.vmname = "UbuntuPrecise"
    h.cpus = 1
    h.memory = 1024
  end

  config.vm.define "first"
  config.vm.define "second"
  config.vm.define "third"
end
```

If you want to do more configuration of your separate instances, you can provider a block to do so:

```ruby
  config.vm.define "third" do |third|
    third.vm.provision "shell", inline: "./vagrant/boot-cluster.sh"
  end
```

### Complete Environment

If you're developing a microservice in an environment with many other microservies which it needs to interact with, it can be a pain to setup all the hosts and supporting infrastructure.

Instead, we can create a single base box which contains all of the setup and services, then each microservice can have a VagrantFile which is based off the base box, but also: stops the service you are developing, and starts the version which is located in the `/vagrant` share instead:

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "mycorp/complete-environment"

  config.vm.provider "hyperv" do |h|
    h.vmname = "WhateverServiceEnvironment"
    h.cpus = 4
    h.memory = 4096
  end

  # replace-service is a script which stops/removes an existing service,
  # and installs/starts a replacement. it uses a convention which expects
  # a service to have a script at `/vagrant/<name>/bin/<name>.sh`
  config.vm.provision "shell", inline: "./replace-service.sh WhateverService"

end
```

In this case, the `mycorp/complete-environment` box would have all the services installed and started, and also a script in the machine root which does all the work to replace a service with the one under development.

This base box could also be used to provide a complete testing environment too - just create a Vagrant file with no additional provisioning, and call `vagrant up`.

## Couldn't you use Docker for this instead?

Well yes, you can user Docker...but for some tasks, this is just easier.  We can also utilise Docker as both input and output for this; the base image could run docker internally to run all the services, or we could use a Packer script which would generate a Docker container of this setup **and** a vagrant box.

Just because Docker is the cool thing to be using these days, doesn't mean Vagrant doesn't have any uses any more.  Far from it!
