package wizserver

import (
	"errors"
	"sync"
)

// MockSocket is a simple Socket implementation that records sent messages. It
// is used heavily in the unit tests to assert broadcast behaviour.
type MockSocket struct {
	mu       sync.Mutex
	Messages []Message
	FailSend bool
}

// Send implements the Socket interface.
func (m *MockSocket) Send(msgs ...Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.FailSend {
		return errors.New("forced send failure")
	}

	m.Messages = append(m.Messages, msgs...)
	return nil
}

// Drain returns a snapshot of the messages captured so far.
func (m *MockSocket) Drain() []Message {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := make([]Message, len(m.Messages))
	copy(out, m.Messages)
	return out
}
