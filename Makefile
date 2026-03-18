BINARY_NAME=mykola-1-bot
SERVICE_NAME=mykola-bot
BUILD_DIR=.

.PHONY: all build chmod restart deploy status logs clean

all: deploy

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME)

chmod:
	chmod +x $(BUILD_DIR)/$(BINARY_NAME)

restart:
	@if command -v systemctl >/dev/null 2>&1; then \
		echo "Restarting $(SERVICE_NAME).service..."; \
		sudo systemctl restart $(SERVICE_NAME); \
		sudo systemctl status $(SERVICE_NAME) --no-pager -l; \
	else \
		echo "systemctl not found, skipping restart"; \
	fi

deploy: build chmod restart

status:
	@sudo systemctl status $(SERVICE_NAME) --no-pager -l

logs:
	@journalctl -u $(SERVICE_NAME) -f

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)