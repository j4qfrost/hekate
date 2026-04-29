package firehose

import "context"

// Mock is an in-memory firehose source that emits a fixed slice of events
// then closes. Used by tests and by `hekate-server --demo` to exercise the
// indexer without a real relay connection.
type Mock struct {
	Events []Event
}

func (m *Mock) Run(ctx context.Context, sink chan<- Event) error {
	for _, ev := range m.Events {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sink <- ev:
		}
	}
	return nil
}
