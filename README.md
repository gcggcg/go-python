# Go + Python 高并发融合示例

这个仓库演示如何让 Go 无缝调用 Python 函数，并通过 Go 的 goroutine + worker pool 把 Python 计算任务并发化。

## 设计思路

- **Python 侧**：`bridge/python_worker.py` 提供函数注册表，使用 JSON 行协议通过 stdin/stdout 接收调用请求。
- **Go 侧**：`bridge/bridge.go` 启动多个常驻 Python worker，构建连接池。
- **并发模型**：每个 Python worker 串行处理自身请求；Go 通过多个 worker 并发执行任务，实现吞吐提升。

## 快速运行

```bash
go test ./...
go run ./cmd/demo
```

> 需要本地存在 `python3`。

## 目前内置的 Python 函数

- `sum_numbers(*values)`：数值求和
- `word_count(text)`：词频统计
- `fib(n)`：斐波那契（迭代）

## 对你的项目如何落地

1. 把 Python 核心逻辑拆成可调用函数，注册到 `FUNCTIONS`。
2. Go 侧根据 CPU 和任务类型设置 worker 数量（例如 `runtime.NumCPU()` 附近）。
3. 在业务层将热点路径迁移到 `pool.Invoke(ctx, fn, args...)`。
4. 对大型 payload 可升级为 Unix socket / gRPC / shared memory 等更高效协议。

