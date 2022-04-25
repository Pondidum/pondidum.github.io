---
date: "2018-06-17T00:00:00Z"
tags: vault security microservices postgres
title: Managing Postgres Connection Strings with Vault
---

One of the points I made in my recent NDC talk on 12 Factor microservices, was that you shouldn't be storing sensitive data, such as API keys, usernames, passwords etc. in the environment variables.

> Don't Store Sensitive Data in the Environment

My reasoning is that when you were accessing Environment Variables in Heroku's platform, you were actually accessing some (probably) secure key-value store, rather than actual environment variables.

While you can use something like Consul's key-value store for this, it's not much better as it still stores all the values in plaintext, and has no auditing or logging.

## Enter Vault

Vault is a secure secret management application, which not only can store static values, but also generate credentials on the fly, and automatically expire them after usage or after a time period.  We're going to look at setting up Vault to generate Postgres connection strings.

## What you'll need

1. Docker, as we'll be running both Vault and Postgres in containers
2. A SQL client (for a GUI, I recommend [DBeaver](https://dbeaver.io/), for CLI [PSQL](https://www.postgresql.org/download/) included in the Postgres download is fine.)
3. The [Vault executable](https://www.vaultproject.io/downloads.html)

## What we'll do

1. Setup Postgres and create a SQL user for Vault to use
2. Setup Vault
3. Setup Vault's database functionality
4. Fetch and renew credentials from Vault.


## 1. Setup a Postgres container

When running on my local machine, I like to use the Alpine variant of the [official Postgres](https://hub.docker.com/_/postgres/) container, as it's pretty small, and does everything I've needed so far.

We'll run a copy of the image, configure it to listen on the default port, and use the super secure password of `postgres`:

```bash
docker run \
  -d \
  --name postgres_demo \
  -p 5432:5432 \
  -e 'POSTGRES_PASSWORD=postgres' \
  postgres:alpine
```

Next up, we need to create a user for Vault to use when generating credentials.  You can execute this SQL in any SQL editor which can connect to postgres, or use the PSQL command line interface:

```bash
psql --username postgres --dbname postgres   # it will prompt for password
psql> create role VaultAdmin with Login password 'vault' CreateRole;
psql> grant connect on database postgres to vaultadmin;
```

You can verify this has worked by running another instance of psql as the new user:

```bash
psql --username VaultAdmin --dbname postgres   # it will prompt for password
```

## 2. Setting up the Vault container

The official Vault container image will by default run in `dev` mode, which means it will startup unsealed, and will use whatever token you specify for authentication.  However, it won't persist any information across container restarts, which is a bit irritating, so instead, we will run it in server mode, and configure file storage to give us (semi) persistent storage.

The configuration, when written out and appropriately formatted, looks as follows:

```bash
backend "file" {
    path = "/vault/file"
}
listener "tcp" {
    address = "0.0.0.0:8200"
    tls_disable = 1
}
ui = true
```

We are binding the listener to all interfaces on the container, disabling SSL (don't do this in production environments!) and enabling the UI.  To pass this through to the container, we can set the `VAULT_LOCAL_CONFIG` environment variable:

```bash
docker run \
    -d \
    --name vault_demo \
    --cap-add=IPC_LOCK \
    -p 8200:8200 \
    -e 'VAULT_LOCAL_CONFIG=backend "file" { path = "/vault/file" } listener "tcp" { address = "0.0.0.0:8200" tls_disable = 1 } ui = true' \
    vault server
```

When we use the Vault CLI to interact with a Vault server, it want's to use TLS, but as we are running without TLS, we need to override this default.  Luckily it's just a case of setting the `VAULT_ADDR` environment variable:

```bash
export VAULT_ADDR="http://localhost:8200"
```

You can run `vault status` to check you can communicate with the container successfully.

Before we can start configuring secret engines in Vault, it needs initialising.  By default, the `init` command will generate five key shares, of which you will need any three to unseal Vault.  The reason for Key Shares is so that you can distribute the keys to different people so that no one person has access to unseal Vault on their own.  While this is great for production, for experimenting locally, one key is enough.

```bash
vault operator init -key-shares=1 -key-threshold=1
```

The output will amongst other things give you two lines, one with the Unseal Key, and one with the Initial Root Token:

> Unseal Key 1: sk+C4xJihsMaa+DCBHHgoGVozz+dMC4Kd/ijX8oMcrQ=
Initial Root Token: addaaeed-d387-5eab-128d-60d6e92b0757

We'll need the Unseal key to unseal Vault so we can configure it and generate secrets, and the Root Token so we can authenticate with Vault itself.

```bash
 vault operator unseal "sk+C4xJihsMaa+DCBHHgoGVozz+dMC4Kd/ijX8oMcrQ="
```

To make life a bit easier, we can also set an environment variable with our token so that we don't have to specify it on all the subsequent requests:

```bash
export VAULT_TOKEN="addaaeed-d387-5eab-128d-60d6e92b0757"
```

## 3. Configure Vault's Database Secret Engine

First off we need to enable the database secret engine.  This engine supports many different databases, such as Postgres, MSSQL, Mysql, MongoDB and Cassandra amongst others.

```bash
vault secrets enable database
```

Next, we need to configure how vault will connect to the database.  You will need to substitute the IPAddress in the connection string for your docker host IP (in my case, the network is called `DockerNAT`, and my machine's IP is `10.0.75.1`, yours will probably be different.)

```bash
vault write database/config/postgres_demo \
    plugin_name=postgresql-database-plugin \
    allowed_roles="*" \
    connection_url="postgresql://{{username}}:{{password}}@10.0.75.1:5432/postgres?sslmode=disable" \
    username="VaultAdmin" \
    password="vault"
```

To explain more of the command:  We can limit what roles can be granted by this database backend by specifying a CSV of roles (which we will define next).  In our case, however, we are using the allow anything wildcard (`*`).

Next, we need to define a role which our applications can request.  In this case, I am creating a role which only allows reading of data, so it's named `reader`.  We also specify the `default_ttl` which controls how long the user is valid for, and the `max_ttl` which specifies for how long we can renew a user's lease.

```bash
vault write database/roles/reader \
    db_name=postgres_demo \
    creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; \
        GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
    default_ttl="10m" \
    max_ttl="1h"
```

```bash
vault read database/creds/reader
> Key                Value
> ---                -----
> lease_id           database/creds/reader/15cf95eb-a2eb-c5ba-5111-8c0c48ae30a6
> lease_duration     10m
> lease_renewable    true
> password           A1a-3gkMQpmoh3gbj2aM
> username           v-root-reader-tgl6FSXHZaC5LZOK4q0u-1529138525
```

We can now use the username and password to connect to postgres, but only for 10 minutes, after which, the user will be deleted (Note, Vault sets the expiry of the user in Postgres, but will also remove the user when it expires.)

Verify the user can connect using PSQL again:

```bash
psql --username v-root-reader-tgl6FSXHZaC5LZOK4q0u-1529138525 --dbname postgres
```

If we want to keep using our credentials, we can run the renew command passing in the `lease_id`, which will increase the current lease timeout by the value of `default_ttl`.  You can provide the `-increment` value to request a different duration extension in seconds, but you cannot go further than the `max_ttl`.

```bash
vault lease renew database/creds/reader/15cf95eb-a2eb-c5ba-5111-8c0c48ae30a6
# or
vault lease renew database/creds/reader/15cf95eb-a2eb-c5ba-5111-8c0c48ae30a6 -increment 360
```

## Done!

There are a lot more options and things you can do with Vault, but hopefully, this will give you an idea of how to start out.