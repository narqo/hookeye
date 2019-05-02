package stream

import (
	"bytes"
	"context"
	"testing"
)

func TestTopic_Truncate(t *testing.T) {
	ctx := context.Background()

	topic := &Topic{}
	group1 := topic.NewGroup()
	group2 := topic.NewGroup()

	for i := 0; i < 10; i++ {
		err := topic.Push(ctx, []byte{byte('A' + i)})
		if err != nil {
			t.Fatal(err)
		}
	}

	assertTopicPop(t, group1, 0, []byte{'A'})
	assertTopicPop(t, group1, 1, []byte{'B'})
	assertTopicPop(t, group1, 2, []byte{'C'})

	for i := 0; i < 5; i++ {
		assertTopicPop(t, group2, int64(i), []byte{byte('A' + i)})
	}

	topic.Truncate(2) // truncate up to 'C'

	err := topic.Push(ctx, []byte{'x'})
	if err != nil {
		t.Fatal(err)
	}

	assertTopicPop(t, group1, 3, []byte{'D'})

	// read out group1
	for i := 4; i < 10; i++ {
		assertTopicPop(t, group1, int64(i), []byte{byte('A' + i)})
	}

	assertTopicPop(t, group1, 10, []byte{'x'})

	// read out group2
	for i := 5; i < 10; i++ {
		assertTopicPop(t, group2, int64(i), []byte{byte('A' + i)})
	}

	assertTopicPop(t, group2, 10, []byte{'x'})

	err = topic.Push(ctx, []byte{'y'})
	if err != nil {
		t.Fatal(err)
	}

	topic.Truncate(10) // truncate up to 'x'

	err = topic.Push(ctx, []byte{'z'})
	if err != nil {
		t.Fatal(err)
	}

	assertTopicPop(t, group1, 11, []byte{'y'})
	assertTopicPop(t, group2, 11, []byte{'y'})

	assertTopicPop(t, group1, 12, []byte{'z'})
	assertTopicPop(t, group2, 12, []byte{'z'})
}

func assertTopicPop(t *testing.T, group *Group, wantOffset int64, wantData []byte) {
	offset, data, _ := group.Pop(context.Background())
	if offset != wantOffset {
		t.Errorf("group.Pop: want %v, got %v", wantOffset, offset)
	}
	if !bytes.Equal(data, wantData) {
		t.Errorf("group.Pop: data want %v, got %v", wantData, data)
	}
}
