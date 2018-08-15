package kubeless

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/dictyBase/apihelpers/apherror"
	"github.com/go-chi/chi"
	"github.com/kubeless/kubeless/pkg/functions"
	"github.com/minio/minio-go"
	"github.com/spacemonkeygo/errors"
	"github.com/spacemonkeygo/errors/errhttp"
)

var (
	titleErrKey   = errors.GenSym()
	pointerErrKey = errors.GenSym()
	paramErrKey   = errors.GenSym()
	hasAPIServer  = false
	apiPort       = 33333
)

const KEY_PREFIX = "dashboard"

type MetaData struct {
	TaxonId    string `json:"taxon_id"`
	SciName    string `json:"scientific_name,omitempty"`
	CommonName string `json:"common_name,omitempty"`
	Rank       string `json:"rank,omitempty"`
	Bucket     string `json:"bucket"`
	File       string `json:"file"`
}

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

func init() {
	r := chi.NewRouter()
	st, err := getStorage()
	if err != nil {
		return
	}
	hasAPIServer = true
	r.Get("/dashboard/genomes/{taxonid}/{biotype}", func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s-%s", KEY_PREFIX, chi.URLParam(r, "taxonid"))
		field := chi.URLParam(r, "biotype")
		if !st.IsExist(key, field) {
			http.Error(w, fmt.Sprintf("key %s does not exist", key), http.StatusNotFound)
			return
		}
		payload, err := st.Get(key, field)
		if err != nil {
			http.Error(w, fmt.Sprintf("error %s in retrieving %s", err, key), http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "%s", payload)
	})
	go func() {
		lport := fmt.Sprintf(":%d", apiPort)
		log.Printf("starting localhost server on %s", lport)
		http.ListenAndServe(lport, r)
	}()
}

func Handler(event functions.Event, ctx functions.Context) (string, error) {
	r := event.Extensions.Request
	w := event.Extensions.Response
	w.Header().Set("Content-Type", "application/vnd.api+json")
	storage, err := getStorage()
	if err != nil {
		return internalServerError(
			w,
			fmt.Sprintf("error %s in getting storage handler", err),
		)
	}
	defer storage.Close()
	// HTTP POST to "/genomes" expects a gff3 and metadata.json file for uploading
	//	---- metadata structure
	//  {
	//		"taxon_id": "....",
	//		"scientific_name": "....",
	//		"common_name": "....",
	//		"rank: "..."
	//  }
	if r.Method == "POST" && r.URL.Path == "/genomes" {
		meta := &MetaData{}
		if err := json.Unmarshal([]byte(event.Data), meta); err != nil {
			return internalServerError(
				w,
				fmt.Sprintf("error in unmarshalling metadata %s", err),
			)
		}
		log.Printf("received metadata with taxon id %s", meta.TaxonId)
		log.Printf("going to fetch file %s from bucket %s", meta.File, meta.Bucket)
		s3Client, err := minio.New(
			fmt.Sprintf(
				"%s:%s",
				os.Getenv("MINIO_SERVICE_HOST"),
				os.Getenv("MINIO_SERVICE_PORT"),
			),
			os.Getenv("MINIO_ACCESS_KEY"),
			os.Getenv("MINIO_SECRET_KEY"),
			false,
		)
		if err != nil {
			return internalServerError(
				w,
				fmt.Sprintf("error %s in getting minio s3Client handler", err),
			)
		}
		gf, err := s3Client.GetObject(meta.Bucket, meta.File, minio.GetObjectOptions{})
		if err != nil {
			return internalServerError(
				w,
				fmt.Sprintf("error in fetching file from s3 %s", err),
			)
		}
		log.Println("storing information of gff3 file....")
		key := fmt.Sprintf("%s-%s", KEY_PREFIX, meta.TaxonId)
		err = storeGFFInforamtion(
			gf,
			storage,
			key,
			[]string{"chromosome", "gene", "pseudogene"}...,
		)
		if err != nil {
			return internalServerError(
				w,
				fmt.Sprintf("error in saving file %s", err),
			)
		}
		mb, err := json.Marshal(meta)
		if err != nil {
			return internalServerError(
				w,
				fmt.Sprintf("error in marshaling metadata %s", err),
			)
		}
		// store meta information
		if err := storage.Set(key, "organism", string(mb)); err != nil {
			return internalServerError(
				w,
				fmt.Sprintf("error in saving metadata %s", err),
			)
		}
		return fmt.Sprintf("%d OK", http.StatusOK), nil
	}
	if r.Method == "GET" {
		if !hasAPIServer {
			return internalServerError(
				w,
				"api server have not been started",
			)
		}
		url := fmt.Sprintf("http://localhost:%d%s", apiPort, r.Header.Get("X-Original-Uri"))
		resp, err := http.Get(url)
		if err != nil {
			return httpError(
				w,
				resp.StatusCode,
				fmt.Sprintf("error in retrieving data %s", err),
			)
		}
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return internalServerError(
				w,
				fmt.Sprintf("error in reading response %s", err),
			)
		}
		if resp.StatusCode >= 400 {
			return httpError(
				w,
				resp.StatusCode,
				string(b),
			)
		}
		return string(b), nil
	}
	json, status, err := JSONAPIError(
		apherror.ErrMethodNotAllowed.New(
			"%s not allowed",
			r.Method,
		))
	w.WriteHeader(status)
	return json, err
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
			"creator": "kubless gofn error",
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

func internalServerError(w http.ResponseWriter, msg string) (string, error) {
	txt := http.StatusText(http.StatusInternalServerError)
	err := apherror.Errhttp.NewClass(
		txt,
		errhttp.SetStatusCode(http.StatusInternalServerError),
	)
	err.MustAddData(titleErrKey, txt)
	str, _, errn := JSONAPIError(err.New(msg))
	w.WriteHeader(http.StatusInternalServerError)
	return str, errn
}

func httpError(w http.ResponseWriter, code int, msg string) (string, error) {
	err := apherror.Errhttp.NewClass(
		http.StatusText(code),
		errhttp.SetStatusCode(code),
	)
	err.MustAddData(titleErrKey, "http error")
	str, _, errn := JSONAPIError(err.New(msg))
	w.WriteHeader(code)
	return str, errn
}
