#!/usr/bin/env python3
import json
import sys
from typing import Any, Callable


def sum_numbers(*values: Any) -> float:
    return float(sum(float(v) for v in values))


def word_count(text: str) -> dict[str, int]:
    counts: dict[str, int] = {}
    for word in text.lower().split():
        counts[word] = counts.get(word, 0) + 1
    return counts


def fib(n: int) -> int:
    n = int(n)
    if n < 0:
        raise ValueError("n must be >= 0")
    a, b = 0, 1
    for _ in range(n):
        a, b = b, a + b
    return a


FUNCTIONS: dict[str, Callable[..., Any]] = {
    "sum_numbers": sum_numbers,
    "word_count": word_count,
    "fib": fib,
}


def handle_request(line: str) -> dict[str, Any]:
    payload = json.loads(line)
    req_id = payload.get("id")
    fn_name = payload.get("func")
    args = payload.get("args", [])

    if fn_name not in FUNCTIONS:
        return {"id": req_id, "error": f"unknown function: {fn_name}"}

    try:
        result = FUNCTIONS[fn_name](*args)
        return {"id": req_id, "result": result}
    except Exception as exc:  # noqa: BLE001
        return {"id": req_id, "error": str(exc)}


def main() -> None:
    for raw in sys.stdin:
        line = raw.strip()
        if not line:
            continue

        response = handle_request(line)
        sys.stdout.write(json.dumps(response, ensure_ascii=False) + "\n")
        sys.stdout.flush()


if __name__ == "__main__":
    main()
