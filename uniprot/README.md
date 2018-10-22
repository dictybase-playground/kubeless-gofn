# dashboard function
A golang based [kubeless](https://kubeless.io) function to deploy in kubernetes cluster.

## Dependencies
[Redis](https://redis.io) have to be installed
for the function to work. The instruction is given
[here](https://github.com/dictyBase/Migration/blob/master/deploy.md#redis)

## Deploy function
> `$_> zip uniprot.zip *.go Gopkg.toml`   
> `$_>  kubeless function deploy \`   
> `uniprotcachefn --runtime go1.10 --from-file uniprot.zip --handler uniprot.CacheIds`   
> `--dependencies Gopkg.toml --namespace dictybase`   

* read the deployment status of function(blocks terminal)
> `$_> kubectl get pod -l function=uniprotcachefn -n dictybase -w`

* check the status of function
> `$_> kubeless function ls uniprotcachefn --namespace dictybase`

## Run function
> `$_> kubeless function call uniprotcachefn --namespace dictybase`

You will receive the following output   
`name:3664       id:8397 isoform:97      unresolved:8    nomap:1063`

Also open the log in another terminal(blocks terminal)   
> `$_> kubeless function log uniprotcachefn --namespace dictybase -f`


