package session

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestRedisStoreRoundTrip(t *testing.T) {
	addr, shutdown := startRedisServer(t)
	defer shutdown()

	store := NewRedisStore(RedisOptions{
		Address:      addr,
		DialTimeout:  500 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		TTL:          time.Minute,
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	sess, err := store.Get(req)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	sess.Set("user_id", "123")

	rec := httptest.NewRecorder()
	if err := store.Save(rec, sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	resp := rec.Result()
	defer resp.Body.Close()

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, cookie := range resp.Cookies() {
		req2.AddCookie(cookie)
	}

	loaded, err := store.Get(req2)
	if err != nil {
		t.Fatalf("get loaded: %v", err)
	}
	if loaded.Get("user_id") != "123" {
		t.Fatalf("expected user_id 123, got %q", loaded.Get("user_id"))
	}
}

func startRedisServer(t *testing.T) (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	store := &fakeRedis{values: make(map[string]string)}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go store.handle(conn)
		}
	}()

	shutdown := func() {
		_ = listener.Close()
		wg.Wait()
	}

	return listener.Addr().String(), shutdown
}

type fakeRedis struct {
	mu     sync.Mutex
	values map[string]string
}

func (f *fakeRedis) handle(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	for {
		args, err := readRESPArray(reader)
		if err != nil {
			return
		}
		if len(args) == 0 {
			_ = writeSimpleString(writer, "OK")
			continue
		}
		cmd := strings.ToUpper(args[0])
		switch cmd {
		case "AUTH":
			_ = writeSimpleString(writer, "OK")
		case "SELECT":
			_ = writeSimpleString(writer, "OK")
		case "SET":
			if len(args) >= 3 {
				f.mu.Lock()
				f.values[args[1]] = args[2]
				f.mu.Unlock()
			}
			_ = writeSimpleString(writer, "OK")
		case "GET":
			if len(args) < 2 {
				_ = writeBulkString(writer, nil)
				continue
			}
			f.mu.Lock()
			value, ok := f.values[args[1]]
			f.mu.Unlock()
			if !ok {
				_ = writeBulkString(writer, nil)
				continue
			}
			_ = writeBulkString(writer, []byte(value))
		case "DEL":
			removed := 0
			f.mu.Lock()
			for _, key := range args[1:] {
				if _, ok := f.values[key]; ok {
					delete(f.values, key)
					removed++
				}
			}
			f.mu.Unlock()
			_ = writeInteger(writer, removed)
		default:
			_ = writeError(writer, "unknown command")
		}
	}
}

func readRESPArray(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\r\n")
	if !strings.HasPrefix(line, "*") {
		return nil, errors.New("expected array")
	}

	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		line, err = reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		line = strings.TrimSuffix(line, "\r\n")
		if !strings.HasPrefix(line, "$") {
			return nil, errors.New("expected bulk string")
		}
		length, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(reader, buf); err != nil {
			return nil, err
		}
		args = append(args, string(buf[:length]))
	}
	return args, nil
}

func writeSimpleString(writer *bufio.Writer, value string) error {
	if _, err := writer.WriteString("+" + value + "\r\n"); err != nil {
		return err
	}
	return writer.Flush()
}

func writeError(writer *bufio.Writer, value string) error {
	if _, err := writer.WriteString("-" + value + "\r\n"); err != nil {
		return err
	}
	return writer.Flush()
}

func writeBulkString(writer *bufio.Writer, payload []byte) error {
	if payload == nil {
		if _, err := writer.WriteString("$-1\r\n"); err != nil {
			return err
		}
		return writer.Flush()
	}
	if _, err := writer.WriteString("$" + strconv.Itoa(len(payload)) + "\r\n"); err != nil {
		return err
	}
	if _, err := writer.Write(payload); err != nil {
		return err
	}
	if _, err := writer.WriteString("\r\n"); err != nil {
		return err
	}
	return writer.Flush()
}

func writeInteger(writer *bufio.Writer, value int) error {
	if _, err := writer.WriteString(":" + strconv.Itoa(value) + "\r\n"); err != nil {
		return err
	}
	return writer.Flush()
}
