package kubeless

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/dictyBase/apihelpers/apherror"
	"github.com/fatih/structs"
	"github.com/kubeless/kubeless/pkg/functions"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/errhttp"
)

var (
	pubRegxp      = regexp.MustCompile(`^/(\d+)$`)
	titleErrKey   = errors.GenSym()
	pointerErrKey = errors.GenSym()
	paramErrKey   = errors.GenSym()
)

const REDIS_KEY = "PUBLICATION_KEY"

type PubJsonAPI struct {
	Data  *PubData `json:"data"`
	Links *Links   `json:"links"`
}

type Links struct {
	Self string `json:"self"`
}

type PubData struct {
	Type       string       `json:"type"`
	ID         string       `json:"id"`
	Attributes *Publication `json:"attributes"`
}

type Author struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	Initials  string `json:"initials"`
}

type Publication struct {
	Abstract       string    `json:"abstract"`
	Doi            string    `json:"doi,omitempty"`
	FullTextURL    string    `json:"full_text_url,omitempty"`
	PubmedURL      string    `json:"pubmed_url"`
	Journal        string    `json:"journal"`
	Issn           string    `json:"issn,omitempty"`
	Page           string    `json:"page,omitempty"`
	Pubmed         string    `json:"pubmed"`
	Title          string    `json:"title"`
	Source         string    `json:"source"`
	Status         string    `json:"status"`
	PubType        string    `json:"pub_type"`
	Issue          string    `json:"issue"`
	Volume         string    `json:"volume"`
	JournalIssueId int64     `json:"journalIssueId,omitempty"`
	PublishedDate  string    `json:"publication_date"`
	Authors        []*Author `json:"authors"`
}

type EuroPMC struct {
	HitCount       int64  `json:"hitCount"`
	NextCursorMark string `json:"nextCursorMark"`
	Request        struct {
		CursorMark string `json:"cursorMark"`
		PageSize   int64  `json:"pageSize"`
		Query      string `json:"query"`
		ResultType string `json:"resultType"`
		Sort       string `json:"sort"`
		Synonym    bool   `json:"synonym"`
	} `json:"request"`
	ResultList struct {
		Result []struct {
			AbstractText string `json:"abstractText"`
			Affiliation  string `json:"affiliation"`
			AuthMan      string `json:"authMan"`
			AuthorList   struct {
				Author []struct {
					Affiliation string `json:"affiliation"`
					FirstName   string `json:"firstName"`
					FullName    string `json:"fullName"`
					Initials    string `json:"initials"`
					LastName    string `json:"lastName"`
				} `json:"author"`
			} `json:"authorList"`
			AuthorString              string `json:"authorString"`
			CitedByCount              int64  `json:"citedByCount"`
			DateOfCreation            string `json:"dateOfCreation"`
			DateOfRevision            string `json:"dateOfRevision"`
			Doi                       string `json:"doi"`
			ElectronicPublicationDate string `json:"electronicPublicationDate"`
			EpmcAuthMan               string `json:"epmcAuthMan"`
			FirstPublicationDate      string `json:"firstPublicationDate"`
			FullTextURLList           struct {
				FullTextURL []struct {
					Availability     string `json:"availability"`
					AvailabilityCode string `json:"availabilityCode"`
					DocumentStyle    string `json:"documentStyle"`
					Site             string `json:"site"`
					URL              string `json:"url"`
				} `json:"fullTextUrl"`
			} `json:"fullTextUrlList"`
			HasBook               string `json:"hasBook"`
			HasDBCrossReferences  string `json:"hasDbCrossReferences"`
			HasLabsLinks          string `json:"hasLabsLinks"`
			HasPDF                string `json:"hasPDF"`
			HasReferences         string `json:"hasReferences"`
			HasTMAccessionNumbers string `json:"hasTMAccessionNumbers"`
			HasTextMinedTerms     string `json:"hasTextMinedTerms"`
			ID                    string `json:"id"`
			InEPMC                string `json:"inEPMC"`
			InPMC                 string `json:"inPMC"`
			IsOpenAccess          string `json:"isOpenAccess"`
			JournalInfo           struct {
				DateOfPublication string `json:"dateOfPublication"`
				Journal           struct {
					Essn                string `json:"essn"`
					Isoabbreviation     string `json:"isoabbreviation"`
					Issn                string `json:"issn"`
					MedlineAbbreviation string `json:"medlineAbbreviation"`
					Nlmid               string `json:"nlmid"`
					Title               string `json:"title"`
				} `json:"journal"`
				JournalIssueID       int64  `json:"journalIssueId"`
				MonthOfPublication   int64  `json:"monthOfPublication"`
				PrintPublicationDate string `json:"printPublicationDate"`
				YearOfPublication    int64  `json:"yearOfPublication"`
				Issue                string `json:"issue"`
				Volume               string `json:"volume"`
			} `json:"journalInfo"`
			KeywordList struct {
				Keyword []string `json:"keyword"`
			} `json:"keywordList"`
			Language    string `json:"language"`
			NihAuthMan  string `json:"nihAuthMan"`
			PageInfo    string `json:"pageInfo"`
			Pmid        string `json:"pmid"`
			PubModel    string `json:"pubModel"`
			PubTypeList struct {
				PubType []string `json:"pubType"`
			} `json:"pubTypeList"`
			PubYear string `json:"pubYear"`
			Source  string `json:"source"`
			Title   string `json:"title"`
		} `json:"result"`
	} `json:"resultList"`
	Version string `json:"version"`
}

