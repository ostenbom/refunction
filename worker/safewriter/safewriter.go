package safewriter

import (
	"io"
	"sync"
)

type SafeWriter struct {
	writer io.Writer
	mux    sync.Mutex
}

func NewSafeWriter(writer io.Writer) *SafeWriter {
	return &SafeWriter{
		writer: writer,
	}
}

func (s *SafeWriter) Write(reader io.Reader) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	_, err := io.Copy(s.writer, reader)
	if err != nil {
		return err
	}
	return nil
}
