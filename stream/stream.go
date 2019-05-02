package stream

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"
)

type Processor interface {
	Process(ctx context.Context, msg *Message) error
}

type ProcessorFunc func(ctx context.Context, msg *Message) error

func (p ProcessorFunc) Process(ctx context.Context, msg *Message) error {
	return p(ctx, msg)
}

type Message struct {
	Offset int64
	Data   []byte

	group *Group
}

type Stream struct {
	mu     sync.RWMutex
	topics map[string]*Topic

	compactInterval time.Duration

	wg   sync.WaitGroup
	done chan struct{}
}

func New() *Stream {
	return &Stream{
		topics: make(map[string]*Topic),
		done:   make(chan struct{}),
	}
}

func (stream *Stream) SubscribeN(key string, p Processor, n int) {
	stream.mu.Lock()
	defer stream.mu.Unlock()

	topic, ok := stream.topics[key]
	if !ok {
		topic = &Topic{}
		stream.topics[key] = topic
	}

	group := topic.NewGroup()

	stream.wg.Add(n)
	for ; n > 0; n-- {
		go func() {
			readGroup(group, p, stream.done)
			stream.wg.Done()
		}()
	}
}

func (stream *Stream) Stop() error {
	close(stream.done)
	stream.wg.Wait()
	return nil
}

func readGroup(group *Group, p Processor, done <-chan struct{}) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		<-done
		cancel()
	}()

	for {
		offset, data, err := group.Pop(ctx)
		if err == nil {
			msg := &Message{
				Offset: offset,
				Data:   data,
				group:  group,
			}
			err = p.Process(ctx, msg)
		}

		select {
		case <-ctx.Done():
			return
		default:
		}

		if err != nil {
			log.Printf("stream: failed to process message from group %s: %v", group, err)
		}
	}
}

func (stream *Stream) Push(ctx context.Context, key string, data []byte) error {
	stream.mu.RLock()
	topic, ok := stream.topics[key]
	stream.mu.RUnlock()

	if !ok {
		stream.mu.Lock()
		topic, ok = stream.topics[key]
		if !ok {
			topic = &Topic{}
			stream.topics[key] = topic
		}
		stream.mu.Unlock()
	}

	return topic.Push(ctx, data)
}

// test only
func (stream *Stream) Topic(key string) *Topic {
	stream.mu.RLock()
	topic := stream.topics[key]
	stream.mu.RUnlock()
	return topic
}

func (stream *Stream) Compact() {
	stream.mu.RLock()
	for key, topic := range stream.topics {
		compactTopic(key, topic)
	}
	stream.mu.RUnlock()
}

func compactTopic(key string, topic *Topic) {
	offsets := topic.Offsets()

	sort.Slice(offsets, func(i, j int) bool {
		return offsets[i] < offsets[j]
	})

	//log.Printf("stream(debug): truncate topic %q to offset %d: %v\n", key, offsets[0], topic)

	topic.Truncate(offsets[0])
}
