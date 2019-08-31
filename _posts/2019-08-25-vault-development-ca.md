---
layout: post
title: Using Vault as a Development CA
tags: vault security ssl
---

Often when developing or testing some code, I need (or want) to use SSL, and one of the easiest ways to do that is to use [Vault](https://www.vaultproject.io/).  However, it gets pretty annoying having to generate a new CA for each project, and import the CA cert into windows (less painful in Linux, but still annoying), especially as I forget which cert is in use, and accidentally clean up the wrong ones.

My solution has been to generate a single CA certificate and PrivateKey, import this into my Trusted Root Certificate Store, and then whenever I need a Vault instance, I just setup Vault to use the existing certificate and private key.  The documentation for how to do this seems somewhat lacking, so here's how I do it.

Things you'll need:

* Docker
* Vault cli
* JQ

## Generating the Root Certificate

First we need to create a Certificate, which we will do using the Vault docker container, and our local Vault CLI.  We start the docker container in the background, and mark it for deletion when it stops (`--rm`):

```bash
container=$(docker run -d --rm  --cap-add=IPC_LOCK -p 8200:8200 -e "VAULT_DEV_ROOT_TOKEN_ID=vault" vault:latest)

export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="vault"

certs_dir="./ca"
max_ttl="87600h" # 10 years why not

mkdir -p $certs_dir
rm -rf $certs_dir/*.*

vault secrets enable pki
vault secrets tune -max-lease-ttl=$max_ttl pki
```

Finally, we generate a certificate by writing to the `pki/root/generate/exported` path.  If the path ends with `exported` the Private Key is returned too.  If you specify `/internal` then the Private Key is stored internally to Vault, and never accessible.

```bash
result=$(vault write -format "json" \
  pki/root/generate/exported \
  common_name="Local Dev CA" \
  alt_names="localhost,mshome.net" \
  ttl=$max_ttl)

echo "$result" > $certs_dir/response.json
echo "$result" | jq -r .data.certificate > $certs_dir/ca.crt
echo "$result" | jq -r .data.private_key > $certs_dir/private.key

docker stop $container
```

We put the entire response into a `json` file just incase there is something interesting we want out of it later, and store the certificate and private key into the same directory too.  Note for the certificate's `alt_names` I have specified both `localhost` and `mshome.net`, which is the domain that Hyper-V machines use.

Lastly, we can now import the root CA into our machine/user's Trusted Root Certification Authorities store, meaning our later uses of this certificate will be trusted by our local machine.

## Creating a Vault CA

As before, we use a Docker container to run the Vault instance, except this time we import the existing CA certificate into the PKI backend.  The first half of the script (`run_ca.sh`) is pretty much the same as before, except we don't delete the contents of the `./ca` directory, and our certificate `max_ttl` is much lower:

```bash
docker run -d --rm  --cap-add=IPC_LOCK -p 8200:8200 -e "VAULT_DEV_ROOT_TOKEN_ID=vault" vault:latest

export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="vault"

certs_dir="./ca"
max_ttl="72h"

vault secrets enable pki
vault secrets tune -max-lease-ttl=$max_ttl pki
```

The last part is to read in the certificate and private key, bundle them together, and configure the `pki` backend to use them, and add a single role to use for issuing certificates:

```bash
pem=$(cat $certs_dir/ca.crt $certs_dir/private.key)

vault write pki/config/ca pem_bundle="$pem"

vault write pki/roles/cert \
  allowed_domains=localhost,mshome.net \
  allow_subdomains=true \
  max_ttl=$max_ttl
```

Also note how we don't stop the docker container either.  Wouldn't be much of a CA if it stopped the second it was configured...


## Creating a Vault Intermediate CA

Sometimes, I want to test that a piece of software works when I have issued certificates from an Intermediate CA, rather than directly from the root.  We can configure Vault to do this too, with a modified script which this time we start two PKI secret backends, one to act as the root, and onc as the intermediate:

```bash
#!/bin/bash

set -e

docker run -d --rm  --cap-add=IPC_LOCK -p 8200:8200 -e "VAULT_DEV_ROOT_TOKEN_ID=vault" vault:latest

export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="vault"

# create root ca
certs_dir="./ca"
pem=$(cat $certs_dir/ca.crt $certs_dir/private.key)

vault secrets enable -path=pki_root pki
vault secrets tune -max-lease-ttl=87600h pki_root
vault write pki_root/config/ca pem_bundle="$pem"

# create the intermediate
vault secrets enable pki
vault secrets tune -max-lease-ttl=43800h pki

csr=$(vault write pki/intermediate/generate/internal \
  -format=json common_name="Spectre Dev Intermdiate CA" \
  | jq -r .data.csr)

intermediate=$(vault write pki_root/root/sign-intermediate \
  -format=json csr="$csr" format=pem_bundle ttl=43800h \
  | jq -r .data.certificate)

vault write pki/intermediate/set-signed certificate="$intermediate"

echo "$intermediate" > intermediate.crt

vault write pki/roles/cert \
  allowed_domains=localhost,mshome.net \
  allow_subdomains=true \
  max_ttl=43800h

# destroy the temp root
vault secrets disable pki_root
```

We use the `pki_root` backend to sign a CSR from the `pki` (intermediate) backend, and once the signed response is stored in `pki`, we delete the `pki_root` backend, as it is no longer needed for our Development Intermediate CA.


## Issuing Certificates

We can now use the `cert` role to issue certificates for our applications, which I have in a script called `issue.sh`:

```bash
#!/bin/bash

export VAULT_ADDR="http://localhost:8200"
export VAULT_TOKEN="vault"

vault write \
  -format=json \
  pki/issue/cert \
  common_name=$1.mshome.net
```

This script I usually use with `jq` to do something useful with:

```bash
response=$(./issue.sh consul)

cert=$(echo "$response" | jq -r .data.certificate)
key=$(echo "$response" | jq -r .data.private_key)
```

## Cleaning Up

When I have finished with an application or demo, I can just stop the Vault container, and run the `run_ca.sh` script again if I need Vault for another project.
