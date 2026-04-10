package runtime

import (
	"context"

	"github.com/docker/docker-agent/pkg/chat"
)

// QueuedMessage is a user message waiting to be injected into the agent loop,
// either mid-turn (via the steer queue) or at end-of-turn (via the follow-up
// queue).
type QueuedMessage struct {
	Content      string
	MultiContent []chat.MessagePart
}

// MessageQueue is the interface for storing messages that are injected into
// the agent loop. Implementations must be safe for concurrent use: Enqueue
// is called from API handlers while Dequeue/Drain are called from the agent
// loop goroutine.
//
// Dequeue uses a Lock + Confirm/Cancel pattern: Dequeue locks the next
// message (making it invisible to subsequent Dequeue calls), Confirm
// permanently removes it after the message has been successfully processed,
// and Cancel releases it back to the queue if processing fails. This
// prevents message loss in persistent queue implementations where the
// session store is also durable.
//
// Note: for the default in-memory queue, Confirm and Cancel are no-ops
// because the message is consumed from the channel on Dequeue and the
// session is also in-memory. The pattern exists so that persistent
// implementations (with a durable session store) can guarantee
// exactly-once delivery.
//
// The default implementation is NewInMemoryMessageQueue. Callers that need
// durable or distributed storage can provide their own implementation
// via the WithSteerQueue or WithFollowUpQueue options.
type MessageQueue interface {
	// Enqueue adds a message to the queue. Returns false if the queue is
	// full or the context is cancelled.
	Enqueue(ctx context.Context, msg QueuedMessage) bool
	// Dequeue locks and returns the next message from the queue. The
	// message is invisible to subsequent Dequeue calls until Confirm or
	// Cancel is called. Returns the message and true, or a zero value
	// and false if the queue is empty. Must not block.
	Dequeue(ctx context.Context) (QueuedMessage, bool)
	// Confirm permanently removes the most recently dequeued message.
	// Must be called after the message has been successfully persisted
	// to the session. For in-memory queues this is a no-op.
	Confirm(ctx context.Context) error
	// Cancel releases the most recently dequeued message back to the
	// queue. For in-memory queues this is a no-op (the message was
	// already consumed from the channel).
	Cancel(ctx context.Context) error
	// Drain locks, returns, and auto-confirms all pending messages.
	// Must not block — if the queue is empty it returns nil.
	Drain(ctx context.Context) []QueuedMessage
	// Len returns the current number of messages in the queue.
	Len(ctx context.Context) int
}

// inMemoryMessageQueue is the default MessageQueue backed by a buffered channel.
type inMemoryMessageQueue struct {
	ch chan QueuedMessage
}

const (
	// defaultSteerQueueCapacity is the buffer size for the default in-memory steer queue.
	defaultSteerQueueCapacity = 5
	// defaultFollowUpQueueCapacity is the buffer size for the default in-memory follow-up queue.
	// Higher than steer because follow-ups accumulate while waiting for the turn to end.
	defaultFollowUpQueueCapacity = 20
)

// NewInMemoryMessageQueue creates a MessageQueue backed by a buffered channel
// with the given capacity.
func NewInMemoryMessageQueue(capacity int) MessageQueue {
	return &inMemoryMessageQueue{ch: make(chan QueuedMessage, capacity)}
}

func (q *inMemoryMessageQueue) Enqueue(_ context.Context, msg QueuedMessage) bool {
	select {
	case q.ch <- msg:
		return true
	default:
		return false
	}
}

func (q *inMemoryMessageQueue) Dequeue(_ context.Context) (QueuedMessage, bool) {
	select {
	case m := <-q.ch:
		return m, true
	default:
		return QueuedMessage{}, false
	}
}

// Confirm is a no-op for in-memory queues — the message was already
// removed from the channel on Dequeue.
func (q *inMemoryMessageQueue) Confirm(_ context.Context) error { return nil }

// Cancel is a no-op for in-memory queues — the message cannot be put
// back into a buffered channel without risking deadlock.
func (q *inMemoryMessageQueue) Cancel(_ context.Context) error { return nil }

func (q *inMemoryMessageQueue) Drain(_ context.Context) []QueuedMessage {
	var msgs []QueuedMessage
	for {
		select {
		case m := <-q.ch:
			msgs = append(msgs, m)
		default:
			return msgs
		}
	}
}

func (q *inMemoryMessageQueue) Len(_ context.Context) int {
	return len(q.ch)
}
