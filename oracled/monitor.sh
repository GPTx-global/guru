#!/bin/bash

# Oracle Daemon Monitor Script
# This script monitors the oracle daemon process and restarts it if necessary

# Configuration
DAEMON_NAME="oracled"
DAEMON_PATH="./cmd/oracled"
PID_FILE="/tmp/oracled.pid"
LOG_FILE="/tmp/oracled_monitor.log"
MAX_RESTART_ATTEMPTS=5
RESTART_DELAY=10
HEALTH_CHECK_INTERVAL=30
CONSECUTIVE_FAILURES_THRESHOLD=3

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Global variables
consecutive_failures=0
restart_count=0

# Logging function
log_message() {
    local level=$1
    local message=$2
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    case $level in
        "INFO")
            echo -e "${GREEN}[INFO ]${NC} ${timestamp} - ${message}" | tee -a "$LOG_FILE"
            ;;
        "WARN")
            echo -e "${YELLOW}[WARN ]${NC} ${timestamp} - ${message}" | tee -a "$LOG_FILE"
            ;;
        "ERROR")
            echo -e "${RED}[ERROR]${NC} ${timestamp} - ${message}" | tee -a "$LOG_FILE"
            ;;
        "DEBUG")
            echo -e "${BLUE}[DEBUG]${NC} ${timestamp} - ${message}" | tee -a "$LOG_FILE"
            ;;
    esac
}

# Check if daemon process is running
is_daemon_running() {
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        if ps -p "$pid" > /dev/null 2>&1; then
            return 0  # Process is running
        else
            # PID file exists but process is not running
            rm -f "$PID_FILE"
            return 1
        fi
    else
        return 1  # PID file doesn't exist
    fi
}

# Start the daemon
start_daemon() {
    log_message "INFO" "Starting Oracle Daemon..."
    
    # Check if already running
    if is_daemon_running; then
        log_message "WARN" "Daemon is already running (PID: $(cat $PID_FILE))"
        return 0
    fi
    
    # Change to daemon directory
    if [ ! -f "$DAEMON_PATH" ]; then
        log_message "ERROR" "Daemon binary not found at $DAEMON_PATH"
        return 1
    fi
    
    # Start daemon in background
    cd "$(dirname "$DAEMON_PATH")"
    nohup ./oracled > /tmp/oracled.out 2>&1 &
    local daemon_pid=$!
    
    # Save PID
    echo "$daemon_pid" > "$PID_FILE"
    
    # Wait a moment and check if it's still running
    sleep 3
    if is_daemon_running; then
        log_message "INFO" "Daemon started successfully (PID: $daemon_pid)"
        consecutive_failures=0
        return 0
    else
        log_message "ERROR" "Failed to start daemon"
        rm -f "$PID_FILE"
        return 1
    fi
}

# Stop the daemon
stop_daemon() {
    log_message "INFO" "Stopping Oracle Daemon..."
    
    if [ -f "$PID_FILE" ]; then
        local pid=$(cat "$PID_FILE")
        
        # Send SIGTERM for graceful shutdown
        if ps -p "$pid" > /dev/null 2>&1; then
            log_message "INFO" "Sending SIGTERM to process $pid"
            kill -TERM "$pid"
            
            # Wait for graceful shutdown (max 30 seconds)
            local countdown=30
            while [ $countdown -gt 0 ] && ps -p "$pid" > /dev/null 2>&1; do
                sleep 1
                countdown=$((countdown - 1))
            done
            
            # Force kill if still running
            if ps -p "$pid" > /dev/null 2>&1; then
                log_message "WARN" "Graceful shutdown timeout, forcing kill"
                kill -KILL "$pid"
                sleep 2
            fi
            
            if ! ps -p "$pid" > /dev/null 2>&1; then
                log_message "INFO" "Daemon stopped successfully"
            else
                log_message "ERROR" "Failed to stop daemon"
            fi
        fi
        
        rm -f "$PID_FILE"
    else
        log_message "INFO" "Daemon is not running"
    fi
}

# Restart the daemon
restart_daemon() {
    restart_count=$((restart_count + 1))
    
    if [ $restart_count -gt $MAX_RESTART_ATTEMPTS ]; then
        log_message "ERROR" "Maximum restart attempts ($MAX_RESTART_ATTEMPTS) exceeded"
        log_message "ERROR" "Manual intervention required"
        exit 1
    fi
    
    log_message "WARN" "Restarting daemon (attempt $restart_count/$MAX_RESTART_ATTEMPTS)"
    
    stop_daemon
    sleep $RESTART_DELAY
    
    if start_daemon; then
        log_message "INFO" "Daemon restarted successfully"
        return 0
    else
        log_message "ERROR" "Failed to restart daemon"
        return 1
    fi
}

