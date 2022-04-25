---
date: "2019-01-01T00:00:00Z"
tags: ["logstash", "microservices", "infrastructure", "vagrant", "packer", "aws", "testing"]
title: Testing Immutable Infrastructure
---

In my [previous post](/2018/12/22/serilog-elk-jaeger/), I glossed over one of the most important and useful parts of Immutable Infrastructure: Testability.  There are many kinds of tests we can write for our infrastructure, but they should all be focused on the machine/service and *maybe* it's nearest dependencies, [not the entire system](https://medium.com/@copyconstruct/testing-microservices-the-sane-way-9bb31d158c16).

While this post focuses on testing a full machine (both locally in a VM, and remotely as an Amazon EC2 instance), it is also possible to do most of the same kind of tests against a Docker container.  In fact, one of the tools used in this post supports building Docker containers as an output in parallel to the AMIs, so this can also assist in providing a migration path to/from Docker.

As an example, I will show how I built and tested a LogStash machine, including how to verify that the script to create the production machine is valid, that the machine itself has been provisioned correctly, and that the services inside work as expected.

I have [published all the source code](https://github.com/Pondidum/immutable-infra-testing-demo) to GitHub.  The examples in this post are all taken from the repository but might have a few bits removed just for readability.  Check the full source out if you are interested!

## Repository Structure and Tools

When it comes to building anything that you will have lots of, consistency is key to making it manageable.  To that end, I have a small selection of tools that I use, and a repository structure I try and stick to.  They are the following:

**[Vagrant](https://www.vagrantup.com/)** - This is a tool for building and managing virtual machines.  It can be backed by many different [providers](https://www.vagrantup.com/docs/providers/) such as Docker, HyperV and VirtualBox.  We'll use this to build a local Linux machine to develop and test LogStash in.  I use the HyperV provisioner, as that is what Docker For Windows also uses, and HyperV disables other virtualisation tools.

**[Packer](https://packer.io/)** - This tool provides a way to build machine images.  Where Vagrant builds running machines, Packer builds the base images for you to boot, and can build multiple different ones (in parallel) from one configuration.  We'll use this to create our AMIs (Amazon Machine Images.)

**[Jest](http://jestjs.io/)** - This is a testing framework written in (and for) NodeJS applications.  Whatever testing tool works best for your environment is what you should be using, but I use Jest as it introduces minimal dependencies, is cross-platform, and has some useful libraries for doing things like diffing json.

The repository structure is pretty simple:

- scripts/
- src/
- test/
- build.sh
- logstash.json
- package.json
- vagrantfile

The `src` directory is where our application code will live.  If the application is compiled, the output goes to the `build` directory (which is not tracked in source-control.)  The `test` directory will contain all of our tests, and the `scripts` directory will contain everything needed for provisioning our machines.

We'll describe what the use of each of these files is as we go through the next section.

## Local Development

To create our virtual machine locally, we will use [Vagrant](https://www.vagrantup.com).  To tell Vagrant how to build our machine, we need to create a `vagrantfile` in our repository, which will contain the machine details and provisioning steps.

The machine itself has a name, CPU count, and memory specified.  There is also a setting for Hyper-V which allows us to use a differencing disk, which reduces the startup time for the VM, and how much disk space it uses on the host machine.

For provisioning, we specify to run the relevant two files from the `scripts` directory.

```ruby
Vagrant.configure("2") do |config|
    config.vm.box = "bento/ubuntu-16.04"

    config.vm.provider "hyperv" do |hv|
        hv.vmname = "LogStash"
        hv.cpus = 1
        hv.memory = 2048
        hv.linked_clone = true
    end

    config.vm.provision "shell", path: "./scripts/provision.sh"
    config.vm.provision "shell", path: "./scripts/vagrant.sh"
end
```

To keep things as similar as possible between our development machine and our output AMI, I keep as much of the setup script in one file: `scripts/provision.sh`.  In the case of our LogStash setup, this means installing Java, LogStash, some LogStash plugins, and enabling the service on reboots:

```bash
#! /bin/bash

# add elastic's package repository
wget -qO - https://artifacts.elastic.co/GPG-KEY-elasticsearch | sudo apt-key add -
echo "deb https://artifacts.elastic.co/packages/6.x/apt stable main" | sudo tee -a /etc/apt/sources.list.d/elastic-6.x.list
sudo apt-get update

# install openjdk and set environment variable
sudo apt-get install openjdk-8-jre -y
JAVA=$(readlink -f $(which java) | sed "s:bin/java::")
echo "JAVA_HOME=$JAVA" | sudo tee --append /etc/environment

#install logstash and plugins
sudo apt-get install logstash -y
/usr/share/logstash/bin/logstash-plugin install logstash-filter-uuid
/usr/share/logstash/bin/logstash-plugin install logstash-filter-prune

sudo systemctl enable logstash.service
```

Vagrant will automatically mount it's working directory into the VM under the path `/vagrant`.  This means we can add a second provisioning script (`scripts/vagrant.sh`) to link the `/vagrant/src` directory to the LogStash configuration directory (`/etc/logstash/conf.d`), meaning we can edit the files on the host machine, and then restart LogStash to pick up the changes.

```bash
#! /bin/bash
sudo rm -rf /etc/logstash/conf.d
sudo ln -s /vagrant/src /etc/logstash/conf.d

sudo systemctl start logstash.service
```

Now that we have a `vagrantfile`, we can start the virtual machine with a single command.  Note, Hyper-V requires administrator privileges, so you need to run this command in an admin terminal:

```bash
vagrant up
```

After a while, your new LogStash machine will be up and running.  If you want to log into the machine and check files an processes etc., you can run the following command:

```bash
vagrant ssh
```

An argument can also be provided to the `ssh` command to be executed inside the VM, which is how I usually trigger LogStash restarts (as it doesn't seem to detect when I save the config files in the `src` directory):

```bash
vagrant ssh -c 'sudo systemctl restart logstash'
```

## Deployment

To create the deployable machine image, I use Packer.  The process is very similar to how Vagrant is used: select a base AMI, create a new EC2 machine, provision it, and save the result as a new AMI.

Packer is configured with a single json file, in this case, named `logstash.json`.  The file is split into four parts: `variables`, `builders`, `provisioners`, and `outputs`.  I won't include the `outputs` section as it's not needed when building AMIs.

### Variables

The `variables` property is for all configuration that you can pass to Packer.  Their values can come from Environment Variables, CLI parameters, Consul, Vault, [and others](https://www.packer.io/docs/templates/user-variables.html).  In the LogStash example, there are three variables:

```json
{
  "variables": {
    "aws_access_key": "",
    "aws_secret_key": "",
    "ami_users": "{{env `AMI_ACCOUNTS`}}"
  }
}
```

The `aws_access_key` and `aws_secret_key` are known names - unless we specify some value, they will automatically be read from your AWS config (in `~/.aws/`), or if running on EC2, from the EC2 machine profile.

The `ami_users` is a custom variable which will read the `AMI_ACCOUNTS` environment variable by default.  This particular one is used so that I can grant access to the resulting AMI to multiple AWS accounts, which is useful if you're running in an Organisation with multiple Accounts.  For example, if the AMI is built in a `common` account, and will be deployed into `dev`, `qa` and `prod` accounts, then you would populate the `AMI_ACCOUNTS` as a CSV of account IDs.


### Builders

Packer can build [many different kinds](https://www.packer.io/docs/builders/index.html) of machine image, but for this, we only need one: `amazon-ebs`.

```json
{
  "builders": [
    {
      "type": "amazon-ebs",
      "access_key": "{{user `aws_access_key`}}",
      "secret_key": "{{user `aws_secret_key`}}",
      "region": "eu-west-1",
      "instance_type": "t2.micro",
      "source_ami_filter": {
        "filters": {
          "virtualization-type": "hvm",
          "name": "ubuntu/images/*ubuntu-xenial-16.04-amd64-server-*",
          "root-device-type": "ebs"
        },
        "owners": ["099720109477"],
        "most_recent": true
      },
      "ssh_username": "ubuntu",
      "ami_name": "logstash {{timestamp}}",
      "ami_users": "{{user `ami_users`}}"
    },
  ]
}
```

The two most interesting properties of this are `source_ami_filter` and `ami_users`.  The `source_ami_filter` works in a very similar manner to the AWS CLI's `describe-images` `--filters` parameter, albeit in a more readable format.  In this case, I am specifying that I want an `ubuntu-xenial` base, and I want it to be an official Canonical image, so specify their Account ID as the `owner`.  I also specify the `most_recent` property, as this filter will return all versions of this AMI which Canonical publish.

The `ami_users` is what lets me grant access to the AMI from other accounts (rather than just making it public).  The property's value should be an array, but Packer is smart enough to expand the CSV in the user variable into an array for us.

### Provisioners

The `provisioners` array items are executed in the order they are specified.  To set up the machine, I use the `shell` provisioner to create a temporary directory, then the `file` provisioner to upload the files in the `src` directory to that temporary directory.  Finally a second `shell` provisioner uploads and runs the `scripts/provision.sh` and `scripts/aws.sh` files.

```json
{
  "provisioners": [
    {
      "type": "shell",
      "inline": "mkdir -p /tmp/src"
    },
    {
      "type": "file",
      "source": "./src/",
      "destination": "/tmp/src"
    },
    {
      "type": "shell",
      "scripts": ["./scripts/provision.sh", "./scripts/aws.sh"]
    }
  ]
}
```

The `aws.sh` file is very small and does roughly the same thing as the `vagrant.sh` script, but rather than symlinking the `/vagrant` directory, it moves the uploaded `src` directory into the right location for LogStash:

```bash
#! /bin/sh

sudo rm /etc/logstash/conf.d/*
sudo cp -r /tmp/src/* /etc/logstash/conf.d
```

Note that this doesn't start the LogStash service - this gets done by the UserData when we launch a new instance, as often we need to pass in additional configuration parameters, and don't want the service running until that has been done.

### Running

To create the AMI, we need to invoke packer.  If I am running packer on a remote machine via SSH, I run it inside `tmux`, so that disconnects don't fail the process:

```bash
packer build -var "ami_users=111,222,333" logstash.json
```

After a while, Packer will finish, leaving you with an output which will include the new AMI ID:

```bash
==> Builds finished. The artifacts of successful builds are:
--> amazon-ebs: AMIs were created:

eu-west-1: ami-123123123
```

We'll get back to this output later when we create a build script that will also run our tests.  Before we get to that, however, let's look at how we can write tests which target both the local Vagrant machine and the AMI too.

## Testing

To test the machines, I am using [Jest](https://jestjs.io).  There isn't anything particularly interesting going on in the `package.json`, other than a few babel packages being installed so that I can use ES6 syntax:

```json
{
  "scripts": {
    "watch": "jest --watch",
    "test": "jest "
  },
  "devDependencies": {
    "babel-core": "^6.26.3",
    "babel-jest": "^23.6.0",
    "babel-preset-env": "^1.7.0",
    "jest": "^23.6.0",
    "regenerator-runtime": "^0.13.1"
  }
}
```

### Packer Configuration Testing

There are a number of tests we can do to make sure our Packer configuration is valid before running it.  This includes things like checking the base AMI is from a whitelisted source (such as our accounts, Amazon and Canonical).  The test has to handle the possibility of multiple builders, and that some builders might not have a `source_ami_filter`.  It also handles if no owner has been specified at all, which we also consider a "bad thing":

```javascript
const ourAccounts = [ "111111", "222222", "333333", "444444" ];
const otherOwners = [ "amazon", "099720109477" /*canonical*/ ];

describe("ami builder", () => {

  it("should be based on a whitelisted owner", () => {
    const allOwners = ourAccounts.concat(otherOwners);
    const invalidOwners = owners => owners.filter(owner => !allOwners.includes(owner));

    const amisWithInvalidOwners = packer.builders
      .filter(builder => builder.source_ami_filter)
      .map(builder => ({
        name: builderName(builder),
        invalidOwners: invalidOwners(builder.source_ami_filter.owners || [ "NO OWNER SPECIFIED" ])
      }))
      .filter(builders => builders.invalidOwners.length > 0);

    expect(amisWithInvalidOwners).toEqual([]);
  });

});
```

I also test that certain variables (`ami_users`) have been defined, and have been used in the right place:

```javascript
describe("variables", () => {
  it("should have a variable for who can use the ami", () => {
    expect(packer.variables).toHaveProperty("ami_users");
  });

  it("should read ami_users from AMI_ACCOUNTS", () => {
    expect(packer.variables.ami_users).toMatch(
      /{{\s*env\s*`AMI_ACCOUNTS`\s*}}/
    );
  });
});

describe("ami builder", () => {
  it("should set the ami_user", () => {

    const invalidUsers = packer.builders
      .map(builder => ({
        name: builderName(builder),
        users: builder.ami_users || "NO USERS SPECIFIED"
      }))
      .filter(ami => !ami.users.match(/{{\s*user\s*`ami_users`\s*}}/));

    expect(invalidUsers).toEqual([]);
  });
})
```

Other tests you might want to add are that the base AMI is under a certain age, or that your AMI has certain tags included, or that it is named in a specific manner.

### Machine Testing

Machine testing is for checking that our provisioning worked successfully.  This is very useful, as subtle bugs can creep in when you don't verify what happens.

For example, a machine I built copied configuration directory to a target location but was missing the `-r` flag, so when I later added a subdirectory, the machine failed as the referenced files didn't exist.

So that the tests work with both the Vagrant and Packer built versions, we take in their address and key paths from the environment:

```javascript
import { spawnSync } from "child_process";
import { createConnection } from "net";

// figure out where to look these up
const host = process.env.LOGSTASH_ADDRESS; // e.g. "172.27.48.28";
const keyPath = process.env.LOGSTASH_KEYPATH; // ".vagrant/machines/default/hyperv/private_key";
```

We also define two helper methods: one to check if a TCP port is open, and one which uses SSH to execute a command and read the response in the machine:

```javascript
const execute = command => {
  const args = [`vagrant@${host}`, `-i`, keyPath, command];
  const ssh = spawnSync("ssh", args, { encoding: "utf8" });
  const lines = ssh.stdout.split("\n");

  if (lines[lines.length - 1] === "") {
    return lines.slice(0, lines.length - 1);
  }
  return lines;
};

const testPort = port => new Promise((resolve, reject) => {
  const client = createConnection({ host: host, port: port });

  client.on("error", err => reject(err));
  client.on("connect", () => {
    client.end();
    resolve();
  });
});
```

We can then add some tests which check the files were written to the right place, that port `5044` is open, and port `9600` is closed:

```javascript
describe("the machine", () => {

  it("should have the correct configuration", () => {
    const files = execute("find /etc/logstash/conf.d/* -type f");

    expect(files).toEqual([
      "/etc/logstash/conf.d/beats.conf",
      "/etc/logstash/conf.d/patterns/custom.txt"
    ]);
  });

  it("should be listening on 5044 for beats", () => testPort(5044));
  it("should not be listening on 9600", () => expect(testPort(9600)).rejects.toThrow("ECONNREFUSED"));
});
```

Of course, as we can execute any command inside the machine, we can check pretty much anything:

* `tail` the LogStash log and see if it's got the right contents
* check if the service is started
* check the service is enabled on boot
* check the environment variables been written to the right files

### Application Testing

There are two styles of Application Testing: white-box and black-box.  White-box will be tests run on the application inside the machine, using minimal external dependencies (preferably none at all), and Black-box will be run on the application from outside the machine, either using direct dependencies, or fakes.

It's worth noting that both white-box and black-box tests are **slow**, mostly down to how slow LogStash is at starting up, although only giving it 1 CPU and 2Gb of RAM probably doesn't help.

#### Whitebox Testing LogStash

To white-box test LogStash, I use a technique partially based on the [Agolo LogStash Test Runner](https://github.com/agolo/logstash-test-runner).  The process for the tests is to run LogStash interactively (rather than as a service), send it a single event, record the output events, and compare them to an expected output.

The test cases are kept in separate folders, with two files.  First is the input file, imaginatively called `input.log`, which will contain one json encoded event per line.  The format needs to match what the result of FileBeat sending an event to LogStash would be.  In this case, it means a few extra fields, and a `message` property containing a string of json.  Formatted for readability, the object looks like this:

```json
{
  "@timestamp": "2018-12-27T14:08:24.753Z",
  "beat": { "hostname": "Spectre", "name": "Spectre", "version": "5.3.0" },
  "fields": { "environment": "local", "log_type": "application" },
  "input_type": "log",
  "message": "{\"Timestamp\": \"2018-12-18T17:06:27.7112297+02:00\",\"Level\": \"Information\",\"MessageTemplate\": \"This is the {count} message\",\"Properties\": {\"count\": 4,\"SourceContext\": \"LogLines.GetOpenPurchasesHandler\",\"ApplicationName\": \"FileBeatTest\",\"CorrelationId\": \"8f341e8e-6b9c-4ebf-816d-d89c014bad90\",\"TimedOperationElapsedInMs\": 1000}}",
  "offset": 318,
  "source": "D:\\tmp\\logs\\single.log",
  "type": "applicationlog"
}
```

I also define an `output.log`, which contains the expected result(s), again one json encoded event per line.  The example pipeline in the repository will emit two events for a given input, so this file contains two lines of json (again, newlines added for readability here):

```json
{
  "source": "D:\\tmp\\logs\\single.log",
  "@version": "1",
  "fields": { "log_type": "application", "environment": "local" },
  "@timestamp": "2018-12-18T15:06:27.711Z",
  "offset": 318,
  "ApplicationName": "FileBeatTest",
  "host": "ubuntu-16",
  "type": "applicationlog",
  "CorrelationId": "8f341e8e-6b9c-4ebf-816d-d89c014bad90",
  "MessageTemplate": "This is the {count} message",
  "Level": "Information",
  "Context": "LogLines.GetOpenPurchasesHandler",
  "TimeElapsed": 1000,
  "Properties": { "count": 4 }
}
{
  "duration": 1000000,
  "timestamp": 1545145586711000,
  "id": "<generated>",
  "traceid": "8f341e8e6b9c4ebf816dd89c014bad90",
  "name": "LogLines.GetOpenPurchasesHandler",
  "localEndpoint": { "serviceName": "FileBeatTest" }
}
```

To enable sending the lines directly to LogStash (rather than needing to use FileBeat), we define an `input.conf` file, which configures LogStash to read json from stdin:

```conf
input {
  stdin { codec => "json_lines" }
}
```

And an `ouput.conf` file which configures LogStash to write the output as json lines a known file path:

```conf
output {
  file {
    path => "/tmp/test/output.log"
    codec => "json_lines"
  }
}
```

The tests need to be run inside the machine itself, so I created a script in the `./scripts` directory which will do all the work, and can be run by the `execute` method in a Jest test.  The script stops the LogStash service, copies the current configuration from the `./src` directory and the replacement `input.conf` and `output.conf` files to a temporary location, and then runs LogStash once per test case, copying the result file to the test case's directory.

```bash
#! /bin/bash

sudo systemctl stop logstash

temp_path="/tmp/test"
test_source="/vagrant/test/acceptance"

sudo rm -rf "$temp_path/*"
sudo mkdir -p $temp_path
sudo cp -r /vagrant/src/* $temp_path
sudo cp $test_source/*.conf $temp_path

find $test_source/* -type d | while read test_path; do
    echo "Running $(basename $test_path) tests..."

    sudo /usr/share/logstash/bin/logstash \
        "--path.settings" "/etc/logstash" \
        "--path.config" "$temp_path" \
        < "$test_path/input.log"

    sudo touch "$temp_path/output.log"   # create it if it doesn't exist (dropped logs etc.)
    sudo rm -f "$test_path/result.log"
    sudo mv "$temp_path/output.log" "$test_path/result.log"

    echo "$(basename $test_path) tests done"
done

sudo systemctl start logstash
```

To execute this, we use the `beforeAll` function to run it once - we also pass in `Number.MAX_SAFE_INTEGER` as by default `beforeAll` will time out after 5 seconds, and the `test.sh` is **slow as hell** (as LogStash takes ages to start up).

Once the `test.sh` script has finished running, we load each test's `output.log` and `result.log` files, parse each line as json, compare the objects, and print out the delta if the objects are not considered equal:

```javascript
const source = "./test/acceptance";
const isDirectory = p => fs.lstatSync(p).isDirectory();

const cases = fs
  .readdirSync(source)
  .map(name => path.join(source, name))
  .filter(isDirectory);

describe("logstash", () => {
  beforeAll(
    () => execute("/vagrant/scripts/test.sh"),
    Number.MAX_SAFE_INTEGER);

  test.each(cases)("%s", directoryPath => {
    const expected = readFile(path.join(directoryPath, "output.log"));
    const actual = readFile(path.join(directoryPath, "result.log"));

    const diffpatch = new DiffPatcher({
      propertyFilter: (name, context) => {
        if (name !== "id") {
          return true;
        }

        return context.left.id !== "<generated>";
      }
    });

    const delta = diffpatch.diff(expected, actual);
    const output = formatters.console.format(delta);

    if (output.length) {
      console.log(output);
    }

    expect(output.length).toBe(0);
  });
});
```

#### Blackbox Testing LogStash

As the machine has ports open for FileBeat and will send it's output to ElasticSearch, we can set up a fake HTTP server, send some log events via FileBeat to the VM and check we receive the right HTTP calls to our fake server.

While looking on how to do this, I came across the [lumberjack-protocol](https://www.npmjs.com/package/lumberjack-protocol) package on NPM, but unfortunately, it only supports lumberjack v1, and FileBeat and LogStash are now using v2, so you would have to use a local copy of filebeat to do the sending.

Due to the complexity of implementing this, and the diminished return on investment (the other tests should be sufficient), I have skipped creating the Blackbox tests for the time being.

## AMI Testing

The final phase!  Now that we are reasonably sure everything works locally, we need to build our AMI and test that everything works there too, as it would be a shame to update an Auto Scale Group with the new image which doesn't work!

All that needs to happen to run the tests against an EC2 instance is to set the three environment variables we used with Vagrant, to values for communicating with the EC2 instance. To do this, we'll need the EC2 IP Address, the username for SSH, and the private key for SSH authentication.

The first thing our build script needs to do is create the AMI.  This is done in the same way as [mentioned earlier](#running), but with the slight difference of also piping the output to `tee`:

```bash
packer_log=$(packer build logstash.json | tee /dev/tty)
ami_id=$(echo "$packer_log" | tail -n 1 | sed 's/.*\(ami.*\)/\1/')
```

By using `tee`, we can pipe the build log from Packer to both the real terminal (`/dev/tty`), and to a variable called `packer_log`.  The script then takes the last line and uses some regex to grab the AMI ID.

Next up, the script uses the AWS CLI to launch an EC2 instance based on the AMI, and store it's IP Address and Instance ID:

```bash
json=$(aws ec2 run-instances \
  --image-id "$ami_id" \
  --instance-type t2.small \
  --key-name "$keypair_name" \
  --region eu-west-1 \
  --subnet-id "$subnet_id" \
  --security-group-ids "$security_group_id" \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=logstash-verification}]' \
  --user-data "$userdata")

instance_id=$(echo "$json" | jq -r .Instances[0].InstanceId)
private_ip=$(echo "$json" | jq -r .Instances[0].PrivateIpAddress)
```

The IP Address is then used to set up the environment variables which the node test scripts use to locate the machine:

```bash
LOGSTASH_ADDRESS="$private_ip"
LOGSTASH_SSH="ubuntu"
LOGSTASH_KEYPATH="~/.ssh/id_rsa" build ou

npm run test
```

Finally, the script uses the Instance ID to terminate the instance:

```bash
aws ec2 terminate-instances \
  --instance-ids "$instance_id"
```

## Wrapping Up

Hopefully, this (rather long) post is a useful introduction (!) to how I tackle testing Immutable Infrastructure.  All of these techniques for testing the machine and application can be used for testing things like Docker containers too (and handily, Packer can be used to create Docker containers also).

As mentioned earlier [The Repository is available here](https://github.com/Pondidum/immutable-infra-testing-demo).
