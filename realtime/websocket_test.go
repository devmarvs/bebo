package realtime

import (
	"bufio"
	"io"
	"net"
	"testing"
)

func TestAcceptKey(t *testing.T) {
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	expected := "s3pPLMBiTxaQ9kYGzzhZRbK+xOo="
	if got := acceptKey(key); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestConnReadWrite(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	conn := newConn(server, bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server)), WebSocketOptions{})

	go func() {
		_ = writeMaskedText(client, []byte("hello"))
	}()

	opcode, payload, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if opcode != OpText {
		t.Fatalf("expected opcode %d, got %d", OpText, opcode)
	}
	if string(payload) != "hello" {
		t.Fatalf("expected payload %q, got %q", "hello", string(payload))
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- conn.WriteText("pong")
	}()

	frame := make([]byte, 2+4)
	if _, err := io.ReadFull(client, frame[:2]); err != nil {
		t.Fatalf("read frame header: %v", err)
	}
	if frame[0]&0x0F != OpText {
		t.Fatalf("expected text opcode")
	}
	length := int(frame[1] & 0x7F)
	payloadBuf := make([]byte, length)
	if _, err := io.ReadFull(client, payloadBuf); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if string(payloadBuf) != "pong" {
		t.Fatalf("expected payload %q, got %q", "pong", string(payloadBuf))
	}

	err = <-errCh
	if err != nil {
		t.Fatalf("write text: %v", err)
	}
}

func writeMaskedText(conn net.Conn, payload []byte) error {
	maskKey := []byte{1, 2, 3, 4}
	frame := []byte{0x81, 0x80 | byte(len(payload)), maskKey[0], maskKey[1], maskKey[2], maskKey[3]}
	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ maskKey[i%4]
	}
	frame = append(frame, masked...)
	_, err := conn.Write(frame)
	return err
}
