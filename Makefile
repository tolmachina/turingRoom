.PHONY: run test build clean frontend1 frontend2 local

# Run the main application with logging
run:
	go run cmd/main.go 2>&1 | tee logs/app.log

# Run tests
test:
	go test ./...

# Build the application
build:
	go build -o bin/app cmd/main.go

# Clean built binaries and logs
clean:
	rm -f bin/app
	rm -f logs/*.log

# Run first frontend instance (assumes you have Live Server VS Code extension)
frontend1:
	code view/resources/index.html -r

# Run second frontend instance
frontend2:
	python3 -m http.server 8000 --directory view/resources

# Full local setup (run these in separate terminal windows)
local:
	@echo "Run these commands in separate terminals:"
	@echo "make run"
	@echo "make frontend1"
	@echo "make frontend2"

# Ensure log directory exists
logs:
	mkdir -p logs