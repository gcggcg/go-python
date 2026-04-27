package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"go-python/bridge"
)

func main() {
	script := filepath.Join("bridge", "python_worker.py")
	pool, err := bridge.NewPool(4, script)
	if err != nil {
		log.Fatalf("create pool failed: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	start := time.Now()
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(job int) {
			defer wg.Done()
			res, err := pool.Invoke(ctx, "fib", 38)
			if err != nil {
				log.Printf("job=%d invoke error: %v", job, err)
				return
			}
			fmt.Printf("job=%d fib(38)=%v\n", job, int(res.(float64)))
		}(i)
	}
	wg.Wait()
	fmt.Printf("elapsed=%s\n", time.Since(start))
}
