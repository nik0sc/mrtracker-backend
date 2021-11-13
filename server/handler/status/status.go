package status

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

const (
	envGitRev = "GIT_REV"
)

type Handler struct{}

type result struct {
	Version string `json:"version"`
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")

	res := result{
		Version: os.Getenv(envGitRev),
	}

	b, err := json.Marshal(res)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("error: marshal of status result: %s", err.Error())
		return
	}

	_, err = w.Write(b)
	if err != nil {
		log.Printf("error: writing response: %s", err.Error())
		return
	}

	return
}
