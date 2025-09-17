# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: run quality control checks
.PHONY: audit
audit: test
	go mod tidy -diff
	go mod verify
	test -z "$(shell gofmt -l .)"
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## test: run all tests
.PHONY: test
test:
	go test -v -race -buildvcs ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## upgradeable: list direct dependencies that have upgrades available
.PHONY: upgradeable
upgradeable:
	@go run github.com/oligot/go-mod-upgrade@latest

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## tidy: tidy modfiles and format .go files
.PHONY: tidy
tidy:
	go mod tidy -v
	go fmt ./...

## build: build the padd application
.PHONY: build
build:
	go build -o=./tmp/bin/padd ./cmd/padd/...

## run: run the padd application
.PHONY: run
run: build
	./tmp/bin/padd -data ./data

## run/live: run the application with reloading on file changes
.PHONY: run/live
run/live:
	go run github.com/air-verse/air@v1.62.0 \
		--build.cmd "make build" \
		--build.bin "tmp/bin/padd" \
		--build.args_bin "-data, ./data, -keys-dir ./data/keys" \
		--build.delay "250" \
		--build.exclude_dir "data, node_modules" \
		--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico" \
		--build.send_interrupt "true" \
		--build.kill_delay "1000" \
		--misc.clean_on_exit "true"

# ==================================================================================== #
# INSTALLATION
# ==================================================================================== #

## install: install the padd application using go install
.PHONY: install
install:
	go install ./cmd/padd/...
	@echo "padd installed to $(shell go env GOPATH)/bin/padd"

## install-service: install the service management script
.PHONY: install-service
install-service:
	@mkdir -p $(HOME)/.local/bin
	@cp scripts/padd-service.sh $(HOME)/.local/bin/padd-service
	@chmod +x $(HOME)/.local/bin/padd-service
	@echo "Service script installed to $(HOME)/.local/bin/padd-service"
	@echo "Make sure $(HOME)/.local/bin is in your PATH"

## install-all: install both the application and service script
.PHONY: install-all
install-all: install install-service
	@echo "Installation complete!"
	@echo "Usage: padd-service {start|stop|restart|status|logs|config}"

# ==================================================================================== #
# SERVICE MANAGEMENT
# ==================================================================================== #

## service-info: information about padd-service
.PHONY: service-info
service-info:
	@if [ -x $(HOME)/.local/bin/padd-service ]; then \
  		$(HOME)/.local/bin/padd-service \
	else \
		echo "Service script not installed. Run 'make install-service' first."; \
	fi

## reinstall-and-restart: install an updated binary and restart service
.PHONY: reinstall-and-restart
reinstall-and-restart: install
	@if [ -x $(HOME)/.local/bin/padd-service ]; then \
		echo "Restarting service with updated binary..."; \
		$(HOME)/.local/bin/padd-service restart; \
	else \
		echo "Service script not installed. The binary has been updated, but you'll need to manually restart if running."; \
	fi
