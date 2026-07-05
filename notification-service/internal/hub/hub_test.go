package hub

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bober-17/meeting-room-booking-system/notification-service/internal/model"
)

var testNotification = model.Notification{ID: "n1", Type: model.TypeBookingCreated}

func TestHub_BroadcastDelivered(t *testing.T) {
	h := NewHub()
	ch, unsub, err := h.Subscribe("user1")
	require.NoError(t, err)
	defer unsub()

	h.Broadcast("user1", testNotification)

	select {
	case n := <-ch:
		assert.Equal(t, testNotification, n)
	default:
		t.Fatal("expected notification, got nothing")
	}
}

func TestHub_BroadcastNoSubscribers(t *testing.T) {
	h := NewHub()
	h.Broadcast("unknown_user", testNotification) // не должно паниковать
}

func TestHub_UnsubscribeStopsDelivery(t *testing.T) {
	h := NewHub()
	_, unsub, err := h.Subscribe("user1")
	require.NoError(t, err)

	unsub()
	h.Broadcast("user1", testNotification)

	h.mu.Lock()
	_, exists := h.subs["user1"]
	h.mu.Unlock()
	assert.False(t, exists)
}

// Покрывает исправленный баг: при удалении не последнего элемента
// nil не должен попадать в середину среза подписчиков.
func TestHub_UnsubscribeMiddleSubscriber(t *testing.T) {
	h := NewHub()

	ch1, unsub1, err := h.Subscribe("user1")
	require.NoError(t, err)
	ch2, unsub2, err := h.Subscribe("user1")
	require.NoError(t, err)
	ch3, unsub3, err := h.Subscribe("user1")
	require.NoError(t, err)
	defer unsub3()

	unsub1() // удаляем первый (не последний) — раньше nil оказывался внутри среза

	h.Broadcast("user1", testNotification)

	select {
	case <-ch1:
		t.Fatal("ch1 got notification after unsubscribe")
	default:
	}
	select {
	case n := <-ch2:
		assert.Equal(t, testNotification, n)
	default:
		t.Fatal("ch2 did not receive notification")
	}
	select {
	case n := <-ch3:
		assert.Equal(t, testNotification, n)
	default:
		t.Fatal("ch3 did not receive notification")
	}

	h.mu.Lock()
	for _, c := range h.subs["user1"] {
		assert.NotNil(t, c)
	}
	h.mu.Unlock()

	unsub2()
}

func TestHub_MultipleUsersIsolated(t *testing.T) {
	h := NewHub()
	ch1, unsub1, err := h.Subscribe("user1")
	require.NoError(t, err)
	defer unsub1()
	ch2, unsub2, err := h.Subscribe("user2")
	require.NoError(t, err)
	defer unsub2()

	h.Broadcast("user1", testNotification)

	select {
	case <-ch1:
	default:
		t.Fatal("user1 channel should receive notification")
	}
	select {
	case <-ch2:
		t.Fatal("user2 channel should not receive notification for user1")
	default:
	}
}

func TestHub_MaxConnectionsLimit(t *testing.T) {
	h := NewHub()
	var unsubs []func()

	for i := range maxSubsPerUser {
		_, unsub, err := h.Subscribe("user1")
		require.NoError(t, err, "connection %d should succeed", i+1)
		unsubs = append(unsubs, unsub)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	_, _, err := h.Subscribe("user1")
	assert.Error(t, err)
}

func TestHub_LimitRestoredAfterUnsubscribe(t *testing.T) {
	h := NewHub()
	var unsubs []func()

	for range maxSubsPerUser {
		_, unsub, err := h.Subscribe("user1")
		require.NoError(t, err)
		unsubs = append(unsubs, unsub)
	}

	// освобождаем одно место
	unsubs[0]()
	unsubs = unsubs[1:]
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	_, unsub, err := h.Subscribe("user1")
	assert.NoError(t, err)
	unsub()
}
