# BudgetBot

A production-ready Telegram expense tracking bot powered by Google's **Gemini-1.5-flash**. Send natural language messages like "bought eggs for 70" and the bot logs the structured expense automatically.

## Features

- **Natural Language Parsing** — Uses Gemini-1.5-flash to extract items and amounts from plain text.
- **Persistent Storage** — Logs to `$HOME/expenses.txt` in `YYYY-MM-DD | ITEM | AMOUNT` format.
- **Authorization** — Only your Telegram User ID can interact with the bot.
- **Error Feedback** — You'll get a Telegram reply if something goes wrong (API error, timeout, etc.).
- **VPS Ready** — Ships with a systemd service file for always-on deployment.

## Prerequisites

- Go 1.21+
- A [Telegram Bot Token](https://t.me/BotFather)
- A [Google Gemini API Key](https://aistudio.google.com/app/apikey)
- Your **Telegram User ID** — get it by messaging [@userinfobot](https://t.me/userinfobot) on Telegram.

## Quick Start

### 1. Clone & Configure

```bash
git clone <your-repo-url> && cd Budgetbot
cp .env.example .env
```

Fill in `.env` with your values:

```env
BOT_API_KEY=1234567890:ABCdefGHIjklmNOPqrstUVwxyz
GEMINI_API_KEY=AIzaSy...
AUTHORIZED_USER_ID=123456789
```

### 2. Run

```bash
make run
```

Or step-by-step:

```bash
go mod tidy
go build -o budgetbot .
./budgetbot
```

### 3. Use It

Send a message to your bot on Telegram:

```
bought eggs for 70
```

The bot replies:

```
✅ Logged 70 MVR for Eggs.
```

Expenses are saved to `~/expenses.txt`:

```text
2026-06-15 | Eggs | 70
2026-06-15 | Milk | 25
```

---

## VPS Deployment (systemd)

Install the bot as a systemd service for perpetual background execution.

### Build & Install

```bash
make build
```

Or use the deploy target (builds, installs the service, and starts it):

```bash
make deploy
```

### Manual Service Setup

```bash
sudo cp budgetbot.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable budgetbot
sudo systemctl start budgetbot
```

### Check Status & Logs

```bash
# View live logs
journalctl -u budgetbot -f

# Check service status
systemctl status budgetbot

# View recent logs
journalctl -u budgetbot --since "5 minutes ago"
```

### Stopping / Restarting

```bash
sudo systemctl stop budgetbot
sudo systemctl restart budgetbot
```

---

## Configuration

All configuration is done via the `.env` file (or environment variables):

| Variable | Description | Required |
|----------|-------------|----------|
| `BOT_API_KEY` | Telegram Bot Token from @BotFather | ✅ Yes |
| `GEMINI_API_KEY` | Google Gemini API key | ✅ Yes |
| `AUTHORIZED_USER_ID` | Your numeric Telegram User ID | ✅ Yes |

---

## Available Commands

```bash
make tidy    # Download dependencies
make vet     # Run code analysis
make build   # Compile binary
make run     # Build & run
make deploy  # Build & deploy as systemd service
make clean   # Remove binary
```

---

## Dependencies

- [Telegram Bot API (v5)](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [Google Generative AI SDK](https://github.com/google/generative-ai-go)
- [Google API Client](https://google.golang.org/api)

---

## Project Structure

```
Budgetbot/
├── main.go              # Bot entry point & logic
├── go.mod               # Go module definition
├── Makefile             # Build automation
├── budgetbot.service    # systemd service unit
├── .env.example         # Configuration template
└── README.md            # This file
```
