#!/usr/bin/env python3
import argparse
import json
import pathlib
import time
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parents[1]


def load_payload(name: str) -> dict:
    path = ROOT / "payloads" / f"{name}.json"
    with path.open("r", encoding="utf-8") as file:
        return json.load(file)


def with_sequence(payload: dict, index: int) -> dict:
    value = dict(payload)
    suffix = f"{index:06d}"
    value["event_id"] = f"{payload['event_id']}-{suffix}"
    value["correlation_id"] = f"{payload['correlation_id']}-{suffix}"
    return value


def send_rest(url: str, payload: dict) -> int:
    data = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(
        url,
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(request, timeout=10) as response:
        response.read()
        return response.status


def write_file(output: pathlib.Path, payload: dict) -> None:
    output.parent.mkdir(parents=True, exist_ok=True)
    with output.open("a", encoding="utf-8") as file:
        file.write(json.dumps(payload, separators=(",", ":")) + "\n")


def main() -> None:
    parser = argparse.ArgumentParser(description="Gera eventos do case ecommerce-distributed.")
    parser.add_argument("--scenario", default="happy-path", choices=["happy-path", "partial-data", "stop-and-reprocess"])
    parser.add_argument("--count", type=int, default=1)
    parser.add_argument("--target", default="file", choices=["file", "rest"])
    parser.add_argument("--rest-url", default="http://localhost:8088/process")
    parser.add_argument("--output", default=str(ROOT / "results" / "events.ndjson"))
    parser.add_argument("--sleep-ms", type=int, default=0)
    args = parser.parse_args()

    base = load_payload(args.scenario)
    started = time.time()
    for index in range(args.count):
        payload = with_sequence(base, index) if args.count > 1 else dict(base)
        if args.target == "rest":
            status = send_rest(args.rest_url, payload)
            print(f"sent rest status={status} event_id={payload['event_id']}")
        else:
            write_file(pathlib.Path(args.output), payload)
            print(f"wrote event_id={payload['event_id']}")
        if args.sleep_ms > 0:
            time.sleep(args.sleep_ms / 1000)
    elapsed = time.time() - started
    print(f"generated={args.count} target={args.target} elapsed_seconds={elapsed:.3f}")


if __name__ == "__main__":
    main()
