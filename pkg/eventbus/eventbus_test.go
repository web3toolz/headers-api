package eventbus

import (
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
)

func mockFunc(header *types.Header) {}

func createBusWithData() *EventBus {
	localFunc := func(header *types.Header) {}

	bus := EventBus{
		m:  make(map[string]map[string]*func(header *types.Header)),
		mu: sync.Mutex{},
	}

	bus.m["topic1"] = make(map[string]*func(header *types.Header))
	bus.m["topic1"]["id1"] = &localFunc

	bus.m["topic2"] = make(map[string]*func(header *types.Header))
	bus.m["topic2"]["id2"] = &localFunc
	bus.m["topic2"]["id3"] = &localFunc

	return &bus
}

func TestNew(t *testing.T) {
	bus := New()
	require.IsType(t, sync.Mutex{}, bus.mu)
	require.Equal(t, len(bus.m), 0)
}

func TestSubscribe(t *testing.T) {
	t.Run("empty map", func(t *testing.T) {
		bus := New()
		require.Equal(t, len(bus.m), 0)

		err := bus.Subscribe("topic", "id", mockFunc)

		require.NoError(t, err)

		require.Equal(t, len(bus.m), 1)
		require.Equal(t, len(bus.m["topic"]), 1)
	})

	t.Run("add to existing topic", func(t *testing.T) {
		bus := createBusWithData()

		require.Equal(t, len(bus.m), 2)

		err := bus.Subscribe("topic1", "id", mockFunc)
		require.NoError(t, err)
		require.Equal(t, len(bus.m), 2)
		require.Equal(t, len(bus.m["topic1"]), 2)
	})

	t.Run("error while adding callback to existing id", func(t *testing.T) {
		bus := createBusWithData()

		err := bus.Subscribe("topic1", "id1", mockFunc)

		require.Error(t, err)
	})
}

func TestUnsubscribe(t *testing.T) {
	t.Run("remove non-existing callback", func(t *testing.T) {
		bus := createBusWithData()

		bus.Unsubscribe("topic3", "id1")

		require.Equal(t, len(bus.m), 2)
		require.Equal(t, len(bus.m["topic1"]), 1)
		require.Equal(t, len(bus.m["topic2"]), 2)
	})
	t.Run("remove existing callback with topic deletion", func(t *testing.T) {
		bus := createBusWithData()

		bus.Unsubscribe("topic1", "id1")

		require.Equal(t, len(bus.m), 1)
		require.Equal(t, len(bus.m["topic1"]), 0)
		require.Equal(t, len(bus.m["topic2"]), 2)
	})
	t.Run("remove existing callback without topic deletion", func(t *testing.T) {
		bus := createBusWithData()

		bus.Unsubscribe("topic2", "id2")

		require.Equal(t, len(bus.m), 2)
		require.Equal(t, len(bus.m["topic1"]), 1)
		require.Equal(t, len(bus.m["topic2"]), 1)
	})
}

func TestPublish(t *testing.T) {
	t.Run("publish with no callbacks", func(t *testing.T) {
		bus := New()
		bus.Publish("topic", &types.Header{})
	})

	t.Run("publish with single callback", func(t *testing.T) {
		counter := 0
		increase := func(header *types.Header) {
			counter += 1
		}

		bus := New()
		err := bus.Subscribe("topic", "id", increase)
		require.NoError(t, err)

		bus.Publish("topic", &types.Header{})
		bus.Wait()
		require.Equal(t, counter, 1)
	})

	t.Run("publish with multiple callbacks", func(t *testing.T) {
		counter := 0
		increase := func(header *types.Header) {
			counter += 1
		}

		bus := New()
		err := bus.Subscribe("topic", "id1", increase)
		require.NoError(t, err)
		err = bus.Subscribe("topic", "id2", increase)
		require.NoError(t, err)

		bus.Publish("topic", &types.Header{})
		bus.Wait()
		require.Equal(t, counter, 2)
	})
}
