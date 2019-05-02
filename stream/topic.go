package stream

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/xerrors"
)

type Topic struct {
	groupsMu sync.RWMutex
	groups   []*Group

	groupsOffsetsMu sync.Mutex
	groupsOffsets   []int64

	dataMu     sync.RWMutex
	data       [][]byte
	index      map[int64]int
	nextOffset int64
}

func (topic *Topic) NewGroup() *Group {
	topic.groupsMu.Lock()
	defer topic.groupsMu.Unlock()

	group := &Group{
		head:  make(chan int64, 1),
		topic: topic,
	}
	topic.groups = append(topic.groups, group)
	topic.groupsOffsets = append(topic.groupsOffsets, 0)

	group.id = len(topic.groups) - 1

	return group
}

func (topic *Topic) Push(ctx context.Context, data []byte) error {
	topic.dataMu.Lock()
	offset := topic.push(data)
	topic.dataMu.Unlock()

	topic.groupsMu.RLock()
	groups := topic.groups
	topic.groupsMu.RUnlock()

	// simply fan-out for now
	for _, group := range groups {
		group.Push(ctx, offset)
	}

	return nil
}

func (topic *Topic) push(data []byte) (offset int64) {
	offset = topic.nextOffset
	topic.nextOffset++

	if topic.index == nil {
		topic.index = make(map[int64]int)
	}

	topic.index[offset] = len(topic.data)
	topic.data = append(topic.data, data)

	return offset
}

func (topic *Topic) DataAt(offset int64) (data []byte, ok bool) {
	topic.dataMu.RLock()
	defer topic.dataMu.RUnlock()

	pos, ok := topic.index[offset]
	if !ok {
		return nil, ok
	}
	data = topic.data[pos]

	return data, true
}

func (topic *Topic) Offsets() (offsets []int64) {
	topic.groupsOffsetsMu.Lock()
	offsets = append(offsets, topic.groupsOffsets...)
	topic.groupsOffsetsMu.Unlock()
	return offsets
}

func (topic *Topic) CommitOffset(groupID int, offset int64) {
	topic.dataMu.RLock()
	nextOffset := topic.nextOffset
	topic.dataMu.RUnlock()

	topic.groupsOffsetsMu.Lock()
	defer topic.groupsOffsetsMu.Unlock()

	oldOffset := topic.groupsOffsets[groupID]
	if oldOffset > offset && nextOffset > offset {
		return
	}
	topic.groupsOffsets[groupID] = offset
}

func (topic *Topic) Truncate(offset int64) {
	topic.dataMu.Lock()
	defer topic.dataMu.Unlock()

	pos := topic.index[offset] + 1
	topic.data = topic.data[pos:]

	for ixOffset, ixPos := range topic.index {
		if ixOffset <= offset {
			delete(topic.index, ixOffset)
		} else {
			topic.index[ixOffset] = ixPos - pos
		}
	}
}

type Group struct {
	mu   sync.Mutex
	head chan int64
	tail []int64

	// id of the group in the topic
	id    int
	topic *Topic
}

func (group *Group) Pop(ctx context.Context) (offset int64, data []byte, err error) {
	select {
	case offset = <-group.head:
	case <-ctx.Done():
		return 0, nil, ctx.Err()
	}

	group.mu.Lock()
	if len(group.tail) > 0 {
		group.tryPushLocked()
	}
	group.mu.Unlock()

	data, ok := group.topic.DataAt(offset)
	if !ok {
		return 0, nil, xerrors.New("not found")
	}

	group.topic.CommitOffset(group.id, offset)

	return offset, data, nil
}

func (group *Group) Push(ctx context.Context, offset int64) {
	group.mu.Lock()
	defer group.mu.Unlock()

	group.tail = append(group.tail, offset)

	group.tryPushLocked()
}

func (group *Group) tryPushLocked() {
	select {
	case group.head <- group.tail[0]:
		group.tail = group.tail[1:]
	default:
	}
}

func (group *Group) String() string {
	return fmt.Sprintf("<Group:%d>", group.id)
}
