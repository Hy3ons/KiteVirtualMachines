package apis

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestBridgeConsoleWritesBrowserInputToUpstreamAsBinary(t *testing.T) {
	browser := newScriptedConsoleSocket([]consoleFrame{
		{messageType: websocket.TextMessage, data: []byte("a")},
		{messageType: websocket.TextMessage, data: []byte("\r")},
	}, false)
	upstream := newScriptedConsoleSocket(nil, true)

	err := bridgeConsole(context.Background(), browser, upstream)

	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected bridge to stop on EOF, got %v", err)
	}
	writes := upstream.writes()
	if len(writes) != 2 {
		t.Fatalf("expected 2 upstream writes, got %d", len(writes))
	}
	for i, write := range writes {
		if write.messageType != websocket.BinaryMessage {
			t.Fatalf("write %d used message type %d, want %d", i, write.messageType, websocket.BinaryMessage)
		}
	}
	if string(writes[0].data) != "a" || string(writes[1].data) != "\r" {
		t.Fatalf("unexpected upstream payloads: %q %q", string(writes[0].data), string(writes[1].data))
	}
}

func TestBridgeConsolePreservesUpstreamOutputMessageType(t *testing.T) {
	browser := newScriptedConsoleSocket(nil, true)
	upstream := newScriptedConsoleSocket([]consoleFrame{
		{messageType: websocket.BinaryMessage, data: []byte("boot")},
	}, false)

	err := bridgeConsole(context.Background(), browser, upstream)

	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected bridge to stop on EOF, got %v", err)
	}
	writes := browser.writes()
	if len(writes) != 1 {
		t.Fatalf("expected 1 browser write, got %d", len(writes))
	}
	if writes[0].messageType != websocket.BinaryMessage {
		t.Fatalf("browser write used message type %d, want %d", writes[0].messageType, websocket.BinaryMessage)
	}
	if string(writes[0].data) != "boot" {
		t.Fatalf("unexpected browser payload: %q", string(writes[0].data))
	}
}

func TestBridgeConsoleClosesSocketsWhenContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	browser := newScriptedConsoleSocket(nil, true)
	upstream := newScriptedConsoleSocket(nil, true)
	errs := make(chan error, 1)

	go func() {
		errs <- bridgeConsole(ctx, browser, upstream)
	}()
	cancel()

	select {
	case err := <-errs:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected bridge to stop on context cancellation, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected bridge to stop after context cancellation")
	}
}

type consoleFrame struct {
	messageType int
	data        []byte
}

type scriptedConsoleSocket struct {
	mu         sync.Mutex
	readIndex  int
	readFrames []consoleFrame
	writeLog   []consoleFrame
	blockRead  bool
	closed     chan struct{}
	closeOnce  sync.Once
}

func newScriptedConsoleSocket(readFrames []consoleFrame, blockRead bool) *scriptedConsoleSocket {
	return &scriptedConsoleSocket{
		readFrames: readFrames,
		blockRead:  blockRead,
		closed:     make(chan struct{}),
	}
}

func (s *scriptedConsoleSocket) ReadMessage() (int, []byte, error) {
	s.mu.Lock()
	if s.readIndex < len(s.readFrames) {
		frame := s.readFrames[s.readIndex]
		s.readIndex++
		s.mu.Unlock()
		return frame.messageType, frame.data, nil
	}
	blockRead := s.blockRead
	s.mu.Unlock()

	if blockRead {
		<-s.closed
	}
	return 0, nil, io.EOF
}

func (s *scriptedConsoleSocket) WriteMessage(messageType int, data []byte) error {
	copied := append([]byte(nil), data...)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.writeLog = append(s.writeLog, consoleFrame{messageType: messageType, data: copied})
	return nil
}

func (s *scriptedConsoleSocket) Close() error {
	s.closeOnce.Do(func() {
		close(s.closed)
	})
	return nil
}

func (s *scriptedConsoleSocket) writes() []consoleFrame {
	s.mu.Lock()
	defer s.mu.Unlock()

	writes := make([]consoleFrame, len(s.writeLog))
	copy(writes, s.writeLog)
	return writes
}
