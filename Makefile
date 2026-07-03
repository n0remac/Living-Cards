SHELL := /bin/bash
.SHELLFLAGS := -Eeuo pipefail -c
.ONESHELL:
.SILENT:

APP_NAME ?= living-card
WEB_ADDR ?= 127.0.0.1:8090
PORT ?= $(shell printf '%s' '$(WEB_ADDR)' | sed 's/.*://')
DEV_MODE ?= true

BIN_DIR ?= $(CURDIR)/.tmp/bin
RUN_DIR ?= $(CURDIR)/.tmp/run
LOG_DIR ?= $(CURDIR)/.tmp/logs
BIN_PATH ?= $(BIN_DIR)/$(APP_NAME)
PID_FILE ?= $(RUN_DIR)/$(APP_NAME).pid
LOG_FILE ?= $(LOG_DIR)/$(APP_NAME).log
STOP_TIMEOUT_SECONDS ?= 10
START_TIMEOUT_SECONDS ?= 10
GOCACHE ?= $(CURDIR)/.tmp/go-build-cache

export GOCACHE

.PHONY: restart stop build start status logs

restart: stop build start

stop:
	[[ "$(PORT)" =~ ^[0-9]+$$ ]] || { echo "ERROR: PORT must be numeric; got '$(PORT)' from WEB_ADDR='$(WEB_ADDR)'"; exit 1; }
	mkdir -p "$(RUN_DIR)"

	if command -v lsof >/dev/null 2>&1; then
		mapfile -t pids < <(lsof -nP -tiTCP:"$(PORT)" -sTCP:LISTEN 2>/dev/null | sort -u)
	elif command -v fuser >/dev/null 2>&1; then
		mapfile -t pids < <(fuser -n tcp "$(PORT)" 2>/dev/null | tr ' ' '\n' | sed '/^$$/d' | sort -u)
	elif command -v ss >/dev/null 2>&1; then
		mapfile -t pids < <(ss -ltnp "sport = :$(PORT)" 2>/dev/null | sed -nE 's/.*pid=([0-9]+).*/\1/p' | sort -u)
	else
		echo "ERROR: cannot search port $(PORT); install lsof, fuser, or ss"
		exit 1
	fi

	if (( $${#pids[@]} == 0 )); then
		echo "No process is listening on port $(PORT)."
		rm -f "$(PID_FILE)"
		exit 0
	fi

	echo "Stopping process(es) on port $(PORT): $${pids[*]}"
	kill -TERM "$${pids[@]}" 2>/dev/null || true

	deadline=$$((SECONDS + $(STOP_TIMEOUT_SECONDS)))
	while (( SECONDS < deadline )); do
		still_running=()
		for pid in "$${pids[@]}"; do
			if kill -0 "$$pid" 2>/dev/null; then
				still_running+=("$$pid")
			fi
		done
		if (( $${#still_running[@]} == 0 )); then
			rm -f "$(PID_FILE)"
			exit 0
		fi
		sleep 1
	done

	echo "Process(es) did not stop after $(STOP_TIMEOUT_SECONDS)s; sending SIGKILL: $${still_running[*]}"
	kill -KILL "$${still_running[@]}" 2>/dev/null || true
	rm -f "$(PID_FILE)"

build:
	echo "Compiling $(APP_NAME)..."
	mkdir -p "$(BIN_DIR)" "$(RUN_DIR)" "$(LOG_DIR)" "$(GOCACHE)"
	go build -o "$(BIN_PATH)" .
	echo "Compiled binary: $(BIN_PATH)"

start:
	[[ "$(PORT)" =~ ^[0-9]+$$ ]] || { echo "ERROR: PORT must be numeric; got '$(PORT)' from WEB_ADDR='$(WEB_ADDR)'"; exit 1; }
	mkdir -p "$(RUN_DIR)" "$(LOG_DIR)"

	echo "Starting $(APP_NAME) on $(WEB_ADDR)..."
	nohup env WEB_ADDR="$(WEB_ADDR)" DEV_MODE="$(DEV_MODE)" "$(BIN_PATH)" >>"$(LOG_FILE)" 2>&1 &
	printf '%s\n' "$$!" >"$(PID_FILE)"

	pid="$$(cat "$(PID_FILE)")"
	echo "Started PID $$pid. Logs: $(LOG_FILE)"

	deadline=$$((SECONDS + $(START_TIMEOUT_SECONDS)))
	while (( SECONDS < deadline )); do
		if ! kill -0 "$$pid" 2>/dev/null; then
			tail -40 "$(LOG_FILE)" >&2 || true
			echo "ERROR: $(APP_NAME) exited during startup"
			exit 1
		fi

		if command -v lsof >/dev/null 2>&1; then
			port_pids="$$(lsof -nP -tiTCP:"$(PORT)" -sTCP:LISTEN 2>/dev/null || true)"
		elif command -v fuser >/dev/null 2>&1; then
			port_pids="$$(fuser -n tcp "$(PORT)" 2>/dev/null | tr ' ' '\n' | sed '/^$$/d' || true)"
		elif command -v ss >/dev/null 2>&1; then
			port_pids="$$(ss -ltnp "sport = :$(PORT)" 2>/dev/null | sed -nE 's/.*pid=([0-9]+).*/\1/p' || true)"
		else
			echo "ERROR: cannot search port $(PORT); install lsof, fuser, or ss"
			exit 1
		fi

		if grep -qx "$$pid" <<<"$$port_pids"; then
			echo "$(APP_NAME) is listening on http://$(WEB_ADDR)"
			exit 0
		fi
		sleep 1
	done

	tail -40 "$(LOG_FILE)" >&2 || true
	echo "ERROR: $(APP_NAME) did not start listening on port $(PORT) within $(START_TIMEOUT_SECONDS)s"
	exit 1

status:
	if [[ -f "$(PID_FILE)" ]]; then
		pid="$$(cat "$(PID_FILE)")"
		if [[ "$$pid" =~ ^[0-9]+$$ ]] && kill -0 "$$pid" 2>/dev/null; then
			echo "$(APP_NAME) is running with PID $$pid."
			exit 0
		fi
	fi
	echo "$(APP_NAME) is not running from $(PID_FILE)."

logs:
	mkdir -p "$(LOG_DIR)"
	touch "$(LOG_FILE)"
	tail -f "$(LOG_FILE)"
