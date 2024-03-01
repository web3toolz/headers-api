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
	client          *ethclient.Client
	isHTTP          bool
	headers         chan *types.Header
	interval        time.Duration
	timeout         time.Duration
	sub             ethereum.Subscription
	unsubscribeFunc func()
}

func NewHeadersSubscriber(
	ctx context.Context,
	url string,
	headers chan *types.Header,
	timeout time.Duration,
	interval time.Duration,
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
		client:   client,
		isHTTP:   isHTTP,
		headers:  headers,
		interval: interval,
		timeout:  timeout,
	}, nil
}

func (s *HeadersSubscriber) subscribeWS() error {
	s.sub = event.Resubscribe(time.Second*10, func(ctx context.Context) (event.Subscription, error) {
		log.Error("Trying to resubscribe")
		return s.client.SubscribeNewHead(ctx, s.headers)
	})
	s.unsubscribeFunc = s.sub.Unsubscribe
	return nil
}

func (s *HeadersSubscriber) subscribeHTTP() error {
	//var latestBlockNumber uint64
	ticker := time.Tick(s.interval)
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
					s.headers <- header
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (s *HeadersSubscriber) Subscribe() error {
	if s.isHTTP {
		return s.subscribeHTTP()
	} else {
		return s.subscribeWS()
	}
}

func (s *HeadersSubscriber) Unsubscribe() {
	s.unsubscribeFunc()
}
