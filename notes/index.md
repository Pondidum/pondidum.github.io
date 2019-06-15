# Random Notes

Just little snippets of things which might be useful later, but are not really worthy of a blog post.

## Fabio

The  `urlprefix-` in the tags for a service in Consul is a literal constant.
e.g. `urlprefix-/` -> matches everything


## Traefik

When you specify a Traefik constraint as `--consulcatalog.constraints='tag==service'`, the tag it looks for in Consul is `traefik.tags=service`, NOT `service`.
