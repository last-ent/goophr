package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/last-ent/goophr/librarian/common"
)

type tPayload struct {
	Token  string `json:"token"`
	Title  string `json:"title"`
	DocID  string `json:"doc_id"`
	LIndex int    `json:"line_index"`
	Index  int    `json:"token_index"`
}

type tIndex struct {
	Index  int
	LIndex int
}

func (ti *tIndex) String() string {
	return fmt.Sprintf("i: %d, li: %d", ti.Index, ti.LIndex)
}

type tIndices []tIndex

// document - key in Indices represent Line Index.
type document struct {
	Count   int
	DocID   string
	Title   string
	Indices map[int]tIndices
}

func (d *document) String() string {
	str := fmt.Sprintf("%s (%s): %d\n", d.Title, d.DocID, d.Count)
	var buffer bytes.Buffer

	for lin, tis := range d.Indices {
		var lBuffer bytes.Buffer
		for _, ti := range tis {
			lBuffer.WriteString(fmt.Sprintf("%s ", ti.String()))
		}
		buffer.WriteString(fmt.Sprintf("@%d -> %s\n", lin, lBuffer.String()))
	}
	return str + buffer.String()
}

// documentCatalog - key represents DocID.
type documentCatalog map[string]*document

func (dc *documentCatalog) String() string {
	return fmt.Sprintf("%#v", dc)
}

// tCatalog - key in map represents Token.
type tCatalog map[string]documentCatalog

func (tc *tCatalog) String() string {
	return fmt.Sprintf("%#v", tc)
}

type tcCallback struct {
	Token string
	Ch    chan tcMsg
}

type tcMsg struct {
	Token string
	DC    documentCatalog
}

// pProcessCh is used to process /index's payload and start process to add the token to catalog (tCatalog).
var pProcessCh chan tPayload

// tcGet is used to retrieve a token's catalog (documentCatalog).
var tcGet chan tcCallback

func StartIndexSystem() {
	pProcessCh = make(chan tPayload, 100)
	tcGet = make(chan tcCallback, 20)
	go tIndexer(pProcessCh, tcGet)
}

func returnDocumentCatalog(msg tcCallback, store tCatalog) {
	dc := store[msg.Token]
	msg.Ch <- tcMsg{
		DC:    dc,
		Token: msg.Token,
	}
}

func getDocumentCatalog(token string, store tCatalog) documentCatalog {
	dc, exists := store[token]
	if !exists {
		dc = documentCatalog{}
		store[token] = dc
	}
	return dc
}

func getDocumentForDocID(docID string, docCatalog documentCatalog) *document {

	doc, exists := docCatalog[docID]
	if !exists {
		doc = &document{
			DocID:   docID,
			Indices: map[int]tIndices{},
		}
		docCatalog[docID] = doc
	}

	return doc
}

func increaseDocumentScore(payload tPayload, store tCatalog) {
	docCatalog := getDocumentCatalog(payload.Token, store)
	doc := getDocumentForDocID(payload.DocID, docCatalog)
	doc.Title = payload.Title

	tin := tIndex{
		Index:  payload.Index,
		LIndex: payload.LIndex,
	}
	doc.Indices[tin.LIndex] = append(doc.Indices[tin.LIndex], tin)
	doc.Count++
}

// tIndexer maintains a catalog of all tokens along with where they occur within documents.
func tIndexer(ch chan tPayload, callback chan tcCallback) {
	store := tCatalog{}
	for {
		select {
		case msg := <-callback:
			returnDocumentCatalog(msg, store)

		case payload := <-ch:
			increaseDocumentScore(payload, store)
		}
	}
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	if common.SignalIfMethodNotAllowed(w, r, "POST") {
		return
	}

	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()

	var tp tPayload
	decoder.Decode(&tp)
	log.Println("Token received", fmt.Sprintf("%#v", tp))

	pProcessCh <- tp

	w.Write([]byte(`{"code": 200, "msg": "Tokens are being added to index."}`))
}
