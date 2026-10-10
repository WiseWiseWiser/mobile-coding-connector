package fileupload

import (
	"sync"

	"github.com/gorilla/websocket"
)

// streamWriter is the only data writer for an upload socket.
// The queue holds one unsent message. send returns as soon as that slot is
// taken, so the read loop keeps going while a write is blocked. sendWait is
// for terminal frames (done, error) that the caller must know were written.
type streamWriter struct {
	conn *websocket.Conn
	ch   chan streamWrite
	dead chan struct{}

	mu        sync.Mutex
	err       error
	failOnce  sync.Once
	closeOnce sync.Once
}

type streamWrite struct {
	v    any
	done chan error
}

func newStreamWriter(conn *websocket.Conn) *streamWriter {
	w := &streamWriter{
		conn: conn,
		ch:   make(chan streamWrite, 1),
		dead: make(chan struct{}),
	}
	go w.loop()
	return w
}

func (w *streamWriter) loop() {
	for msg := range w.ch {
		err := writeStreamJSON(w.conn, msg.v)
		if msg.done != nil {
			msg.done <- err
		}
		if err != nil {
			w.fail(err)
			return
		}
	}
}

func (w *streamWriter) fail(err error) {
	w.failOnce.Do(func() {
		w.mu.Lock()
		w.err = err
		w.mu.Unlock()
		close(w.dead)
	})
}

func (w *streamWriter) send(v any) error {
	return w.enqueue(v, nil)
}

func (w *streamWriter) sendWait(v any) error {
	done := make(chan error, 1)
	if err := w.enqueue(v, done); err != nil {
		return err
	}
	select {
	case err := <-done:
		return err
	case <-w.dead:
		return w.writeErr()
	}
}

func (w *streamWriter) enqueue(v any, done chan error) error {
	msg := streamWrite{v: v, done: done}
	select {
	case <-w.dead:
		return w.writeErr()
	case w.ch <- msg:
		return nil
	}
}

func (w *streamWriter) writeErr() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.err != nil {
		return w.err
	}
	return websocket.ErrCloseSent
}

func (w *streamWriter) close() {
	w.closeOnce.Do(func() {
		close(w.ch)
	})
}
