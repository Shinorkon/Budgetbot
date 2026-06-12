# BudgetBot

A production-ready Telegram expense tracking bot powered by Google's Gemini-1.5-flash. It parses natural language messages into structured expense data and logs them to a local text file.

## Features

- **Natural Language Parsing**: Uses Gemini-1.5-flash to extract items and amounts from text.
- **Persistent Storage**: Logs data to `$HOME/expenses.txt` in a clean `YYYY-MM-DD | ITEM | AMOUNT` format.
- **Security**: Only allows an authorized User ID to interact with the bot.
- **VPS Ready**: Built for perpetual execution on Linux.

## Setup

1. **Install Go**: Ensure Go 1.21+ is installed on your system.
2. **Environment Variables**: Create a `.env` file in the root directory with the following:
   ```env
   BOT_API_KEY: your_telegram_bot_token
   GEMINI_API_KEY: your_google_gemini_api_key
   ```
3. **Authorize User**: Open `main.go` and update `AuthorizedUserID` with your actual Telegram User ID.
4. **Install Dependencies**:
   ```bash
   go mod tidy
   ```
5. **Run**:
   ```bash
   go run main.go
   ```

## Dependencies

- [Telegram Bot API (v5)](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [Google Generative AI SDK](https://github.com/google/generative-ai-go)
- [Google API Client](https://google.golang.org/api)

## Storage Format

Expenses are saved in `~/expenses.txt`:
```text
2026-06-12 | Eggs | 70
2026-06-12 | Milk | 25
```
