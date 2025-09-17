#!/bin/bash

# PADD Service Management Script
# Supports starting, stopping, restarting, and checking status of PADD service

# Configuration - can be overridden by environment variables
PADD_PORT="${PADD_PORT:-4242}"
PADD_ADDR="${PADD_ADDR:-localhost}"
PADD_DATA_DIR="${PADD_DATA_DIR:-$HOME/.local/share/padd}"
PADD_BINARY="${PADD_BINARY:-$(which padd)}"
PADD_PID_FILE="$PADD_DATA_DIR/service/padd.pid"
PADD_LOG_FILE="$PADD_DATA_DIR/service/padd.log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Ensure data directory exists
mkdir -p "$(dirname "$PADD_PID_FILE")"
mkdir -p "$(dirname "$PADD_LOG_FILE")"

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_binary() {
    if [ ! -f "$PADD_BINARY" ] || [ ! -x "$PADD_BINARY" ]; then
        log_error "PADD binary not found or not executable at: $PADD_BINARY"
        log_info "Try running 'make install' first, or set PADD_BINARY environment variable"
        exit 1
    fi
}

is_running() {
    if [ -f "$PADD_PID_FILE" ]; then
        local pid=$(cat "$PADD_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            return 0
        else
            # PID file exists but process is dead, clean it up
            rm -f "$PADD_PID_FILE"
        fi
    fi
    return 1
}

start_service() {
    if is_running; then
        log_warn "PADD is already running (PID: $(cat "$PADD_PID_FILE"))"
        log_info "Access it at: http://$PADD_ADDR:$PADD_PORT"
        return 1
    fi

    check_binary

    log_info "Starting PADD service..."
    log_info "Port: $PADD_PORT"
    log_info "Address: $PADD_ADDR"
    log_info "Data directory: $PADD_DATA_DIR"
    log_info "Log file: $PADD_LOG_FILE"

    # Start PADD in background
    nohup "$PADD_BINARY" \
        -port "$PADD_PORT" \
        -addr "$PADD_ADDR" \
        -data "$PADD_DATA_DIR" \
        > "$PADD_LOG_FILE" 2>&1 &

    local pid=$!
    echo "$pid" > "$PADD_PID_FILE"

    # Give it a moment to start
    sleep 2

    if is_running; then
        log_info "PADD started successfully (PID: $pid)"
        log_info "Access it at: http://$PADD_ADDR:$PADD_PORT"
        log_info "Logs: $PADD_LOG_FILE"
    else
        log_error "Failed to start PADD"
        log_info "Check the log file for errors: $PADD_LOG_FILE"
        return 1
    fi
}

stop_service() {
    if ! is_running; then
        log_warn "PADD is not running"
        return 1
    fi

    local pid=$(cat "$PADD_PID_FILE")
    log_info "Stopping PADD service (PID: $pid)..."

    # Try graceful shutdown first
    kill "$pid" 2>/dev/null

    # Wait up to 10 seconds for graceful shutdown
    for i in {1..10}; do
        if ! is_running; then
            log_info "PADD stopped successfully"
            return 0
        fi
        sleep 1
    done

    # Force kill if still running
    log_warn "Graceful shutdown failed, forcing termination..."
    kill -9 "$pid" 2>/dev/null
    rm -f "$PADD_PID_FILE"
    log_info "PADD force stopped"
}

restart_service() {
    log_info "Restarting PADD service..."
    stop_service
    sleep 2
    start_service
}

status_service() {
    if is_running; then
        local pid=$(cat "$PADD_PID_FILE")
        log_info "PADD is running (PID: $pid)"
        log_info "Access it at: http://$PADD_ADDR:$PADD_PORT"
        log_info "Data directory: $PADD_DATA_DIR"
        log_info "Log file: $PADD_LOG_FILE"

        # Show recent log entries if available
        if [ -f "$PADD_LOG_FILE" ]; then
            echo
            echo "Recent log entries:"
            tail -5 "$PADD_LOG_FILE" 2>/dev/null || echo "No log entries found"
        fi
    else
        log_warn "PADD is not running"
        return 1
    fi
}

show_logs() {
    if [ -f "$PADD_LOG_FILE" ]; then
        if [ "$1" = "-f" ] || [ "$1" = "--follow" ]; then
            log_info "Following PADD logs (Ctrl+C to exit)..."
            tail -f "$PADD_LOG_FILE"
        else
            log_info "PADD logs:"
            cat "$PADD_LOG_FILE"
        fi
    else
        log_error "Log file not found: $PADD_LOG_FILE"
    fi
}

show_config() {
    echo "PADD Service Configuration:"
    echo "  Binary: $PADD_BINARY"
    echo "  Port: $PADD_PORT"
    echo "  Address: $PADD_ADDR"
    echo "  Data Directory: $PADD_DATA_DIR"
    echo "  PID File: $PADD_PID_FILE"
    echo "  Log File: $PADD_LOG_FILE"
}

show_usage() {
    cat << EOF
PADD Service Management Script

Location: $0

Usage: padd-service {start|stop|restart|status|logs|config}

Commands:
  start     Start the PADD service
  stop      Stop the PADD service
  restart   Restart the PADD service
  status    Show service status
  logs      Show service logs (use -f to follow)
  config    Show current configuration

Environment Variables:
  PADD_PORT       Port to run on (default: 4242)
  PADD_ADDR       Address to bind to (default: localhost)
  PADD_DATA_DIR   Data directory (default: \$HOME/.local/share/padd)
  PADD_BINARY     Path to PADD binary (default: auto-detect)

Examples:
  padd-service start                    # Start with default settings
  PADD_PORT=8080 padd-service start     # Start on port 8080
  padd-service logs -f                  # Follow logs
EOF
}

case "$1" in
    start)
        start_service
        ;;
    stop)
        stop_service
        ;;
    restart)
        restart_service
        ;;
    status)
        status_service
        ;;
    logs)
        show_logs "$2"
        ;;
    config)
        show_config
        ;;
    *)
        show_usage
        exit 0
        ;;
esac