func getRedisConnection() Cacher {
	var cache Cacher
	rhost := os.Getenv("REDIS_MASTER_SERVICE_HOST")
	rport := os.Getenv("REDIS_MASTER_SERVICE_PORT")
	if len(rhost) > 0 && len(rport) > 0 {
		shost := os.Getenv("REDIS_SLAVE_SERVICE_HOST")
		sport := os.Getenv("REDIS_SLAVE_SERVICE_PORT")
		if len(shost) > 0 && len(sport) > 0 {
			cache = NewRedisReplicationCache(
				fmt.Sprintf("%s:%s", rhost, rport),
				fmt.Sprintf("%s:%s", shost, sport),
			)
			log.Println("connected to redis with replication")
			return cache
		}
		log.Println("connected to redis master")
		cache = NewRedisCache(fmt.Sprintf("%s:%s", rhost, rport))
	}
	return cache
}

var cache = getRedisConnection()

func Handler(event functions.Event, ctx functions.Context) (string, error) {
	r := event.Extensions.Request
	w := event.Extensions.Response
	w.Header().Set("Content-Type", "application/vnd.api+json")
	if r.Method != "GET" {
		json, status, err := JSONAPIError(
			apherror.ErrMethodNotAllowed.New(
				"%s not allowed",
				r.Method,
			))
		w.WriteHeader(status)
		return json, err
	}
	m := pubRegxp.FindStringSubmatch(r.URL.Path)
	if len(m) == 0 {
		json, status, err := JSONAPIError(
			apherror.ErrNotFound.New(
				"no route for %s",
				generateLink(r),
			),
		)
		w.WriteHeader(status)
		return json, err
	}
	rkey := fmt.Sprintf(
		"%s%s",
		REDIS_KEY, r.URL.Path,
	)
	if cache != nil {
		if cache.IsExist(rkey) {
			v, err := cache.Get(rkey)
			if err == nil {
				log.Printf("got key %s from cache", rkey)
				return string(v), nil
			}
			log.Printf("error in getting existing key %s %s", rkey, err)
		}
	} else {
		log.Println("no redis cache")
	}
	url := fmt.Sprintf(
		"%s?format=json&resultType=core&query=ext_id:%s",
		"https://www.ebi.ac.uk/europepmc/webservices/rest/search",
		m[1],
	)
	res, err := http.Get(url)
	if err != nil {
		json, _, err := JSONAPIError(
			apherror.Errhttp.NewClass(
				res.Status,
				errhttp.SetStatusCode(res.StatusCode),
			).New("error %s in fetching %s", err.Error(), m[1]),
		)
		w.WriteHeader(res.StatusCode)
		return json, err
	}
	defer res.Body.Close()
	epmc := &EuroPMC{}
	err = json.NewDecoder(res.Body).Decode(epmc)
	if err != nil {
		json, _, err := JSONAPIError(
			apherror.Errhttp.NewClass(
				http.StatusText(http.StatusInternalServerError),
				errhttp.SetStatusCode(http.StatusInternalServerError),
			).New("error in decoding body %s", err.Error()),
		)
		w.WriteHeader(http.StatusInternalServerError)
		return json, err
	}
	b, err := json.Marshal(&PubJsonAPI{
		Data: &PubData{
			Type:       "publications",
			ID:         m[1],
			Attributes: EuroPMC2Pub(epmc),
		},
		Links: &Links{
			Self: generateLink(r),
		},
	})
	if err != nil {
		json, status, err := JSONAPIError(
			apherror.ErrStructMarshal.New(
				"error in making final response %s",
				err.Error(),
			),
		)
		w.WriteHeader(status)
		return json, err

	}
	if cache != nil {
		if err := cache.Set(rkey, b, 30*24*time.Hour); err != nil {
			log.Printf("error in setting key %s %s", rkey, err)
		} else {
			log.Printf("stored key %s in cache", rkey)
		}
	}
	return string(b), nil
}

