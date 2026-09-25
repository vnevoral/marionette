package execengine

import (
	"io"
	"sync"
)

const outputLimit = 4 * 1024

type limitedWriter struct {
	mu        sync.Mutex
	data      []byte
	truncated bool
}

func newLimitedWriter(limit int) *limitedWriter {
	return &limitedWriter{data: make([]byte, 0, limit)}
}

func (writer *limitedWriter) Write(data []byte) (int, error) {
	writer.mu.Lock()
	defer writer.mu.Unlock()

	remaining := cap(writer.data) - len(writer.data)
	if len(data) > remaining {
		writer.data = append(writer.data, data[:remaining]...)
		writer.truncated = true
		return len(data), nil
	}
	writer.data = append(writer.data, data...)
	return len(data), nil
}

func (writer *limitedWriter) String() string {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return string(append([]byte(nil), writer.data...))
}

func (writer *limitedWriter) Truncated() bool {
	writer.mu.Lock()
	defer writer.mu.Unlock()
	return writer.truncated
}

var _ io.Writer = (*limitedWriter)(nil)
