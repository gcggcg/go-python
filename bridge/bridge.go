package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type request struct {
	ID   uint64 `json:"id"`
	Func string `json:"func"`
	Args []any  `json:"args"`
}

type response struct {
	ID     uint64 `json:"id"`
	Result any    `json:"result,omitempty"`
	Error  string `json:"error,omitempty"`
}

type Worker struct {
	cmd    *exec.Cmd
	stdin  *bufio.Writer
	stdout *bufio.Scanner
	mu     sync.Mutex
	nextID atomic.Uint64
}

func NewWorker(scriptPath string) (*Worker, error) {
	if scriptPath == "" {
		return nil, errors.New("script path is required")
	}

	absScript, err := filepath.Abs(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("resolve script path: %w", err)
	}

	cmd := exec.Command("python3", absScript)
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start python worker: %w", err)
	}

	scanner := bufio.NewScanner(stdoutPipe)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	return &Worker{
		cmd:    cmd,
		stdin:  bufio.NewWriter(stdinPipe),
		stdout: scanner,
	}, nil
}

func (w *Worker) Call(ctx context.Context, fn string, args ...any) (any, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	id := w.nextID.Add(1)
	req := request{ID: id, Func: fn, Args: args}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	if _, err := w.stdin.Write(append(payload, '\n')); err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	if err := w.stdin.Flush(); err != nil {
		return nil, fmt.Errorf("flush request: %w", err)
	}

	if !w.stdout.Scan() {
		if err := w.stdout.Err(); err != nil {
			return nil, fmt.Errorf("read response: %w", err)
		}
		return nil, errors.New("python worker closed stdout")
	}

	var resp response
	if err := json.Unmarshal(w.stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if resp.ID != id {
		return nil, fmt.Errorf("response id mismatch: got=%d want=%d", resp.ID, id)
	}
	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}
	return resp.Result, nil
}

func (w *Worker) Close() error {
	if w == nil || w.cmd == nil || w.cmd.Process == nil {
		return nil
	}
	if err := w.cmd.Process.Kill(); err != nil {
		return err
	}
	_, err := w.cmd.Process.Wait()
	return err
}

type Pool struct {
	workers chan *Worker
}

func NewPool(size int, scriptPath string) (*Pool, error) {
	if size <= 0 {
		return nil, errors.New("pool size must be > 0")
	}

	p := &Pool{workers: make(chan *Worker, size)}
	for i := 0; i < size; i++ {
		w, err := NewWorker(scriptPath)
		if err != nil {
			p.Close()
			return nil, err
		}
		p.workers <- w
	}
	return p, nil
}

func (p *Pool) Invoke(ctx context.Context, fn string, args ...any) (any, error) {
	if p == nil {
		return nil, errors.New("pool is nil")
	}

	var w *Worker
	select {
	case w = <-p.workers:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { p.workers <- w }()

	return w.Call(ctx, fn, args...)
}

func (p *Pool) Close() error {
	if p == nil || p.workers == nil {
		return nil
	}

	close(p.workers)
	var firstErr error
	for w := range p.workers {
		if err := w.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
