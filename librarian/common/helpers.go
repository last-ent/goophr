package common

import (
	"fmt"
	"log"
	"net/http"
)

func Log(msg string) {
	log.Println("INFO - ", msg)
}

func Warn(msg string) {
	log.Println("---------------------------")
	log.Println(fmt.Sprintf("WARN: %s", msg))
	log.Println("---------------------------")
}

// SignalIfMethodNotAllowed check if the method on request is allowed and if not, writes to ResponseWriter
func SignalIfMethodNotAllowed(w http.ResponseWriter, r *http.Request, method string) (notAllowed bool) {
	notAllowed = r.Method != method
	if notAllowed {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte(`{"code": 405, "msg": "Method Not Allowed."}`))
	}
	return
}
