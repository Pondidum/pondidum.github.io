---
date: "2019-04-06T00:00:00Z"
tags: ["infrastructure", "vagrant", "nomad", "consul", "rabbitmq", "vault"]
title: Running a Secure RabbitMQ Cluster in Nomad
---

Last time I wrote about running a RabbitMQ cluster in Nomad, one of the main pieces of feedback I received was about the (lack) of security of the setup, so I decided to revisit this, and write about how to launch as secure RabbitMQ node in Nomad.

The things I want to cover are:

* Username and Password for the management UI
* Secure value for the Erlang Cookie
* SSL for Management and AMQP

As usual, the [demo repository](https://github.com/Pondidum/Nomad-RabbitMQ-Demo) with all the code is available if you'd rather just jump into that.

## Configure Nomad To Integrate With Vault

To manage the certificates and credentials I will use another Hashicorp tool called [Vault](https://vaultproject.io/), which provides Secrets As A Service.  It can be configured for High Availability, but for the demo, we will just use a single instance on one of our Nomad machines.

### Vault

We'll update the Vagrant script used in the [last post about Nomad Rabbitmq Clustering](/2019/01/28/nomad-rabbitmq-consul-cluster/) to add in a single Vault node.  **This is not suitable for using Vault in production;** for that there should be a separate Vault cluster running somewhere, but as this post is focusing on how to integrate with Vault, a single node will suffice.

Once we have Vault installed ([see the `provision.sh` script](https://github.com/Pondidum/Nomad-RabbitMQ-Demo/blob/master/provision.sh#L50)), we need to set up a few parts.  First is a PKI (public key infrastructure), better known as a Certificate Authority (CA).  We will generate a single root certificate and have our client machines (and optionally the host machine) trust that one certificate.

As this the machines are running in Hyper-V with the Default Switch, we can use the inbuilt domain name, `mshome.net`, and provide our own certificates.  This script is run as part of the Server (`nomad1`) provisioning script, but in a production environment would be outside of this scope.

```bash
domain="mshome.net"
vault secrets enable pki
vault secrets tune -max-lease-ttl=87600h pki

vault write -field=certificate pki/root/generate/internal common_name="$domain" ttl=87600h \
    > /vagrant/vault/mshome.crt

vault write pki/config/urls \
    issuing_certificates="$VAULT_ADDR/v1/pki/ca" \
    crl_distribution_points="$VAULT_ADDR/v1/pki/crl"

vault write pki/roles/rabbit \
    allowed_domains="$domain" \
    allow_subdomains=true \
    generate_lease=true \
    max_ttl="720h"

sudo cp /vagrant/vault/mshome.crt /usr/local/share/ca-certificates/mshome.crt
sudo update-ca-certificates
```

If you don't want scary screens in FireFox and Chrome, you'll need to install the `mshome.crt` certificate into your trust store.

Next up, we have some policies we need in Vault.  The first deals with what the Nomad Server(s) are allowed to do - namely to handle tokens for itself, and anything in the `nomad-cluster` role.  [A full commented version of this policy is available here](https://github.com/Pondidum/Nomad-RabbitMQ-Demo/blob/master/vault/nomad-server-policy.hcl).

```ruby
path "auth/token/create/nomad-cluster" {
  capabilities = ["update"]
}

path "auth/token/roles/nomad-cluster" {
  capabilities = ["read"]
}

path "auth/token/lookup-self" {
  capabilities = ["read"]
}

path "auth/token/lookup" {
  capabilities = ["update"]
}

path "auth/token/revoke-accessor" {
  capabilities = ["update"]
}

path "sys/capabilities-self" {
  capabilities = ["update"]
}

path "auth/token/renew-self" {
  capabilities = ["update"]
}
```

As this policy mentions the `nomad-cluster` role a few times, let's have a look at that also:

```json
{
  "disallowed_policies": "nomad-server",
  "explicit_max_ttl": 0,
  "name": "nomad-cluster",
  "orphan": true,
  "period": 259200,
  "renewable": true
}
```

This allows a fairly long-lived token to be created, which can be renewed.  It is also limiting what the tokens are allowed to do, which can be done as either a block list (`disallowed_policies`) or an allow list (`allowed_policies`).  In this case, I am letting the Clients access any policies except the `nomad-server` policy.

We can install both of these into Vault:

```bash
vault policy write nomad-server /vagrant/vault/nomad-server-policy.hcl
vault write auth/token/roles/nomad-cluster @/vagrant/vault/nomad-cluster-role.json
```

### Nomad

Now that Vault is up and running, we should configure Nomad to talk to it.  This is done in two places - the Server configuration, and the Client configuration.

To configure the **Nomad Server**, we update it's configuration file to include a `vault` block, which contains a role name it will use to generate tokens (for itself and for the Nomad Clients), and an initial token.

```ruby
vault {
    enabled = true
    address = "http://localhost:8200"
    task_token_ttl = "1h"
    create_from_role = "nomad-cluster"
    token = "some_token_here"
}
```

The initial token is generated by the [`./server.sh`](https://github.com/Pondidum/Nomad-RabbitMQ-Demo/blob/master/server.sh) script - how you go about doing this in production will vary greatly depending on how you are managing your machines.

The **Nomad Clients** also need the Vault integration enabling, but in their case, it only needs the location of Vault, as the Server node(s) will provide tokens for the clients to use.

```ruby
vault {
    enabled = true
    address = "http://nomad1.mshome.net:8200"
}
```

## Job Requirements

Before we go about changing the job itself, we need to write some data into Vault for the job to use:

* Credentials: Username and password for the RabbitMQ Management UI, and the `RABBITMQ_ERLANG_COOKIE`
* A policy for the job allowing Certificate Generation and Credentials access

### Credentials

First off, we need to create a username and password to use with the Management UI.  This can be done via the Vault CLI:

```bash
vault kv put secret/rabbit/admin \
    username=administrator \
    password=$(cat /proc/sys/kernel/random/uuid)
```

For the Erlang Cookie, we will also generate a Guid, but this time we will store it under a separate path in Vault so that it can be locked down separately to the admin username and password if needed:

```bash
vault kv put secret/rabbit/cookie \
    cookie=$(cat /proc/sys/kernel/random/uuid)
```

### Job Policy

Following the principle of [Least Privilege](https://en.wikipedia.org/wiki/Principle_of_least_privilege), we will create a policy for our `rabbit` job which only allows certificates to be generated, and rabbit credentials to be read.

```ruby
path "pki/issue/rabbit" {
  capabilities = [ "create", "read", "update", "delete", "list" ]
}

path "secret/data/rabbit/*" {
  capabilities = [ "read" ]
}
```

This is written into Vault in the same way as the other policies were:

```bash
vault policy write rabbit /vagrant/vault/rabbit-policy.hcl
```

## Rabbit Job Configuration

The first thing we need to do to the job is specify what policies we want to use with Vault, and what to do when a token or credential expires:

```ruby
task "rabbit" {
  driver = "docker"

  vault {
    policies = ["default", "rabbit"]
    change_mode = "restart"
  }
  #...
}
```

### Certificates

To configure RabbitMQ to use SSL, we need to provide it with values for 3 environment variables:

* `RABBITMQ_SSL_CACERTFILE` - The CA certificate
* `RABBITMQ_SSL_CERTFILE` - The Certificate for RabbitMQ to use
* `RABBITMQ_SSL_KEYFILE` - the PrivateKey for the RabbitMQ certificate

So let's add a `template` block to the job to generate and write out a certificate.  It's worth noting that **line endings matter**.  You either need your `.nomad` file to use LF line endings, or make the `template` a single line and use `\n` to add the correct line endings in.  I prefer to have the file with LF line endings.

{% raw %}
```bash
template {
  data = <<EOH
{{ $host := printf "common_name=%s.mshome.net" (env "attr.unique.hostname") }}
{{ with secret "pki/issue/rabbit" $host "format=pem" }}
{{ .Data.certificate }}
{{ .Data.private_key }}{{ end }}
EOH
  destination = "secrets/rabbit.pem"
  change_mode = "restart"
}
```
{% endraw %}

As we want to use the Nomad node's hostname within the `common_name` parameter of the secret, we need to use a variable to fetch and format the value:

{% raw %}
```ruby
{{ $host := printf "common_name=%s.mshome.net" (env "attr.unique.hostname") }}
```
{% endraw %}

This can then be used by the `with secret` block to fetch a certificate for the current host:

{% raw %}
```ruby
{{ with secret "pki/issue/rabbit" $host "format=pem" }}
```
{% endraw %}

Now that we have a certificate in the `./secrets/` directory, we can add a couple of volume mounts to the container, and set the environment variables with the container paths to the certificates.  Note how the root certificate is coming from the `/vagrant` directory, not from Vault itself.  Depending on how you are provisioning your machines to trust your CA, you will have a different path here!

```ruby
config {
  image = "pondidum/rabbitmq:consul"
  # ...
  volumes = [
    "/vagrant/vault/mshome.crt:/etc/ssl/certs/mshome.crt",
    "secrets/rabbit.pem:/etc/ssl/certs/rabbit.pem",
    "secrets/rabbit.pem:/tmp/rabbitmq-ssl/combined.pem"
  ]
}

env {
  RABBITMQ_SSL_CACERTFILE = "/etc/ssl/certs/mshome.crt"
  RABBITMQ_SSL_CERTFILE = "/etc/ssl/certs/rabbit.pem"
  RABBITMQ_SSL_KEYFILE = "/etc/ssl/certs/rabbit.pem"
  #...
}
```

You should also notice that we are writing the `secrets/rabbit.pem` file into the container twice:  The second write is to a file in `/tmp` as a workaround for the `docker-entrypoint.sh` script.  If we don't create this file ourselves, the container script will create it by combining the `RABBITMQ_SSL_CERTFILE` file and `RABBITMQ_SSL_KEYFILE` file, which will result in an invalid certificate, and a nightmare to figure out...

If the Vault integration in Nomad could write a single generated secret to two separate files, we wouldn't need this workaround.  Alternatively, you could make a custom container with a customised startup script to deal with this for you.

You can see the version of this file with [only these changes here](https://github.com/Pondidum/Nomad-RabbitMQ-Demo/blob/a588d7c2483c999b2fa0f47433403dfe1838fd50/rabbit/secure.nomad)

### Credentials

Now that we have things running with a certificate, it would be a great idea to start using the Erlang Cookie value and Management UI credentials we stored in Vault earlier.  This is a super easy change to support in the Nomad file - we need to add another `template` block, but this time set `env = true` which will instruct nomad that the key-values in the template should be loaded as environment variables:

{% raw %}
```bash
template {
    data = <<EOH
    {{ with secret "secret/data/rabbit/cookie" }}
    RABBITMQ_ERLANG_COOKIE="{{ .Data.data.cookie }}"
    {{ end }}
    {{ with secret "secret/data/rabbit/admin" }}
    RABBITMQ_DEFAULT_USER={{ .Data.data.username }}
    RABBITMQ_DEFAULT_PASS={{ .Data.data.password }}
    {{ end }}
EOH
    destination = "secrets/rabbit.env"
    env = true
}
```
{% endraw %}

The complete nomad file with [both certificates and credentials can be seen here](https://github.com/Pondidum/Nomad-RabbitMQ-Demo/blob/a78736cac3a93a43a96cbe84492089fca29d15e1/rabbit/secure.nomad).

## Running!

Now, all we need to do is start our new secure cluster:

```bash
nomad job run rabbit/secure.nomad
```

## Client Libraries

Now that you have a secure version of RabbitMQ running, there are some interesting things which can be done with the client libraries.  While you can just use the secure port, RabbitMQ also supports [Peer Verification](https://www.rabbitmq.com/ssl.html#peer-verification), which means that the client has to present a certificate for itself, and RabbitMQ will validate that both certificates are signed by a common CA.

This process can be controlled with two environment variables:

* `RABBITMQ_SSL_VERIFY` set to either `verify_peer` or `verify_none`
* `RABBITMQ_SSL_FAIL_IF_NO_PEER_CERT` set to `true` to require client certificates, `false` to make them optional

In .net land, if you are using MassTransit, the configuration looks like this:

```csharp
var bus = Bus.Factory.CreateUsingRabbitMq(c =>
{
    c.UseSerilog(logger);
    c.Host("rabbitmq://nomad1.mshome.net:5671", r =>
    {
        r.Username("some_application");
        r.Password("some_password");
        r.UseSsl(ssl =>
        {
            ssl.CertificatePath = @"secrets/app.crt";
        });
    });
});
```

There are also lots of other interesting things you can do with SSL and RabbitMQ, such as using the certificate as authentication rather than needing a username and password per app.  But you should be generating your app credentials dynamically with Vault too...

# Wrapping Up

Finding all the small parts to make this work was quite a challenge.  The [Nomad gitter](https://gitter.im/hashicorp-nomad/Lobby) was useful when trying to figure out the certificates issue, and being able to read the [source code](https://github.com/docker-library/rabbitmq/blob/4b2b11c59ee65c2a09616b163d4572559a86bb7b/3.7/alpine/docker-entrypoint.sh#L363) of the Docker image for RabbitMQ was invaluable to making the Certificate work.

If anyone sees anything I've done wrong, or could be improved, I'm happy to hear it!