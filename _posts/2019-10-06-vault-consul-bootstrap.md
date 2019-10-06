---
layout: post
title: Creating a Vault instance with a TLS Consul Cluster
tags: consul vault infrastructure security tls
---

So we want to set up a [Vault](https://www.vaultproject.io/) instance, and have it's storage be a TLS based [Consul](https://www.consul.io/) cluster.  The problem is that the Consul cluster needs Vault to create the certificates for TLS, which is quite the catch-22.  Luckily for us, quite easy to solve:

1. Start a temporary Vault instance as an intermediate ca
2. Launch Consul cluster, using Vault to generate certificates
3. Destroy temporary Vault instance
4. Start a permanent Vault instance, with Consul as the store
5. Reprovision the Consul cluster with certificates from the new Vault instance

![Sequence diagram of the previous numbered list](/images/consul-vault-bootstrap.png)

There is a [repository on Github with all the scripts](https://github.com/Pondidum/vault-consul-bootstrap-demo) used, and a few more details on some options.

## Assumptions:

The Host machine needs the following software available in your `PATH`:

* [Vagrant](https://www.vagrantup.com/)
* [Consul](https://www.consul.io/)
* [Vault](https://www.vaultproject.io/)

You have a TLS Certificate you can use to create an intermediate CA with.  See this blog post for [How to create a local CA](/2019/08/25/vault-development-ca/)

## Running

The `run.sh` script will do all of this for you, but an explanation of the steps is below:

1. Start a Temporary Vault instance

    ```bash
    echo '
    storage "inmem" {}
    listener "tcp" {
      address = "0.0.0.0:8200"
      tls_disable = 1
    }' > "vault/temp_vault.hcl"

    vault server -config="vault/temp_vault.hcl" &
    echo "$!" > vault.pid

    export VAULT_TOKEN=$(./configure_vault.sh | tail -n 1)
    ```

2. Generate a Vault token for the Consul machines to use to authenticate with Vault

    ```bash
    export CONSUL_VAULT_TOKEN=$(vault write -field=token -force auth/token/create)
    ```

3. Launch 3 Consul nodes (uses the `CONSUL_VAULT_TOKEN` variable)

    ```bash
    vagrant up
    ```

    The `vagrantfile` just declares 3 identical machines:

    ```ruby
    Vagrant.configure(2) do |config|
      config.vm.box = "pondidum/hashibox"

      config.vm.provision "consul",
        type: "shell",
        path: "./provision.sh",
        env: {
            "VAULT_TOKEN" => ENV["CONSUL_VAULT_TOKEN"]
        }

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

    The provisioning script just reads a certificate from Vault, and writes out pretty much the same configuration as in the last post on [creating a TLS enabled Consul Cluster](/2019/09/14/consul-tls-cluster), but you can view it in the [repository](https://github.com/Pondidum/vault-consul-bootstrap-demo) for this demo too.

4. Create a local Consul server to communicate with the cluster:

    ```bash
    ./local_consul.sh
    ```

    This is done so that the Vault instance can always communicate with the Consul cluster, no matter which Consul node we are reprovisioning later.  In a production environment, you would have this Consul server running on each machine that Vault is running on.

5.  Stop the temporary Vault instance now that all nodes have a certificate

    ```bash
    kill $(cat vault.pid)
    ```

6. Start the persistent Vault instance, using the local Consul agent

    ```bash
    echo '
    storage "consul" {
      address = "localhost:8501"
      scheme = "https"
    }
    listener "tcp" {
      address = "0.0.0.0:8200"
      tls_disable = 1
    }' > "$config_dir/persistent_vault.hcl"

    vault server -config="$config_dir/persistent_vault.hcl" > /dev/null &
    echo "$!" > vault.pid

    export VAULT_TOKEN=$(./configure_vault.sh | tail -n 1)
    ```

7. Generate a new Vault token for the Consul machines to use to authenticate with Vault (same as step 2)

    ```bash
    export CONSUL_VAULT_TOKEN=$(vault write -field=token -force auth/token/create)
    ```

8. Reprovision the Consul nodes with new certificates

    ```bash
    vagrant provision c1 --provision-with consul
    vagrant provision c2 --provision-with consul
    vagrant provision c3 --provision-with consul
    ```

9. Profit

    To clean up the host's copy of Vault and Consul, you can run this:

    ```bash
    kill $(cat vault.pid)
    kill $(cat consul.pid)
    ```

## Summary & Further Actions

Luckily, this is the kind of thing that should only need doing once (or once per isolated environment).  When running in a real environment, you will also want to set up:

* ACL in Consul which locks down the KV storage Vault uses to only be visible/writeable by Vault
* Provisioning the `VAULT_TOKEN` to the machines in a secure fashion
* Periodic refresh of the Certificates uses in the Consul cluster
