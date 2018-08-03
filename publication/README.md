# publication function
A golang based [kubeless](https://kubeless.io) function to deploy in kubernetes cluster.

## Dependencies
It is recommended to have [redis](https://redis.io) installed for caching the
function output. The instructions are given
[here](https://github.com/dictyBase/Migration/blob/master/deploy.md#redis).

## Deploy the function
> `$_> zip pubfn.zip *.go Gopkg.toml`   
> `$_>  kubeless function deploy \`   
> `pubfn --runtime go1.10 --from-file pubfn.zip --handler publication.Handler`   
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
