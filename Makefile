.PHONY: build run deploy clean vet tidy

BINARY=budgetbot

tidy:
	go mod tidy

vet: tidy
	go vet ./...

build: vet
	go build -o $(BINARY) .

run: build
	./$(BINARY)

deploy: build
	sudo systemctl stop budgetbot 2>/dev/null || true
	sudo cp budgetbot.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable budgetbot
	sudo systemctl start budgetbot
	@echo "================================================"
	@echo " BudgetBot deployed. Check logs with:"
	@echo "   journalctl -u budgetbot -f"
	@echo "================================================"

clean:
	rm -f $(BINARY)
	@echo "Cleaned."
