+++
summary = 'Notes'
title = 'Random Notes'
url = '/notes/'

+++

Just little snippets of things which might be useful later, but are not really worthy of a blog post.

## Fabio

The  `urlprefix-` in the tags for a service in Consul is a literal constant.
e.g. `urlprefix-/` -> matches everything


## Traefik

When you specify a Traefik constraint as `--consulcatalog.constraints='tag==service'`, the tag it looks for in Consul is `traefik.tags=service`, NOT `service`.


## Packer

EC2 (and GCP) instances run scripts on machine start, but Packer won't wait for these to complete before running things, which can break `apt`.  Add this as a provisioner to wait for AWS Cloud-Init to finish before your scripts which need `apt`:

```json
{
    "type":"shell",
    "only": [ "amazon-ebs" ],
    "inline": [ "/usr/bin/cloud-init status --wait" ]
}
```


## Vagrant

Override the default smb share in hyper-v with the username and password to use, pulled from environment:

```ruby
config.vm.synced_folder ".", "/vagrant", smb_username: ENV['VAGRANT_SMB_USER'], smb_password: ENV['VAGRANT_SMB_PASS']
```


## Jenkins

Adding build parameters e.g. `commit_hash` will show up on the `params.commit_hash` in a pipeline, **but will also** get written as an environment variable, maintaining case, e.g. `env.commit_hash` or `$commit_hash`.

If you export in a different case, the original name is preserved, with the new value e.g. `env.COMMIT_HASH = 'new'` will actually be available as `$commit_hash`.

If you try to unset the original casing first, then set on the casing you want, it **removes both** e.g.

```
env.commit_hash = ""
env.COMMIT_HASH = "yes?"

sh "env | grep -i commit_hash | wc -l" // 0.
```