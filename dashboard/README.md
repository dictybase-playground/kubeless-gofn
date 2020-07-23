# dashboard function

A Golang-based [Kubeless](https://kubeless.io) function to deploy in a Kubernetes cluster.

## Dependencies

- [Kubeless v1.0.7](https://github.com/kubeless/kubeless/releases/tag/v1.0.7)
- [Minio](https://dictybase-docker.github.io/developer-docs/deployment/minio/)
- [Redis](https://dictybase-docker.github.io/developer-docs/deployment/redis/)

## Pre-deploy setup

### Upload file to object storage (Minio)

Either use the Minio web interface or use the Minio command line
[tool](https://docs.minio.io/docs/minio-client-quickstart-guide.html)

Add host

> `$_> mc config add locals3 $(minikube service --url minio --namespace dictybase) ACCESS_KEY SECRET_KEY`

Create bucket

> `$_> mc mb locals3/dashboard`

Upload file to any folder inside that bucket

> `$_> mc copy canonical.gff3 locals3/dashboard/genomes/44689/`

The above bucket and folder path are for example only, any name could be used instead.

### Create metadata json file

This file specifies the organism information and file location in the object storage.
So for _D.discoideum_, the metadata file should have following information.

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

Out of all fields, `taxon_id`,`bucket` and `file` are necessary. The
taxonomic information is available [here](https://www.uniprot.org/taxonomy/44689).

## Deploy function

> `$_> zip dashfn.zip *.go go.mod`  
> `$_> kubeless function deploy \`  
> `dashfn --runtime go1.13 --from-file dashfn.zip --handler dashboard.Handler`  
> `--dependencies go.mod --namespace dictybase`  
> `-e MINIO_ACCESS_KEY=xxxxxxxx -e MINIO_SECRET_KEY=xxxxxxxx`

- check the status of function
  > `$_> kubeless function ls -n dictybase`

## Add Ingress

Create a YAML file like this [GKE Ingress example](./gke-ingress.yaml).

> `$_> kubectl apply -f gke-ingress.yaml -n dictybase`

## Endpoints

They will be available through the mapped host, for example through
`betafunc.dictybase.org` assuming the above Ingress.

**POST** `/dashboard/genomes` - Generating information for various biological
feature types (chromosome,gene etc..). It will use the `metadata.json` file to
download the gff3 file from object storage and persist the information in Redis
cache. An example `HTTP` request to this endpoint will look like this.

> `$_> curl -k -d @metadata.json https://betafunction.dictybase.local/dashboard/genomes`

**GET** `/dashboard/genomes/{taxon_id}/{type}` - Information about reference feature such as chromosome.

For reference features such as chromosome, supercontig the JSON format will be the following

> `$_> curl -k https://betafunction.dictybase.local/dashboard/genomes/44689/chromosomes`

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

> `$_> curl -k https://betafunc.dictybase.org/dashboard/genomes/44689/genes`  
> The format for the other features (gene, mRNA, pseudogene, etc)

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

**GET** `/dashboard/genomes/{taxon_id}/pseudogenes` - Information about pseudogenes.

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

The taxon ID for _D.discoideum_ is `44689`.
