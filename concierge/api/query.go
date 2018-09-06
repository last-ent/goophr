package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"

	"github.com/last-ent/goophr/concierge/common"
)

var librarianEndpoints = map[string]string{}

func init() {
	librarianEndpoints["a-m"] = os.Getenv("LIB_A_M")
	librarianEndpoints["n-z"] = os.Getenv("LIB_N_Z")
	librarianEndpoints["*"] = os.Getenv("LIB_OTHERS")
}

type docs struct {
	DocID string `json:"doc_id"`
	Score int    `json:"doc_score"`
}

type queryResult struct {
	Count int    `json:"count"`
	Data  []docs `json:"data"`
}

func queryLibrarian(endpoint string, st []byte, ch chan<- queryResult) {
	resp, err := http.Post(
		endpoint+"/query",
		"application/json",
		bytes.NewBuffer(st),
	)
	if err != nil {
		common.Warn(fmt.Sprintf("%s -> %+v", endpoint, err))
		ch <- queryResult{}
		return
	}

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	var qr queryResult
	json.Unmarshal(body, &qr)
	log.Println(fmt.Sprintf("%s -> %#v", endpoint, qr))
	ch <- qr
}

func getResultsMap(ch <-chan queryResult) map[string]int {
	results := []docs{}
	for range librarianEndpoints {
		if result := <-ch; result.Count > 0 {
			results = append(results, result.Data...)
		}
	}

	resultsMap := map[string]int{}
	for _, doc := range results {
		docID := doc.DocID
		score := doc.Score
		if _, exists := resultsMap[docID]; !exists {
			resultsMap[docID] = 0
		}
		resultsMap[docID] = resultsMap[docID] + score
	}

	return resultsMap
}

func decodeSearchTerms(body io.ReadCloser, w http.ResponseWriter) ([]string, error) {
	decoder := json.NewDecoder(body)

	var searchTerms []string
	if err := decoder.Decode(&searchTerms); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"code": 400, "msg": "Unable to parse payload."}`))
		return nil, err
	}

	return searchTerms, nil
}

func launchSearchQueries(st []byte, ch chan<- queryResult) {
	for _, le := range librarianEndpoints {
		func(endpoint string) {
			go queryLibrarian(endpoint, st, ch)
		}(le)
	}
}

func QueryHandler(w http.ResponseWriter, r *http.Request) {
	if common.SignalIfMethodNotAllowed(w, r, "POST") {
		return
	}

	defer r.Body.Close()
	searchTerms, err := decodeSearchTerms(r.Body, w)
	if err != nil {
		common.Warn("Unable to parse request." + err.Error())
		return
	}

	st, err := json.Marshal(searchTerms)
	if err != nil {
		panic(err)
	}

	resultsCh := make(chan queryResult)
	launchSearchQueries(st, resultsCh)

	resultsMap := getResultsMap(resultsCh)
	close(resultsCh)

	sortedResults := sortResults(resultsMap)

	payload, _ := json.Marshal(sortedResults)
	w.Header().Add("Content-Type", "application/json")
	w.Write(payload)

	fmt.Println(fmt.Sprintf("%#v", sortedResults))
}

func aggregateResultsScore(resultsMap map[string]int) map[int][]document {
	scoreMap := map[int][]document{}
	ch := make(chan document)
	defer close(ch)

	for docID, score := range resultsMap {
		if _, exists := scoreMap[score]; !exists {
			scoreMap[score] = []document{}
		}

		dGetCh <- dMsg{
			DocID: docID,
			Ch:    ch,
		}
		doc := <-ch

		scoreMap[score] = append(scoreMap[score], doc)
	}

	return scoreMap
}

func getResultsSortedByScore(scores []int, scoreMap map[int][]document) []document {
	sortedResults := []document{}
	for _, score := range scores {
		resDocs := scoreMap[score]
		sortedResults = append(sortedResults, resDocs...)
	}
	return sortedResults
}

func sortResults(resultsMap map[string]int) []document {
	scoreMap := aggregateResultsScore(resultsMap)

	scores := []int{}
	for score := range scoreMap {
		scores = append(scores, score)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(scores)))

	return getResultsSortedByScore(scores, scoreMap)
}
