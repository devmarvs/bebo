package realtime

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/devmarvs/bebo"
)

const (
	OpContinuation = 0x0
	OpText         = 0x1
	OpBinary       = 0x2
	OpClose        = 0x8
	OpPing         = 0x9
	OpPong         = 0xA
)

var ErrClosed = errors.New("websocket closed")

const wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// WebSocketOptions configures websocket behavior.
type WebSocketOptions struct {
	MaxMessageSize int64
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
}

// Conn represents a websocket connection.
type Conn struct {
	conn           net.Conn
	rw             *bufio.ReadWriter
	maxMessageSize int64
	readTimeout    time.Duration
	writeTimeout   time.Duration
}

// Upgrade upgrades the request to a WebSocket connection.
func Upgrade(ctx *bebo.Context, options WebSocketOptions) (*Conn, error) {
	req := ctx.Request
	if req.Method != http.MethodGet {
		return nil, errors.New("websocket upgrade requires GET")
	}

	if !headerContains(req.Header, "Connection", "upgrade") || !headerContains(req.Header, "Upgrade", "websocket") {
		return nil, errors.New("websocket upgrade headers missing")
	}
	if req.Header.Get("Sec-WebSocket-Version") != "13" {
		return nil, errors.New("unsupported websocket version")
	}
	key := strings.TrimSpace(req.Header.Get("Sec-WebSocket-Key"))
	if key == "" {
		return nil, errors.New("websocket key missing")
	}

	hj, ok := ctx.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, errors.New("response writer does not support hijacking")
	}

	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}

	accept := acceptKey(key)
	response := fmt.Sprintf("HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", accept)
	if _, err := rw.WriteString(response); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}

	return newConn(conn, rw, options), nil
}

func newConn(conn net.Conn, rw *bufio.ReadWriter, options WebSocketOptions) *Conn {
	maxSize := options.MaxMessageSize
	if maxSize <= 0 {
		maxSize = 1 << 20
	}
	return &Conn{
		conn:           conn,
		rw:             rw,
		maxMessageSize: maxSize,
		readTimeout:    options.ReadTimeout,
		writeTimeout:   options.WriteTimeout,
	}
}

// ReadMessage reads the next non-control frame.
func (c *Conn) ReadMessage() (int, []byte, error) {
	for {
		opcode, payload, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}
		switch opcode {
		case OpPing:
			_ = c.WriteMessage(OpPong, payload)
			continue
		case OpPong:
			continue
		case OpClose:
			_ = c.WriteMessage(OpClose, nil)
			return OpClose, nil, ErrClosed
		default:
			return opcode, payload, nil
		}
	}
}

// ReadText reads the next text frame as a string.
func (c *Conn) ReadText() (string, error) {
	opcode, payload, err := c.ReadMessage()
	if err != nil {
		return "", err
	}
	if opcode != OpText {
		return "", errors.New("unexpected opcode")
	}
	return string(payload), nil
}

// WriteMessage writes a websocket frame.
func (c *Conn) WriteMessage(opcode int, payload []byte) error {
	if c.writeTimeout > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
	}

	fin := byte(0x80)
	b1 := fin | byte(opcode)
	if err := c.rw.WriteByte(b1); err != nil {
		return err
	}

	length := len(payload)
	if opcode >= OpClose && length > 125 {
		return errors.New("control frame too large")
	}
	switch {
	case length <= 125:
		if err := c.rw.WriteByte(byte(length)); err != nil {
			return err
		}
	case length <= 0xFFFF:
		if err := c.rw.WriteByte(126); err != nil {
			return err
		}
		if err := writeUint16(c.rw, uint16(length)); err != nil {
			return err
		}
	default:
		if err := c.rw.WriteByte(127); err != nil {
			return err
		}
		if err := writeUint64(c.rw, uint64(length)); err != nil {
			return err
		}
	}

	if length > 0 {
		if _, err := c.rw.Write(payload); err != nil {
			return err
		}
	}
	return c.rw.Flush()
}

// WriteText writes a text frame.
func (c *Conn) WriteText(message string) error {
	return c.WriteMessage(OpText, []byte(message))
}

// Close closes the websocket connection.
func (c *Conn) Close() error {
	_ = c.WriteMessage(OpClose, nil)
	return c.conn.Close()
}

func (c *Conn) readFrame() (int, []byte, error) {
	if c.readTimeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
	}

	b1, err := c.rw.ReadByte()
	if err != nil {
		return 0, nil, err
	}
	b2, err := c.rw.ReadByte()
	if err != nil {
		return 0, nil, err
	}

	fin := b1&0x80 != 0
	opcode := int(b1 & 0x0F)
	masked := b2&0x80 != 0
	payloadLen := int64(b2 & 0x7F)

	if !fin && opcode != OpContinuation {
		return 0, nil, errors.New("fragmented frames not supported")
	}
	if !masked {
		return 0, nil, errors.New("client frames must be masked")
	}

	switch payloadLen {
	case 126:
		value, err := readUint16(c.rw)
		if err != nil {
			return 0, nil, err
		}
		payloadLen = int64(value)
	case 127:
		value, err := readUint64(c.rw)
		if err != nil {
			return 0, nil, err
		}
		payloadLen = int64(value)
	}

	if payloadLen > c.maxMessageSize {
		return 0, nil, errors.New("message too large")
	}
	if opcode >= OpClose && payloadLen > 125 {
		return 0, nil, errors.New("control frame too large")
	}

	maskKey := make([]byte, 4)
	if _, err := io.ReadFull(c.rw, maskKey); err != nil {
		return 0, nil, err
	}

	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(c.rw, payload); err != nil {
			return 0, nil, err
		}
	}
	for i := int64(0); i < payloadLen; i++ {
		payload[i] ^= maskKey[i%4]
	}

	return opcode, payload, nil
}

func acceptKey(key string) string {
	h := sha1.New()
	_, _ = h.Write([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func headerContains(h http.Header, key, value string) bool {
	for _, part := range h.Values(key) {
		for _, token := range strings.Split(part, ",") {
			if strings.EqualFold(strings.TrimSpace(token), value) {
				return true
			}
		}
	}
	return false
}

func readUint16(r *bufio.ReadWriter) (uint16, error) {
	buf := make([]byte, 2)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	return uint16(buf[0])<<8 | uint16(buf[1]), nil
}

func writeUint16(w *bufio.ReadWriter, value uint16) error {
	buf := []byte{byte(value >> 8), byte(value)}
	_, err := w.Write(buf)
	return err
}

func readUint64(r *bufio.ReadWriter) (uint64, error) {
	buf := make([]byte, 8)
	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}
	var value uint64
	for _, b := range buf {
		value = (value << 8) | uint64(b)
	}
	return value, nil
}

func writeUint64(w *bufio.ReadWriter, value uint64) error {
	buf := make([]byte, 8)
	for i := 7; i >= 0; i-- {
		buf[i] = byte(value & 0xFF)
		value >>= 8
	}
	_, err := w.Write(buf)
	return err
}
