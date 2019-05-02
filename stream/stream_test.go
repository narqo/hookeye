package stream

import (
	"bytes"
	"context"
	"sync"
	"testing"
)

func TestStream_Subscribe_Groups(t *testing.T) {
	ctx := context.Background()

	stream := New()

	group1 := make(chan *Message, 1)
	stream.SubscribeN("topic1", ProcessorFunc(func(ctx context.Context, msg *Message) error {
		group1 <- msg
		return nil
	}), 1)

	group2 := make(chan *Message, 1)
	stream.SubscribeN("topic1", ProcessorFunc(func(ctx context.Context, msg *Message) error {
		group2 <- msg
		return nil
	}), 1)

	assertNoError(t, stream.Push(ctx, "topic1", []byte{'A'}))
	assertNoError(t, stream.Push(ctx, "topic1", []byte{'B'}))
	assertNoError(t, stream.Push(ctx, "topic1", []byte{'C'}))

	assertNoError(t, stream.Push(ctx, "topic2", []byte{'X'}))

	assertGroup := func(t *testing.T, wg *sync.WaitGroup, group <-chan *Message) {
		wg.Add(1)

		go func() {
			var i int
			for msg := range group {
				if want, got := []byte{byte('A' + i)}, msg.Data; !bytes.Equal(want, got) {
					t.Errorf("(i %v): want %v, got %v", i, want, got)
				}
				i++
			}

			wg.Done()
		}()
	}

	var wg sync.WaitGroup
	assertGroup(t, &wg, group1)
	assertGroup(t, &wg, group2)

	stream.Stop()

	close(group1)
	close(group2)

	wg.Wait()
}

func TestStream_Compact(t *testing.T) {
	ctx := context.Background()

	stream := New()

	// waiter helps to linearize concurrent operations
	waiter := make(chan struct{}, 1)

	group1 := make(chan *Message)
	stream.SubscribeN("topic1", ProcessorFunc(func(ctx context.Context, msg *Message) error {
		waiter <- struct{}{}
		group1 <- msg
		return nil
	}), 1)

	assertNoError(t, stream.Push(ctx, "topic1", []byte{'A'}))
	assertNoError(t, stream.Push(ctx, "topic1", []byte{'B'}))

	// wait "A" to be seen in group1 subscriber
	// note, "A" isn't fully processed yet, because sending to group1 above is blocked
	<-waiter

	// compact must remove message "A" only, as it's the only one that is processed (see waiter's note above)
	stream.Compact()

	topic1 := stream.Topic("topic1")

	_, ok := topic1.DataAt(0)
	if ok {
		t.Errorf("offset 0: want to not exist, got %v", ok)
	}

	data, ok := topic1.DataAt(1)
	if !ok {
		t.Errorf("offset 1: want to exist, got %v", ok)
	} else if want, got := []byte{byte('B')}, data; !bytes.Equal(want, got) {
		t.Errorf("offset 1: want %v, got %v", want, got)
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("want error to be nil, got %v", err)
	}
}
