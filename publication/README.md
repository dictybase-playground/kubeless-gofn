# publication function
A [kubeless](https://kubeless.io) function to deploy in kubernetes cluster.

## Deploy the function
> `$_>  kubeless function deploy \`   
> `pubfn --runtime go1.10 --from-file publication.go --handler publication.Handler`   
> `--dependencies Gopkg.toml --namespace dictybase`

* check the status of function
> `$_> kubeless function ls --namespace dictybase`

## Add a http trigger to create an ingress
> `$_> kubeless trigger http create pubfn \`   
> `--function-name pubfn --hostname betafunc.dictybase.local \`   
> `--tls-secret dictybase-local-tls --namespace dictybase --path publications`

The above command assumes a presence of tls secret`(dictybase-local-tls)` and mapping
to the host`(betafunc.dictybase.local)`.

## Endpoints
It will available through the mapped host, for example through
`betafunc.dictybase.local` assuming the above function.

__GET__ `/publications/{pubmed-id}` - Information about a publication with pubmed id.   

> `$_> curl -k https://betafunc.dictybase.local/publications/30048658`