# Perform health check
health_check() {
    local health_status=0
    
    # Check if process is running
    if ! is_daemon_running; then
        log_message "WARN" "Health check failed: Process not running"
        health_status=1
    fi
    
    # Check if daemon is responsive (optional)
    # You can add more sophisticated health checks here
    # For example, checking if daemon responds to specific signals
    # or monitoring log file for recent activity
    
    # Check log file for recent activity (last 2 minutes)
    if [ -f "/tmp/oracled.out" ]; then
        local recent_activity=$(find /tmp/oracled.out -mmin -2 2>/dev/null)
        if [ -z "$recent_activity" ]; then
            log_message "DEBUG" "No recent activity detected in daemon log"
            # This is not necessarily a failure, just informational
        fi
    fi
    
    # Check for error patterns in log
    if [ -f "/tmp/oracled.out" ]; then
        local error_count=$(tail -n 50 /tmp/oracled.out 2>/dev/null | grep -c "ERROR\|FATAL\|panic" 2>/dev/null || echo "0")
        # Ensure error_count is a valid integer
        error_count=$(echo "$error_count" | tr -d '\n\r' | sed 's/[^0-9]//g')
        if [ -z "$error_count" ]; then
            error_count=0
        fi
        if [ "$error_count" -gt 10 ]; then
            log_message "WARN" "High error count detected in daemon log: $error_count errors in last 50 lines"
            health_status=1
        fi
    fi
    
    return $health_status
}

# Monitor loop
monitor_daemon() {
    log_message "INFO" "Starting Oracle Daemon monitor"
    log_message "INFO" "Monitor configuration:"
    log_message "INFO" "  - Health check interval: ${HEALTH_CHECK_INTERVAL}s"
    log_message "INFO" "  - Consecutive failures threshold: $CONSECUTIVE_FAILURES_THRESHOLD"
    log_message "INFO" "  - Max restart attempts: $MAX_RESTART_ATTEMPTS"
    log_message "INFO" "  - Restart delay: ${RESTART_DELAY}s"
    
    # Start daemon if not running
    if ! is_daemon_running; then
        log_message "INFO" "Daemon not running, starting..."
        start_daemon
    fi
    
    while true; do
        if health_check; then
            if [ $consecutive_failures -gt 0 ]; then
                log_message "INFO" "Health check passed, system recovered"
            fi
            consecutive_failures=0
        else
            consecutive_failures=$((consecutive_failures + 1))
            log_message "WARN" "Health check failed (consecutive failures: $consecutive_failures/$CONSECUTIVE_FAILURES_THRESHOLD)"
            
            if [ $consecutive_failures -ge $CONSECUTIVE_FAILURES_THRESHOLD ]; then
                log_message "WARN" "Consecutive failure threshold reached, attempting restart"
                if restart_daemon; then
                    consecutive_failures=0
                else
                    log_message "ERROR" "Restart failed"
                fi
            fi
        fi
        
        # Sleep for health check interval
        sleep $HEALTH_CHECK_INTERVAL
    done
}

# Signal handlers
cleanup() {
    log_message "INFO" "Monitor received shutdown signal"
    stop_daemon
    exit 0
}

# Set up signal handlers
trap cleanup SIGTERM SIGINT

# Main script logic
case "${1:-monitor}" in
    "start")
        start_daemon
        ;;
    "stop")
        stop_daemon
        ;;
    "restart")
        restart_daemon
        ;;
    "status")
        if is_daemon_running; then
            echo -e "${GREEN}Oracle Daemon is running${NC} (PID: $(cat $PID_FILE))"
        else
            echo -e "${RED}Oracle Daemon is not running${NC}"
        fi
        ;;
    "monitor")
        monitor_daemon
        ;;
    "logs")
        if [ -f "/tmp/oracled.out" ]; then
            tail -f /tmp/oracled.out
        else
            echo "No daemon log file found"
        fi
        ;;
    "monitor-logs")
        if [ -f "$LOG_FILE" ]; then
            tail -f "$LOG_FILE"
        else
            echo "No monitor log file found"
        fi
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|monitor|logs|monitor-logs}"
        echo ""
        echo "Commands:"
        echo "  start         - Start the daemon"
        echo "  stop          - Stop the daemon"
        echo "  restart       - Restart the daemon"
        echo "  status        - Show daemon status"
        echo "  monitor       - Start monitoring (default)"
        echo "  logs          - Show daemon logs"
        echo "  monitor-logs  - Show monitor logs"
        exit 1
        ;;
esac 