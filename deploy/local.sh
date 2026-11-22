#!/usr/bin/env bash
# 本机一键部署：构建 H5 与后端二进制、迁移数据库、后台启动并做健康检查。
#
# 用法：
#   ./deploy/local.sh [start]          构建并启动（已在运行则先停止）
#   ./deploy/local.sh start --no-build 跳过构建，用现有二进制启动
#   ./deploy/local.sh restart          不构建，直接重启
#   ./deploy/local.sh stop | status | logs
#
# 图片识别需要先 export DASHSCOPE_API_KEY=...，未设置时服务照常启动。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BACKEND="$ROOT/backend"
BIN="$BACKEND/bin/harmony"
CONFIG="configs/config-local.yaml"
PID_FILE="$BACKEND/data/harmony.pid"
LOG_FILE="$BACKEND/data/harmony.log"
PORT=8080
URL="http://127.0.0.1:$PORT"
USAGE="用法：$0 [start [--no-build]|stop|restart|status|logs]"

# 配置里的相对路径以 backend/ 为基准，所有命令都在这里执行。
cd "$BACKEND"
mkdir -p data

# 输出仍在运行的 harmony 进程号；pid 文件过期或已被其他进程复用时清掉它。
running_pid() {
  [ -f "$PID_FILE" ] || return 1
  local pid
  pid="$(cat "$PID_FILE")"
  if ps -p "$pid" -o command= 2>/dev/null | grep -q "$BIN"; then
    echo "$pid"
  else
    rm -f "$PID_FILE"
    return 1
  fi
}

stop() {
  local pid
  if ! pid="$(running_pid)"; then
    echo "harmony 未在运行"
    return
  fi
  echo "停止 harmony (pid $pid)"
  kill "$pid"
  for _ in $(seq 1 10); do
    kill -0 "$pid" 2>/dev/null || break
    sleep 1
  done
  if kill -0 "$pid" 2>/dev/null; then
    kill -9 "$pid"
  fi
  rm -f "$PID_FILE"
}

start() {
  case "${1:-}" in
    "") echo "==> 构建 H5 与后端"; make -C "$ROOT" release ;;
    --no-build) ;;
    *) echo "$USAGE" >&2; exit 2 ;;
  esac
  [ -x "$BIN" ] || { echo "找不到 $BIN，请去掉 --no-build 重新构建" >&2; exit 1; }

  stop
  if lsof -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    echo "端口 $PORT 被其他进程占用：" >&2
    lsof -iTCP:"$PORT" -sTCP:LISTEN >&2
    exit 1
  fi

  echo "==> 迁移数据库"
  "$BIN" migrate -config "$CONFIG"

  [ -n "${DASHSCOPE_API_KEY:-}" ] || echo "提示：未设置 DASHSCOPE_API_KEY，图片识别不可用"

  echo "==> 启动 harmony"
  nohup "$BIN" -config "$CONFIG" >>"$LOG_FILE" 2>&1 &
  local pid=$!
  echo "$pid" >"$PID_FILE"

  for _ in $(seq 1 15); do
    if curl -fsS "$URL/api/v1/health" >/dev/null 2>&1; then
      echo "启动成功：$URL  (日志 $LOG_FILE)"
      return
    fi
    # 进程已经退出就不必再等。
    kill -0 "$pid" 2>/dev/null || break
    sleep 1
  done
  echo "启动失败，最近日志：" >&2
  tail -n 50 "$LOG_FILE" >&2
  stop >/dev/null
  exit 1
}

status() {
  local pid
  if pid="$(running_pid)"; then
    echo "harmony 运行中 (pid $pid)：$URL"
  else
    echo "harmony 未在运行"
  fi
}

logs() {
  [ -f "$LOG_FILE" ] || { echo "还没有日志：$LOG_FILE" >&2; exit 1; }
  tail -f "$LOG_FILE"
}

case "${1:-start}" in
  start) start "${2:-}" ;;
  stop) stop ;;
  restart) stop; start --no-build ;;
  status) status ;;
  logs) logs ;;
  *) echo "$USAGE" >&2; exit 2 ;;
esac
