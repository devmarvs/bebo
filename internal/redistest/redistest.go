package redistest

import (
	"bufio"
	"errors"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// Start launches a lightweight Redis-compatible server for tests.
func Start(t *testing.T) (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	server := &Server{
		values:  make(map[string]string),
		buckets: make(map[string]*bucket),
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go server.handle(conn)
		}
	}()

	shutdown := func() {
		_ = listener.Close()
		wg.Wait()
	}

	return listener.Addr().String(), shutdown
}

// Server implements a subset of Redis commands for tests.
type Server struct {
	mu      sync.Mutex
	values  map[string]string
	buckets map[string]*bucket
}

type bucket struct {
	tokens float64
	lastMS int64
}

func (s *Server) handle(conn net.Conn) {
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
				s.mu.Lock()
				s.values[args[1]] = args[2]
				s.mu.Unlock()
			}
			_ = writeSimpleString(writer, "OK")
		case "GET":
			if len(args) < 2 {
				_ = writeBulkString(writer, nil)
				continue
			}
			s.mu.Lock()
			value, ok := s.values[args[1]]
			s.mu.Unlock()
			if !ok {
				_ = writeBulkString(writer, nil)
				continue
			}
			_ = writeBulkString(writer, []byte(value))
		case "DEL":
			removed := 0
			s.mu.Lock()
			for _, key := range args[1:] {
				if _, ok := s.values[key]; ok {
					delete(s.values, key)
					removed++
				}
				delete(s.buckets, key)
			}
			s.mu.Unlock()
			_ = writeInteger(writer, removed)
		case "EVAL":
			allowed, err := s.evalTokenBucket(args)
			if err != nil {
				_ = writeError(writer, err.Error())
				continue
			}
			_ = writeInteger(writer, allowed)
		default:
			_ = writeError(writer, "unknown command")
		}
	}
}

func (s *Server) evalTokenBucket(args []string) (int, error) {
	if len(args) < 7 {
		return 0, errors.New("invalid eval args")
	}
	numKeys, err := strconv.Atoi(args[2])
	if err != nil || numKeys < 1 {
		return 0, errors.New("invalid eval key count")
	}
	if len(args) < 3+numKeys+4 {
		return 0, errors.New("invalid eval args")
	}
	key := args[3]
	rate, err := strconv.ParseFloat(args[3+numKeys], 64)
	if err != nil {
		return 0, err
	}
	burst, err := strconv.ParseFloat(args[4+numKeys], 64)
	if err != nil {
		return 0, err
	}
	nowMS, err := strconv.ParseInt(args[5+numKeys], 10, 64)
	if err != nil {
		return 0, err
	}
	_, _ = strconv.ParseInt(args[6+numKeys], 10, 64)

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.buckets[key]
	if !ok {
		state = &bucket{tokens: burst, lastMS: nowMS}
		s.buckets[key] = state
	}

	delta := nowMS - state.lastMS
	if delta < 0 {
		delta = 0
	}
	state.tokens = math.Min(burst, state.tokens+(float64(delta)*rate/1000))
	state.lastMS = nowMS

	if state.tokens < 1 {
		return 0, nil
	}
	state.tokens -= 1
	return 1, nil
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
