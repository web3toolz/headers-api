package rest

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,

	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func WebsocketHandler(subscribe func(string, func(*types.Header)) error, unsubscribe func(string)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(``))
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		headersCh := make(chan *types.Header, 1)

		defer close(headersCh)

		pingPongTicker := time.NewTicker(30 * time.Second)

		defer pingPongTicker.Stop()

		subId := uuid.NewString()
		err = subscribe(subId, func(header *types.Header) {
			headersCh <- header
		})

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(``))
			return
		}
		defer unsubscribe(subId)

		for {
			select {
			case <-pingPongTicker.C:
				err = conn.WriteMessage(websocket.PingMessage, []byte(``))
				if err != nil {
					return
				}

			case header := <-headersCh:
				err = conn.WriteJSON(header)
				if err != nil {
					return
				}
			case <-r.Context().Done():
				return
			default:
			}
		}
	}
}
