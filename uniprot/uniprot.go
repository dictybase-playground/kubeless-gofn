package kubeless

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/kubeless/kubeless/pkg/functions"
)

const (
	// IDCacheKey is the key for storing redis hash field value
	IDCacheKey = "UNIPROT2NAME/uniprot"
	// URL is the uniprot endpoint
	URL = "https://www.uniprot.org/uniprot/?query=taxonomy:44689&columns=id,database(dictyBase),genes(PREFERRED)&format=tab"
)

func getStorage() (Storage, error) {
	var st Storage
	rhost := os.Getenv("REDIS_MASTER_SERVICE_HOST")
	rport := os.Getenv("REDIS_MASTER_SERVICE_PORT")
	shost := os.Getenv("REDIS_SLAVE_SERVICE_HOST")
	sport := os.Getenv("REDIS_SLAVE_SERVICE_PORT")
	if len(rhost) > 0 && len(shost) > 0 {
		st = NewRedisStorage(
			fmt.Sprintf("%s:%s", rhost, rport),
			fmt.Sprintf("%s:%s", shost, sport),
		)
		return st, nil
	}
	return st, fmt.Errorf("no storage backend available")
}

// CacheIds stores uniprot and gene name or identifier mapping in redis
func CacheIds(event functions.Event, ctx functions.Context) (string, error) {
	storage, err := getStorage()
	if err != nil {
		return "", err
	}
	defer storage.Close()
	resp, err := http.Get(URL)
	if err != nil {
		return "", fmt.Errorf("error in retrieving from uniprot %s", err)
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	nc := 0
	gnc := 0
	gic := 0
	urc := 0
	sc := 0
	for scanner.Scan() {
		// ignore header
		if strings.HasPrefix(scanner.Text(), "Entry") {
			continue
		}
		s := strings.Split(strings.TrimSpace(scanner.Text()), "\t")
		sl := len(s)
		switch {
		// if there is no mapping
		case sl == 1:
			nc++
		// only gene ids
		case sl == 2:
			gic++
			gs := strings.Split(s[1], ";")
			if len(gs) > 3 {
				log.Printf("unresolved line %s\t%s\n", s[0], s[1])
				urc++
			} else {
				// store in redis
				err := storage.Set(IDCacheKey, s[0], gs[0])
				if err != nil {
					return "", fmt.Errorf("error in setting the value in redis %s %s", s, err)
				}
			}
		// gene name
		case sl == 3:
			gnc++
			if strings.Contains(s[2], ";") {
				sc++
				ns := strings.Split(s[2], ";")
				// store in redis
				err := storage.Set(IDCacheKey, s[0], ns[0])
				if err != nil {
					return "", fmt.Errorf("error in setting the value in redis %s %s", s, err)
				}
			} else {
				// store in redis
				err := storage.Set(IDCacheKey, s[0], s[2])
				if err != nil {
					return "", fmt.Errorf("error in setting the value in redis %s %s", s, err)
				}
			}
		default:
			log.Printf("something seriously wrong with this line %s\n", s)
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error in scanning output %s", err)
	}
	stat := fmt.Sprintf("name:%d\tid:%d\tisoform:%d\tunresolved:%d\tnomap:%d\n", gnc, gic, sc, urc, nc)
	log.Print(stat)
	return stat, nil
}
