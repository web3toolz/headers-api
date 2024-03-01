package subscriber

import (
	"context"
	"errors"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"strings"
	"time"
)

type IHeadersSubscriber interface {
	Subscribe(url string, headers chan *types.Header, timeout time.Duration, interval time.Duration)
	Unsubscribe()
}

type HeadersSubscriber struct {
	ctx             context.Context
	client          *ethclient.Client
	isHTTP          bool
	publish         func(*types.Header)
	unsubscribeFunc func()
}

func NewHeadersSubscriber(
	ctx context.Context,
	url string,
	publish func(*types.Header),
) (*HeadersSubscriber, error) {
	client, err := ethclient.DialContext(ctx, url)
	isHTTP := false

	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		isHTTP = true
	} else if strings.HasPrefix(url, "ws://") || strings.HasPrefix(url, "wss://") {
		isHTTP = false
	} else {
		return nil, errors.New("failed to parse url schema")
	}

	return &HeadersSubscriber{
		ctx:     ctx,
		client:  client,
		isHTTP:  isHTTP,
		publish: publish,
	}, nil
}

func (s *HeadersSubscriber) subscribeWS() error {
	ch := make(chan *types.Header)

	sub := event.Resubscribe(time.Second*10, func(ctx context.Context) (event.Subscription, error) {
		log.Error("Trying to resubscribe")
		return s.client.SubscribeNewHead(ctx, ch)
	})

	go func() {
		for {
			select {
			case header := <-ch:
				s.publish(header)
			case <-s.ctx.Done():
				close(ch)
				sub.Unsubscribe()
				return
			}
		}
	}()

	s.unsubscribeFunc = sub.Unsubscribe

	return nil
}

func (s *HeadersSubscriber) subscribeHTTP(timeout time.Duration, interval time.Duration) error {
	//var latestBlockNumber uint64
	ticker := time.Tick(interval)
	ctx, cancelFunc := context.WithCancel(context.Background())

	s.unsubscribeFunc = cancelFunc

	getLatestHeaderFunc := func() (*types.Header, error) {
		var head *types.Header
		err := s.client.Client().CallContext(ctx, &head, "eth_getBlockByNumber", "latest", false)
		if err == nil && head == nil {
			err = ethereum.NotFound
		}
		return head, err
	}

	go func() {
		for {
			select {
			case <-ticker:
				header, err := getLatestHeaderFunc()

				if err != nil {
					log.Error("failed to get latest header", err)
				}
				// TODO: if blocks were skipped (between latestBlockNumber and header.Number.Uint64()), resync them
				if header != nil {
					//latestBlockNumber = header.Number.Uint64()
					s.publish(header)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (s *HeadersSubscriber) Subscribe(timeout time.Duration, interval time.Duration) error {
	if s.isHTTP {
		return s.subscribeHTTP(timeout, interval)
	} else {
		return s.subscribeWS()
	}
}

func (s *HeadersSubscriber) Unsubscribe() {
	s.unsubscribeFunc()
}
