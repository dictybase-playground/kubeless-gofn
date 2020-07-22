# publication function

A Golang-based [Kubeless](https://kubeless.io) function to deploy in a Kubernetes cluster.

## Dependencies

It is recommended to have [Redis](https://redis.io) installed for caching the
function output. [Instructions](https://dictybase-docker.github.io/developer-docs/deployment/redis/)

This requires at least `v1.0.7` of Kubeless.

## Deploy the function

> `$_> zip pubfn.zip *.go go.mod`  
> `$_> kubeless function deploy \`  
> `pubfn --runtime go1.13 --from-file pubfn.zip --handler publication.Handler`  
> `--dependencies go.mod --namespace dictybase`

- check the status of function
  > `$_> kubeless function ls -n dictybase`

## Add Ingress

[GKE Ingress Dev example](./gke-ingress-dev.yaml)

> `$_> kubectl apply -f gke-ingress-dev.yaml -n dictybase`

## Endpoints

It will be available through the mapped host, for example through
`betafunc.dictybase.org` assuming the above function.

**GET** `/publications/{pubmed-id}` - Information about a publication with given Pubmed ID.

> `$_> curl -k https://betafunc.dictybase.local/publications/30048658`

The expected output will be in the following structure

```json
{
  "links": {
    "self": "https://betafunc.dictybase.local/publications/16769729"
  },
  "data": {
    "attributes": {
      "authors": [
        {
          "initials": "A",
          "full_name": "Kortholt A",
          "last_name": "Kortholt",
          "first_name": "Arjan"
        },
        {
          "initials": "H",
          "full_name": "Rehmann H",
          "last_name": "Rehmann",
          "first_name": "Holger"
        },
        {
          "initials": "H",
          "full_name": "Kae H",
          "last_name": "Kae",
          "first_name": "Helmut"
        },
        {
          "initials": "L",
          "full_name": "Bosgraaf L",
          "last_name": "Bosgraaf",
          "first_name": "Leonard"
        },
        {
          "initials": "I",
          "full_name": "Keizer-Gunnink I",
          "last_name": "Keizer-Gunnink",
          "first_name": "Ineke"
        },
        {
          "initials": "G",
          "full_name": "Weeks G",
          "last_name": "Weeks",
          "first_name": "Gerald"
        },
        {
          "initials": "A",
          "full_name": "Wittinghofer A",
          "last_name": "Wittinghofer",
          "first_name": "Alfred"
        },
        {
          "initials": "PJ",
          "full_name": "Van Haastert PJ",
          "last_name": "Van Haastert",
          "first_name": "Peter J M"
        }
      ],
      "publication_date": "2006-06-12",
      "issue": 1341878,
      "pub_type": "Research Support, Non-U.S. Gov't",
      "status": "published",
      "source": "MED",
      "title": "Characterization of the GbpD-activated Rap1 pathway...",
      "abstract": "The regulation of cell polarity...",
      "doi": "10.1074/jbc.m600804200",
      "full_text_url": "https://doi.org/10.1074/jbc.M600804200",
      "pubmed_url": "https://pubmed.gov/16769729",
      "journal": "The Journal of biological chemistry",
      "issn": "0021-9258",
      "page": "23367-23376",
      "pubmed": "16769729"
    },
    "id": "16769729",
    "type": "publications"
  }
}
```
