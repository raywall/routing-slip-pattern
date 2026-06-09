#!/usr/bin/env python3
from __future__ import annotations

import argparse
import concurrent.futures
import json
import pathlib
import statistics
import subprocess
import sys
import time
import urllib.error
import urllib.request
import uuid
from datetime import datetime, timezone


ROOT = pathlib.Path(__file__).resolve().parents[1]
WORKSPACE = ROOT.parents[2]
RESULTS = ROOT / "results"


def now_iso() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def load_payload(name: str = "happy-path") -> dict:
    with (ROOT / "payloads" / f"{name}.json").open("r", encoding="utf-8") as file:
        return json.load(file)


def event_payload(scenario: str = "happy-path") -> dict:
    payload = dict(load_payload(scenario))
    payload["event_id"] = f"{payload['event_id']}-{uuid.uuid4().hex[:12]}"
    payload["correlation_id"] = str(uuid.uuid4())
    return payload


def http_json(method: str, url: str, payload: dict | None = None, headers: dict | None = None, timeout: int = 20) -> dict:
    data = None if payload is None else json.dumps(payload).encode("utf-8")
    request_headers = {"Content-Type": "application/json"}
    request_headers.update(headers or {})
    request = urllib.request.Request(url, data=data, headers=request_headers, method=method)
    started = time.perf_counter()
    try:
        with urllib.request.urlopen(request, timeout=timeout) as response:
            raw = response.read().decode("utf-8")
            body = json.loads(raw) if raw else None
            return {
                "ok": 200 <= response.status < 300,
                "status": response.status,
                "duration_ms": round((time.perf_counter() - started) * 1000, 2),
                "body": body,
            }
    except urllib.error.HTTPError as err:
        raw = err.read().decode("utf-8")
        try:
            body = json.loads(raw) if raw else None
        except json.JSONDecodeError:
            body = raw
        return {
            "ok": False,
            "status": err.code,
            "duration_ms": round((time.perf_counter() - started) * 1000, 2),
            "body": body,
        }
    except Exception as err:
        return {
            "ok": False,
            "status": 0,
            "duration_ms": round((time.perf_counter() - started) * 1000, 2),
            "error": str(err),
        }


def graphql_query(graphql_url: str) -> dict:
    query = """
    query ($orderID: String!, $customerID: String!, $region: String!) {
      dataSources(orderID: $orderID, customerID: $customerID, region: $region) {
        order { id status total items { sku quantity } }
        customer { id tier }
        inventory { sku available warehouse }
        deliveryPolicy { region promise_days carriers { id } }
      }
    }
    """
    return http_json("POST", f"{graphql_url}/graphql", {
        "query": query,
        "variables": {"orderID": "ORD-1001", "customerID": "CUS-1001", "region": "SP"},
    })


def workflow_call(workflow_url: str, payload: dict) -> dict:
    result = http_json("POST", f"{workflow_url}/process", payload, timeout=60)
    body = result.get("body") or {}
    result["message_id"] = body.get("message_id") or payload.get("event_id")
    result["correlation_id"] = body.get("correlation_id") or payload.get("correlation_id")
    result["workflow_error"] = body.get("error")
    result["workflow"] = body.get("workflow")
    return result


def mcp_request(mcp_url: str, method: str, params: dict | None = None, request_id: int = 1) -> dict:
    payload = {"jsonrpc": "2.0", "id": request_id, "method": method}
    if params is not None:
        payload["params"] = params
    return http_json("POST", f"{mcp_url}/mcp", payload, timeout=20)


def mcp_tool(mcp_url: str, name: str, arguments: dict | None = None, request_id: int = 1) -> dict:
    return mcp_request(mcp_url, "tools/call", {
        "name": name,
        "arguments": arguments or {},
    }, request_id=request_id)


def percentile(values: list[float], pct: float) -> float:
    if not values:
        return 0
    ordered = sorted(values)
    index = min(len(ordered) - 1, max(0, int(round((pct / 100) * (len(ordered) - 1)))))
    return round(ordered[index], 2)


def summarize_calls(calls: list[dict]) -> dict:
    durations = [item["duration_ms"] for item in calls]
    completed = [item for item in calls if item.get("status") == 200 and not item.get("workflow_error")]
    failed = [item for item in calls if item.get("status") != 200 or item.get("workflow_error")]
    return {
        "total": len(calls),
        "completed": len(completed),
        "failed": len(failed),
        "average_ms": round(statistics.mean(durations), 2) if durations else 0,
        "p95_ms": percentile(durations, 95),
        "p99_ms": percentile(durations, 99),
        "min_ms": round(min(durations), 2) if durations else 0,
        "max_ms": round(max(durations), 2) if durations else 0,
    }


