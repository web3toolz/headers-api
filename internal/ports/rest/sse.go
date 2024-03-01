package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"net/http"
)

func SSEHandler(subscribe func(string, func(header *types.Header)) error, unsubscribe func(string)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		headersCh := make(chan *types.Header, 1)
		defer close(headersCh)

		subId := uuid.NewString()
		err := subscribe(subId, func(header *types.Header) {
			headersCh <- header
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(``))
			return
		}
		defer unsubscribe(subId)

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		for {
			select {
			case header := <-headersCh:
				var buf bytes.Buffer
				enc := json.NewEncoder(&buf)
				_ = enc.Encode(header)
				_, _ = fmt.Fprint(w, buf.String())
				w.(http.Flusher).Flush()
			case <-r.Context().Done():
				return
			default:

			}
		}

	}
}