func generateLink(r *http.Request) string {
	return fmt.Sprintf(
		"%s://%s%s",
		r.Header.Get("X-Forwarded-Proto"),
		r.Host,
		r.Header.Get("X-Original-Uri"),
	)
}

//JSONAPIError generate JSONAPI formatted http error from an error object
func JSONAPIError(err error) (string, int, error) {
	status := errhttp.GetStatusCode(err, http.StatusInternalServerError)
	title, _ := errors.GetData(err, titleErrKey).(string)
	jsnErr := apherror.Error{
		Status: strconv.Itoa(status),
		Title:  title,
		Detail: errhttp.GetErrorBody(err),
		Meta: map[string]interface{}{
			"creator": "kubeless gofn error",
		},
	}
	errSource := new(apherror.ErrorSource)
	pointer, ok := errors.GetData(err, pointerErrKey).(string)
	if ok {
		errSource.Pointer = pointer
	}
	param, ok := errors.GetData(err, paramErrKey).(string)
	if ok {
		errSource.Parameter = param
	}
	jsnErr.Source = errSource
	ct, encErr := json.Marshal(apherror.HTTPError{Errors: []apherror.Error{jsnErr}})
	if encErr != nil {
		return "", http.StatusInternalServerError, encErr
	}
	return string(ct), status, nil
}

func EuroPMC2Pub(pmc *EuroPMC) *Publication {
	if len(pmc.ResultList.Result) < 1 {
		log.Println("no results found for publication")
		return &Publication{}
	}
	result := pmc.ResultList.Result[0]
	pub := &Publication{
		Abstract:       result.AbstractText,
		Doi:            result.Doi,
		Journal:        result.JournalInfo.Journal.Title,
		Issn:           result.JournalInfo.Journal.Issn,
		Page:           result.PageInfo,
		Pubmed:         result.Pmid,
		PubmedURL:      fmt.Sprintf("https://pubmed.gov/%s", result.Pmid),
		Title:          result.Title,
		Source:         result.Source,
		Status:         "published",
		PubType:        "Journal Article",
		Issue:          result.JournalInfo.Issue,
		Volume:         result.JournalInfo.Volume,
		JournalIssueId: result.JournalInfo.JournalIssueID,
		PublishedDate:  result.FirstPublicationDate,
	}
	rstruct := structs.New(result)
	if !rstruct.Field("FullTextURLList").IsZero() {
		pub.FullTextURL = result.FullTextURLList.FullTextURL[0].URL
	}
	if !rstruct.Field("PubTypeList").IsZero() {
		pub.PubType = result.PubTypeList.PubType[0]
	}
	var authors []*Author
	for _, a := range result.AuthorList.Author {
		authors = append(authors, &Author{
			FirstName: a.FirstName,
			LastName:  a.LastName,
			FullName:  a.FullName,
			Initials:  a.Initials,
		})
	}
	pub.Authors = authors
	return pub
}
