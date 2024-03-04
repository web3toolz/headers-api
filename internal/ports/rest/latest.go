package rest

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/core/types"
	"net/http"
)

func HTTPLatestHeaderHandler(getLatestHeader func() *types.Header) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		header := getLatestHeader()
		payload, err := json.Marshal(header)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}
}
