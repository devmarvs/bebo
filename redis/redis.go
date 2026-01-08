package redis

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// ErrNil indicates a nil Redis response (missing key).
var ErrNil = errors.New("redis: nil")

// Options configures Redis connections.
type Options struct {
	Network      string
	Address      string
	Username     string
	Password     string
	DB           int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// Client executes Redis commands with per-call connections.
type Client struct {
	options Options
}

// New creates a Redis client with defaults.
func New(options Options) *Client {
	cfg := options
	if cfg.Network == "" {
		cfg.Network = "tcp"
	}
	if cfg.Address == "" {
		cfg.Address = "127.0.0.1:6379"
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 2 * time.Second
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 2 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 2 * time.Second
	}
	return &Client{options: cfg}
}

// Do executes a Redis command.
func (c *Client) Do(args ...string) (any, error) {
	return c.DoContext(context.Background(), args...)
}

// DoContext executes a Redis command with context.
func (c *Client) DoContext(ctx context.Context, args ...string) (any, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	conn, err := c.dial(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.close()

	resp, err := conn.do(ctx, args...)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, ErrNil
	}
	return resp, nil
}

func (c *Client) dial(ctx context.Context) (*redisConn, error) {
	dialer := &net.Dialer{Timeout: c.options.DialTimeout}
	conn, err := dialer.DialContext(ctx, c.options.Network, c.options.Address)
	if err != nil {
		return nil, err
	}

	client := &redisConn{
		conn:         conn,
		rw:           bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		readTimeout:  c.options.ReadTimeout,
		writeTimeout: c.options.WriteTimeout,
	}

	if c.options.Password != "" {
		args := []string{"AUTH"}
		if c.options.Username != "" {
			args = append(args, c.options.Username, c.options.Password)
		} else {
			args = append(args, c.options.Password)
		}
		if _, err := client.do(ctx, args...); err != nil {
			client.close()
			return nil, err
		}
	}
	if c.options.DB > 0 {
		if _, err := client.do(ctx, "SELECT", strconv.Itoa(c.options.DB)); err != nil {
			client.close()
			return nil, err
		}
	}

	return client, nil
}

type redisConn struct {
	conn         net.Conn
	rw           *bufio.ReadWriter
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func (c *redisConn) do(ctx context.Context, args ...string) (any, error) {
	if err := c.writeCommand(ctx, args); err != nil {
		return nil, err
	}
	return c.readResponse(ctx)
}

func (c *redisConn) close() {
	_ = c.conn.Close()
}

func (c *redisConn) writeCommand(ctx context.Context, args []string) error {
	if err := c.applyDeadline(ctx, c.writeTimeout, true); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(c.rw, "*%d\r\n", len(args)); err != nil {
		return err
	}
	for _, arg := range args {
		if _, err := fmt.Fprintf(c.rw, "$%d\r\n%s\r\n", len(arg), arg); err != nil {
			return err
		}
	}
	return c.rw.Flush()
}

func (c *redisConn) readResponse(ctx context.Context) (any, error) {
	if err := c.applyDeadline(ctx, c.readTimeout, false); err != nil {
		return nil, err
	}

	line, err := c.rw.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSuffix(line, "\r\n")
	if line == "" {
		return nil, errors.New("redis: empty response")
	}

	switch line[0] {
	case '+':
		return line[1:], nil
	case '-':
		return nil, errors.New(line[1:])
	case ':':
		value, err := strconv.ParseInt(line[1:], 10, 64)
		if err != nil {
			return nil, err
		}
		return value, nil
	case '$':
		length, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		if length == -1 {
			return nil, nil
		}
		buf := make([]byte, length+2)
		if _, err := io.ReadFull(c.rw, buf); err != nil {
			return nil, err
		}
		return buf[:length], nil
	case '*':
		count, err := strconv.Atoi(line[1:])
		if err != nil {
			return nil, err
		}
		items := make([]any, 0, count)
		for i := 0; i < count; i++ {
			item, err := c.readResponse(ctx)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
		return items, nil
	default:
		return nil, errors.New("redis: unknown response")
	}
}

func (c *redisConn) applyDeadline(ctx context.Context, timeout time.Duration, write bool) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	deadline := deadlineFor(ctx, timeout)
	if deadline.IsZero() {
		return nil
	}
	if write {
		return c.conn.SetWriteDeadline(deadline)
	}
	return c.conn.SetReadDeadline(deadline)
}

func deadlineFor(ctx context.Context, timeout time.Duration) time.Time {
	var deadline time.Time
	if timeout > 0 {
		deadline = time.Now().Add(timeout)
	}
	if ctx == nil {
		return deadline
	}
	ctxDeadline, ok := ctx.Deadline()
	if !ok {
		return deadline
	}
	if deadline.IsZero() || ctxDeadline.Before(deadline) {
		return ctxDeadline
	}
	return deadline
}
