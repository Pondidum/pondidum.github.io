---
layout: post
title: Creating a TLS enabled Consul cluster
tags: vault security tls consul
---


This post is going to go through how to set up a [Consul](https://www.consul.io/) cluster to communicate over TLS.  I will be using [Vagrant](https://www.vagrantup.com/) to create three machines locally, which will form my cluster, and in the provisioning step will use Vault to generate the certificates needed.

How to securely communicate with Vault to get the TLS certificates is out of scope for this post.

## Host Configuration

Unless you already have Vault running somewhere on your network, or have another mechanism to generate TLS certificates for each machine, you'll need to start and configure Vault on the Host machine.  I am using my [Vault Dev Intermediate CA script from my previous post](https://andydote.co.uk/2019/08/25/vault-development-ca/#creating-a-vault-intermediate-ca).

To set this up, all I need to do is run this on the host machine, which starts Vault in a docker container, and configures it as an intermediate certificate authority:

```bash
./run_int.sh
```

I also have DNS on my network setup for the `tecra.xyz` domain so will be using that to test with.

## Consul Machine Configuration

The `Vagrantfile` is very minimal - I am using my [Hashibox](https://app.vagrantup.com/pondidum/boxes/hashibox) (be aware the `libvirt` provider for this might not work, for some reason `vagrant package` with libvirt produces a non-bootable box).

```ruby
Vagrant.configure(2) do |config|
  config.vm.box = "pondidum/hashibox"
  config.vm.provision "consul", type: "shell", path: "./provision.sh"

  config.vm.define "c1" do |c1|
    c1.vm.hostname = "consul1"
  end

  config.vm.define "c2" do |c2|
    c2.vm.hostname = "consul2"
  end

  config.vm.define "c3" do |c3|
    c3.vm.hostname = "consul3"
  end
end
```

The hashibox script already has all the tools we'll need installed already: Consul, Vault, and jq.

First up, we request a certificate from Vault to use for Consul - How you get this certificate in a secure manner in a production environment is up to you.  There is a catch-22 here for me, in that in a production environment I use Vault with Consul as it's backing store...but Consul needs Vault to start!  I'll go over how I get around this in a future post.

```bash
export VAULT_ADDR="http://vault.tecra.xyz:8200"
export VAULT_TOKEN="vault"

response=$(vault write pki/issue/cert -format=json common_name=$HOSTNAME.tecra.xyz alt_names="server.dc1.consul")
config_dir="/etc/consul.d"
```

The first thing to note is that we have specified an `alt_names` for the certificate - you *must* have a SAN of `server.$DC.$DOMAIN` so either `server.dc1.consul` or `server.euwest1.tecra.xyz`, and the `server` prefix is required!.

Next, we need to take all the certificates from the response and write them to the filesystem.


```bash
mkdir -p "$config_dir/ca"

for (( i=0; i<$(echo "$response" | jq '.data.ca_chain | length'); i++ )); do
  cert=$(echo "$response" | jq -r ".data.ca_chain[$i]")
  name=$(echo "$cert" | openssl x509 -noout -subject -nameopt multiline | sed -n 's/ *commonName *= //p' | sed 's/\s//g')

  echo "$cert" > "$config_dir/ca/$name.pem"
done

echo "$response" | jq -r .data.private_key > $config_dir/consul.key
echo "$response" | jq -r .data.certificate > $config_dir/consul.crt
echo "$response" | jq -r .data.issuing_ca >> $config_dir/consul.crt
```

The `for` loop iterates through all of the certificates returned in the `ca_chain` and writes them into a `ca` directory.  We use `openssl` to get the name of the certificate, so the files are named nicely!

Finally, it writes the `private_key` for the node's certificate to `consul.key`, and both the `certificate` and `issuing_ca` to  the `consul.crt` file.


Now for the `consul.json`.  To setup a secure cluster, first of all we need to add the certificate configuration, pointing to the files we wrote earlier:

```json
"ca_path": "$config_dir/ca/",
"cert_file": "$config_dir/consul.crt",
"key_file": "$config_dir/consul.key",
```

We will also disable the HTTP port, and enable the HTTPS port:

```json
"ports": {
    "http": -1,
    "https": 8501
}
```

Finally, we need to add some security settings.  First is `encrypt`, which sets that the key that all Consul nodes will use to encrypt their communications.  It must match on all nodes.  The easiest way to generate this is just run `consul keygen` and use the value that produces.

* `"encrypt": "oNMJiPZRlaP8RnQiQo9p8MMK5RSJ+dXA2u+GjFm1qx8="`:

  The key the cluster will use to encrypt all it's traffic.  It must be the same on all nodes, and the easiest way to generate the value is to use the output of `consul keygen`.

* `"verify_outgoing": true`:

  All the traffic leaving this node will be encrypted with the TLS certificates.  However, the node will still accept non-TLS traffic.

* `"verify_incoming_rpc": true`:

  All the gossip traffic arriving at this node must be signed with an authority in the `ca_path`.

* `"verify_incoming_https": false`:

  We are going to use the Consul Web UI, so we want to allow traffic to hit the API without a client certificate.  If you are using the UI from a non-server node, you can set this to `true`.

* `"verify_server_hostname": true`:

  Set Consul to verify **outgoing** connections have a hostname in the format of `server.<datacenter>.<domain>`.  From the [docs](https://www.consul.io/docs/agent/options.html#verify_server_hostname): "This setting is critical to prevent a compromised client from being restarted as a server and having all cluster state including all ACL tokens and Connect CA root keys replicated to it"


The complete config we will use is listed here:

```bash
(
cat <<-EOF
{
    "bootstrap_expect": 3,
    "client_addr": "0.0.0.0",
    "data_dir": "/var/consul",
    "leave_on_terminate": true,
    "rejoin_after_leave": true,
    "retry_join": ["consul1", "consul2", "consul3"],
    "server": true,
    "ui": true,
    "encrypt": "oNMJiPZRlaP8RnQiQo9p8MMK5RSJ+dXA2u+GjFm1qx8=",
    "verify_incoming_rpc": true,
    "verify_incoming_https": false,
    "verify_outgoing": true,
    "verify_server_hostname": true,
    "ca_file": "$config_dir/issuer.crt",
    "cert_file": "$config_dir/consul.crt",
    "key_file": "$config_dir/consul.key",
    "ports": {
        "http": -1,
        "https": 8501
    }
}
EOF
) | sudo tee $config_dir/consul.json
```

Lastly, we'll make a systemd service unit to start consul:

```bash
(
cat <<-EOF
[Unit]
Description=consul agent
Requires=network-online.target
After=network-online.target

[Service]
Restart=on-failure
ExecStart=/usr/bin/consul agent -config-file=$config_dir/consul.json -bind '{{ GetInterfaceIP "eth0" }}'
ExecReload=/bin/kill -HUP $MAINPID

[Install]
WantedBy=multi-user.target
EOF
) | sudo tee /etc/systemd/system/consul.service

sudo systemctl daemon-reload
sudo systemctl enable consul.service
sudo systemctl start consul
```

As the machines we are starting also have docker networks (and potentially others), our startup line specifies to bind to the `eth0` network, using a Consul Template.

## Running

First, we need to run our intermediate CA, then provision our three machines:

```bash
./run_int.sh
vagrant up
```

After a few moments, you should be able to `curl` the consul ui (`curl https://consul1.tecra.xyz:8501`) or open `https://consul1.tecra.xyz:8501` in your browser.

Note, however, the if your root CA is self-signed, like mine is, some browsers (such as FireFox) won't trust it, as they won't use your machine's Trusted Certificate Store, but their own in built store.  You can either accept the warning or add your root certificate to the browser's store.

## Testing

Now that we have our cluster seemingly running with TLS, what happens if we try to connect a Consul client _without_ TLS to it?  On the host machine, I just run a single node, and tell it to connect to one of the cluster nodes:

```bash
consul agent \
  -join consul1.tecra.xyz \
  -bind '{{ GetInterfaceIP "eth0" }}' \
  -data-dir /tmp/consul
```

The result of this is a refusal to connect, as the cluster has TLS configured, but this instance does not:

```
==> Starting Consul agent...
==> Log data will now stream in as it occurs:
==> Joining cluster...
==> 1 error occurred:
  * Failed to join 192.168.121.231: Remote state is encrypted and encryption is not configured
```

Success!

In the next post, I'll go through how we can set up a Vault cluster which stores its data in Consul, but also provision that same Consul cluster with certificates from the Vault instance!
