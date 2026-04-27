package bridge

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
)

func workerScriptPath(t *testing.T) string {
	t.Helper()
	return filepath.Join("python_worker.py")
}

func TestInvokeSumNumbers(t *testing.T) {
	pool, err := NewPool(2, workerScriptPath(t))
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()

	res, err := pool.Invoke(context.Background(), "sum_numbers", 1, 2, 3.5)
	if err != nil {
		t.Fatalf("invoke: %v", err)
	}

	num, ok := res.(float64)
	if !ok {
		t.Fatalf("result type = %T, want float64", res)
	}
	if num != 6.5 {
		t.Fatalf("sum = %v, want 6.5", num)
	}
}

func TestConcurrentFib(t *testing.T) {
	pool, err := NewPool(4, workerScriptPath(t))
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()
	const tasks = 20

	var wg sync.WaitGroup
	errs := make(chan error, tasks)
	for i := 0; i < tasks; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := pool.Invoke(ctx, "fib", 30)
			if err != nil {
				errs <- err
				return
			}
			if int(res.(float64)) != 832040 {
				errs <- fmt.Errorf("unexpected fib result: %v", res)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent invoke: %v", err)
		}
	}
}
