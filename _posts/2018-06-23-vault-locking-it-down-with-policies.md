---
layout: post
title: Locking Vault Down with Policies
tags: vault security microservices
---

The final part of my Vault miniseries focuses on permissioning, which is provided by Vault's [Policies](https://www.vaultproject.io/docs/concepts/policies.html).

As everything in Vault is represented as a path, the policies DSL (Domain Specific Language) just needs to apply permissions to paths to lock things down.  For example, to allow all operations on the `cubbyhole` secret engine, we would define this policy:

```bash
path "cubbyhole/*" {
    capabilities = ["create", "read", "update", "delete", "list"]
}
```

Vault comes with a default policy which allows token operations (such as looking up its own token info, releasing and renewing tokens), and cubbyhole access.

Let's combine the last two posts ([Managing Postgres Connection Strings with Vault](2018/06/17/secret-management-vault-postgres-connection/) and [Secure Communication with Vault](/2018/06/22/vault-secure-communication/)) and create a Policy which will allow the use of generated database credentials.  If you want more details on the how/why of the set up phase, see those two posts.

## Setup

First, we'll create two containers which will get removed on exit - a Postgres one and a Vault one.  Vault is being started in `dev` mode, so we don't need to worry about init and unsealing it.

```bash
docker run --rm -d -p 5432:5432 -e 'POSTGRES_PASSWORD=postgres' postgres:alpine
docker run --rm -d -p 8200:8200 --cap-add=IPC_LOCK -e VAULT_DEV_ROOT_TOKEN_ID=vault vault
```

Next, we'll create our Postgres user account which Vault will use to create temporary credentials:

```bash
psql --username postgres --dbname postgres
psql> create role VaultAdmin with Login password 'vault' CreateRole;
psql> grant connect on database postgres to vaultadmin;
```

Let's also configure the environment to talk to Vault as an administrator, and enable the two Vault plugins we'll need:

```bash
export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="vault"

vault auth enable approle
vault secrets enable database
```

We'll also set up our database secret engine, and configure database roll creation:

```bash
vault write database/config/postgres_demo \
    plugin_name=postgresql-database-plugin \
    allowed_roles="default" \
    connection_url="postgresql://{{username}}:{{password}}@10.0.75.1:5432/postgres?sslmode=disable" \
    username="VaultAdmin" \
    password="vault"

vault write database/roles/reader \
    db_name=postgres_demo \
    creation_statements="CREATE ROLE \"{{name}}\" WITH LOGIN PASSWORD '{{password}}' VALID UNTIL '{{expiration}}'; \
        GRANT SELECT ON ALL TABLES IN SCHEMA public TO \"{{name}}\";" \
    default_ttl="10m" \
    max_ttl="1h"
```

## Creating a Policy

First, we need to create the policy.  This can be supplied inline on the command line, but reading from a file means it can be source-controlled, and you something readable too!

While the filename doesn't need to match the policy name, it helps make it a bit clearer if it does match, so we'll call this file `postgres-connector.hcl`.

```bash
# vault read database/creds/reader
path "database/creds/reader" {
    capabilities = ["read"]
}
```

We can then register this policy into Vault.  The `write` documentation indicates that you need to prefix the file path with `@`, but that doesn't work for me:

```bash
vault policy write postgres-connector postgres-connector.hcl
```

## Setup AppRoles

As before, we'll create a `demo_app` role for our application to use to get a token.  However this time, we'll specify the `policies` field, and pass it in both `default` and our custom `postgres-connector` role.

```bash
vault write auth/approle/role/demo_app \
    policies="postgres-connector,default"
```

When we generate our client token using the `secret_id` and `role_id`, we'll get a token which can create database credentials, as well as access the cubbyhole.

The final part of being an **admin** user for this is to generate and save the `secret_id` and `role_id`:
```bash
vault write -f -field=secret_id auth/approle/role/demo_app/secret-id
vault read -field=role_id auth/approle/role/demo_app/role-id
```

## Creating a Token and Accessing the Database

Opening a new command line window, we need to generate our client token.  Take the two id's output from the admin window, and use them in the following code block:

```bash
export VAULT_ADDR="http://localhost:8200"
SECRET_ID="" # from the 'admin' window!
ROLE_ID="" # from the 'admin' window!

export VAULT_TOKEN=$(curl -X POST --data "{ \"role_id\":\"$ROLE_ID\", \"secret_id\":\"$SECRET_ID\" }" $VAULT_ADDR/v1/auth/approle/login | jq  -r .auth.client_token)
```

Now we have a client token, we can generate a database connection:

```bash
vault read database/creds/reader
# Key                Value
# ---                -----
# lease_id           database/creds/reader/dc2ae2b6-c709-0e2f-49a6-36b45aa84490
# lease_duration     10m
# lease_renewable    true
# password           A1a-1kAiN0gqU07BE39N
# username           v-approle-reader-incldNFPhixc1Kj25Rar-1529764057
```

Which can also be renewed:

```bash
vault lease renew database/creds/reader/dc2ae2b6-c709-0e2f-49a6-36b45aa84490
# Key                Value
# ---                -----
# lease_id           database/creds/reader/dc2ae2b6-c709-0e2f-49a6-36b45aa84490
# lease_duration     10m
# lease_renewable    true
```

However, if we try to write to the database roles, we get an error:

```bash
vault write database/roles/what dbname=postgres_demo
# Error writing data to database/roles/what: Error making API request.
#
# URL: PUT http://localhost:8200/v1/database/roles/what
# Code: 403. Errors:
#
# * permission denied
```

## Summary

It is also a good idea to have separate fine-grained policies, which can then be grouped up against separate AppRoles, allowing each AppRole to have just the permissions it needs.  For example, you could have the following Policies:

* postgres-connection
* postgres-admin
* rabbitmq-connection
* kafka-consumer

You would then have several AppRoles defined which could use different Policies:

* App1: rabbitmq-connection, postgres-connection
* App2: kafka-consumer, rabbitmq-connection
* App3: postgres-admin

Which helps encourage you to have separate AppRoles for each of your applications!

Finally, the Vault website has a [guide](https://www.vaultproject.io/guides/secret-mgmt/dynamic-secrets.html) on how to do this too...which I only found after writing this!  At least what I wrote seems to match up with their guide pretty well, other than I also use `AppRole` authentication (and so should you!)