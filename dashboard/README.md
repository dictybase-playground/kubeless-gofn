# dashboard function
A golang based [kubeless](https://kubeless.io) function to deploy in kubernetes cluster.

## Dependencies
[Redis](https://redis.io) and [minio](https://minio.io) have to be installed
for the function to work. The instructions are given
[here](https://github.com/dictyBase/Migration/blob/master/deploy.md#redis) and
[here](https://github.com/dictyBase/Migration/blob/master/deploy.md#object-storages3-compatible).

## Pre-deploy setup
### Upload file to object storage(minio)
Either use the minio web interface and use the minio command line
[tool](https://docs.minio.io/docs/minio-client-quickstart-guide.html)

Add host   
> `$_> mc config add locals3 $(minikube service --url minio --namespace dictybase) ACCESS_KEY SECRET_KEY`

Create bucket   
> `$_> mc mb locals3/dashboard`

Upload file to any folder inside that bucket   
> `$_> mc copy canonical.gff3 locals3/dashboard/genomes/44689/`

The above bucket and folder path are for example only, any name could be used instead.

### Create metadata json file
This file specify the organism information and file location in the object storage.
So for *D.discoideum*, the metadata file should have following information.
```json
{
    "taxon_id": "44689",
    "scientific_name": "Dictyostelium discoideum",
    "common_name": "Slime mold",
    "rank": "Species",
    "bucket": "dashboard",
    "file": "genomes/44689/canonical_core.gff3"
}
```
Out off all the fields, `taxon_id`,`bucket` and `file` are necessary. The
taxonomic information are available
[here](https://www.uniprot.org/taxonomy/44689).

## Deploy function
> `$_> zip dashfn.zip *.go Gopkg.toml`   
> `$_>  kubeless function deploy \`   
> `dashfn --runtime go1.10 --from-file dashfn.zip --handler dashboard.Handler`   
> `--dependencies Gopkg.toml --namespace dictybase`   
> `-e MINIO_ACCESS_KEY=xxxxxxxx -e MINIO_SECRET_KEY=xxxxxxxx`

* check the status of function
> `$_> kubeless function ls --namespace dictybase`

## Add a http trigger to create an ingress
> `$_> kubeless trigger http create dashfn \`   
> `--function-name dashfn --hostname betafunc.dictybase.local \`   
> `--tls-secret dictybase-local-tls --namespace dictybase --path dashboard`

The above command assumes a presence of tls secret`(dictybase-local-tls)` and mapping
to the host`(betafunc.dictybase.local)`.

## Endpoints
It will available through the mapped host, for example through
`betafunc.dictybase.local` assuming the above http trigger.

__POST__ `/dashboard/genomes` - Generating information for various biological
feature types(chromosome,gene etc..). It will use `metadata.json` file to
download the gff3 file from object storage and persist the information in redis
cache. An example `HTTP` request to this endpoint will look like this.
> `$_> curl -k -d @metadata.json https://betafunction.dictybase.local/dashboard/genomes`

__GET__ `/dashboard/genomes/{taxon_id}/regions` - Information about reference feature such as chromosome.
```json
{
    "data": [
        {
            "type": "chromosomes",
            "id": "....",
            "attributes": {
                "name": "...",
                "start": "...",
                "end": "...",
                "length": "...."
            }
        }
    ]
}
```

__GET__ `/dashboard/genomes/{taxon_id}/genes` - Information about genes.
```json
{
    "data": [
        {
            "type": "genes",
            "id": "....",
            "attributes": {
                "seq_id": "...",
                "block_id": "...",
                "start": "...",
                "end": "...",
                "strand": "....",
                "source": "...."
            }
        }
    ]
}
```

__GET__ `/dashboard/genomes/{taxon_id}/pseudogenes` - Information about pseudogenes.
```json
{
    "data": [
        {
            "type": "pseudogenes",
            "id": "....",
            "attributes": {
                "seq_id": "...",
                "block_id": "...",
                "start": "...",
                "end": "...",
                "strand": "....",
                "source": "...."
            }
        }
    ]
}
```
The taxon id for *D.discoideum* is `44689`