def run_regression(args) -> dict:
    checks = []
    gql = graphql_query(args.graphql_url)
    checks.append({
        "name": "graphql_context",
        "passed": gql["ok"] and not (gql.get("body") or {}).get("errors"),
        "status": gql["status"],
        "duration_ms": gql["duration_ms"],
    })

    payload = event_payload()
    workflow = workflow_call(args.workflow_url, payload)
    checks.append({
        "name": "workflow_happy_path",
        "passed": workflow["status"] == 200 and not workflow.get("workflow_error"),
        "status": workflow["status"],
        "duration_ms": workflow["duration_ms"],
        "message_id": workflow["message_id"],
        "correlation_id": workflow["correlation_id"],
        "error": workflow.get("workflow_error"),
    })

    metrics = http_json("GET", f"{args.metrics_url}/v1/metrics")
    checks.append({
        "name": "metrics_available",
        "passed": metrics["ok"] and isinstance(metrics.get("body"), list),
        "status": metrics["status"],
        "duration_ms": metrics["duration_ms"],
    })

    mcp_health = http_json("GET", f"{args.mcp_url}/health")
    checks.append({
        "name": "mcp_health",
        "passed": mcp_health["ok"],
        "status": mcp_health["status"],
        "duration_ms": mcp_health["duration_ms"],
    })

    return {"suite": "regression", "passed": all(item["passed"] for item in checks), "checks": checks}


def run_performance(args) -> dict:
    calls: list[dict] = []

    def send_one(_: int) -> dict:
        return workflow_call(args.workflow_url, event_payload())

    started = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=args.concurrency) as executor:
        for item in executor.map(send_one, range(args.count)):
            calls.append(item)
    elapsed = round((time.perf_counter() - started) * 1000, 2)
    summary = summarize_calls(calls)
    summary["elapsed_ms"] = elapsed
    summary["throughput_per_second"] = round((len(calls) / elapsed) * 1000, 2) if elapsed else 0
    summary["concurrency"] = args.concurrency
    summary["sample_failures"] = [
        {
            "status": item.get("status"),
            "message_id": item.get("message_id"),
            "error": item.get("workflow_error") or item.get("error"),
        }
        for item in calls
        if item.get("status") != 200 or item.get("workflow_error")
    ][:5]
    return {"suite": "performance", "passed": summary["failed"] == 0, "summary": summary}


def docker_compose(args, *command: str) -> dict:
    started = time.perf_counter()
    completed = subprocess.run(
        ["docker", "compose", *command],
        cwd=WORKSPACE,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
    )
    return {
        "command": "docker compose " + " ".join(command),
        "status": completed.returncode,
        "duration_ms": round((time.perf_counter() - started) * 1000, 2),
        "output": completed.stdout[-4000:],
    }


def run_chaos(args) -> dict:
    checks = []

    invalid = event_payload()
    invalid.pop("order_id", None)
    invalid_call = workflow_call(args.workflow_url, invalid)
    checks.append({
        "name": "invalid_payload_stops_with_snapshot",
        "passed": invalid_call["status"] == 202 and bool(invalid_call.get("workflow_error")),
        "status": invalid_call["status"],
        "message_id": invalid_call["message_id"],
        "error": invalid_call.get("workflow_error"),
    })

    payload = event_payload()
    stop = docker_compose(args, "stop", "go-graphql-connector")
    failed = workflow_call(args.workflow_url, payload)
    start = docker_compose(args, "up", "-d", "go-graphql-connector")
    time.sleep(args.recovery_wait_seconds)
    recovered = workflow_call(args.workflow_url, payload)

    checks.append({
        "name": "graphql_outage_creates_reprocessable_failure",
        "passed": failed["status"] == 202 and bool(failed.get("workflow_error")),
        "status": failed["status"],
        "message_id": failed["message_id"],
        "error": failed.get("workflow_error"),
        "stop": stop["status"],
    })
    checks.append({
        "name": "same_event_reprocesses_after_graphql_recovery",
        "passed": recovered["status"] == 200 and not recovered.get("workflow_error"),
        "status": recovered["status"],
        "message_id": recovered["message_id"],
        "error": recovered.get("workflow_error"),
        "start": start["status"],
    })

    return {"suite": "chaos", "passed": all(item["passed"] for item in checks), "checks": checks}


