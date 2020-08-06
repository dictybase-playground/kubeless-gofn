# uniprot cache function

A Golang-based [Kubeless](https://kubeless.io) function to store mapping
between UniProt and _D.discoideum_ gene identifiers.

## Dependencies

- [Kubeless v1.0.7](https://github.com/kubeless/kubeless/releases/tag/v1.0.7)
- [Redis](https://dictybase-docker.github.io/developer-docs/deployment/redis/)

## Deploy function

> `$_> zip uniprot.zip *.go go.mod`  
> `$_> kubeless function deploy \`  
> `uniprotcachefn --runtime go1.13 --from-file uniprot.zip --handler uniprot.CacheIds`  
> `--dependencies go.mod --namespace dictybase`

- read the deployment status of function (blocks terminal)

  > `$_> kubectl get pod -l function=uniprotcachefn -n dictybase -w`

- check the status of function
  > `$_> kubeless function ls uniprotcachefn --namespace dictybase`

## Run function

> `$_> kubeless function call uniprotcachefn --namespace dictybase`

You will receive the following output  
`name:3664 id:8397 isoform:97 unresolved:8 nomap:1063`

Also open the log in another terminal (blocks terminal)

> `$_> kubeless function log uniprotcachefn --namespace dictybase -f`