def run_mcp(args) -> dict:
    checks = []
    tools = mcp_request(args.mcp_url, "tools/list", request_id=1)
    tool_names = []
    if tools.get("body", {}).get("result", {}).get("tools"):
        tool_names = [item["name"] for item in tools["body"]["result"]["tools"]]
    expected_tools = {"validate_workflow", "explain_workflow", "suggest_metrics", "get_execution", "list_state_snapshots"}
    checks.append({
        "name": "tools_list",
        "passed": tools["ok"] and expected_tools.issubset(set(tool_names)),
        "status": tools["status"],
        "tool_count": len(tool_names),
        "missing": sorted(expected_tools.difference(tool_names)),
    })

    validate = mcp_tool(args.mcp_url, "validate_workflow", request_id=2)
    valid = validate.get("body", {}).get("result", {}).get("structuredContent", {}).get("valid")
    checks.append({"name": "validate_workflow", "passed": validate["ok"] and valid is True, "status": validate["status"]})

    explain = mcp_tool(args.mcp_url, "explain_workflow", request_id=3)
    steps = explain.get("body", {}).get("result", {}).get("structuredContent", {}).get("steps", [])
    checks.append({"name": "explain_workflow", "passed": explain["ok"] and len(steps) >= 10, "status": explain["status"], "steps": len(steps)})

    metrics = mcp_tool(args.mcp_url, "suggest_metrics", request_id=4)
    suggested = metrics.get("body", {}).get("result", {}).get("structuredContent", {}).get("metrics", [])
    checks.append({"name": "suggest_metrics", "passed": metrics["ok"] and len(suggested) > 0, "status": metrics["status"], "metrics": len(suggested)})

    payload = event_payload()
    workflow = workflow_call(args.workflow_url, payload)
    execution = mcp_tool(args.mcp_url, "get_execution", {"message_id": workflow["message_id"]}, request_id=5)
    snapshot = execution.get("body", {}).get("result", {}).get("structuredContent", {})
    checks.append({
        "name": "get_execution_by_message_id",
        "passed": workflow["status"] == 200 and execution["ok"] and snapshot.get("id") == workflow["message_id"],
        "status": execution["status"],
        "message_id": workflow["message_id"],
    })

    return {"suite": "mcp", "passed": all(item["passed"] for item in checks), "checks": checks}


def write_report(result: dict, suite: str) -> pathlib.Path:
    RESULTS.mkdir(parents=True, exist_ok=True)
    stamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    path = RESULTS / f"{stamp}-{suite}.json"
    result["generated_at"] = now_iso()
    with path.open("w", encoding="utf-8") as file:
        json.dump(result, file, indent=2, ensure_ascii=False)
    latest = RESULTS / f"latest-{suite}.json"
    with latest.open("w", encoding="utf-8") as file:
        json.dump(result, file, indent=2, ensure_ascii=False)
    return path


def run_suite(args, suite: str) -> dict:
    if suite == "regression":
        return run_regression(args)
    if suite == "performance":
        return run_performance(args)
    if suite == "chaos":
        return run_chaos(args)
    if suite == "mcp":
        return run_mcp(args)
    if suite == "all":
        suites = ["regression", "performance", "chaos", "mcp"]
        results = [run_suite(args, item) for item in suites]
        return {"suite": "all", "passed": all(item["passed"] for item in results), "results": results}
    raise ValueError(f"unsupported suite: {suite}")


def main() -> int:
    parser = argparse.ArgumentParser(description="Executa testes do case ecommerce-distributed.")
    parser.add_argument("suite", choices=["regression", "performance", "chaos", "mcp", "all"])
    parser.add_argument("--workflow-url", default="http://localhost:8088")
    parser.add_argument("--graphql-url", default="http://localhost:8090")
    parser.add_argument("--metrics-url", default="http://localhost:8080")
    parser.add_argument("--mcp-url", default="http://localhost:9091")
    parser.add_argument("--count", type=int, default=25)
    parser.add_argument("--concurrency", type=int, default=4)
    parser.add_argument("--recovery-wait-seconds", type=int, default=5)
    args = parser.parse_args()

    result = run_suite(args, args.suite)
    path = write_report(result, args.suite)
    print(json.dumps(result, indent=2, ensure_ascii=False))
    print(f"result_file={path}")
    return 0 if result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
