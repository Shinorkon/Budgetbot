package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// AuthorizedUserID is the Telegram ID of the user allowed to use this bot.
// Loaded from AUTHORIZED_USER_ID in .env or environment variables.
var AuthorizedUserID int64

func main() {
	// Load .env file manually
	loadEnv(".env")

	// 1. Load Environment Variables
	botToken := os.Getenv("BOT_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if botToken == "" || geminiKey == "" {
		log.Fatal("Missing environment variables: BOT_API_KEY and GEMINI_API_KEY are required in .env or environment.")
	}

	// 2. Load AuthorizedUserID from environment
	authIDStr := os.Getenv("AUTHORIZED_USER_ID")
	if authIDStr == "" {
		log.Fatal("Missing environment variable: AUTHORIZED_USER_ID is required in .env or environment.")
	}
	uid, err := strconv.ParseInt(authIDStr, 10, 64)
	if err != nil {
		log.Fatalf("Invalid AUTHORIZED_USER_ID: must be a numeric Telegram User ID, got %q", authIDStr)
	}
	AuthorizedUserID = uid

	// 3. Initialize Telegram Bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize Telegram Bot: %v", err)
	}
	log.Printf("[INFO] Authorized on account %s", bot.Self.UserName)
	log.Printf("[INFO] Authorized User ID: %d", AuthorizedUserID)

	// 4. Initialize Gemini Client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiKey))
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize Gemini Client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.0)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text("Extract the ITEM and the AMOUNT from the user's message. Capitalize the first letter of the ITEM. Strip all currency symbols from the AMOUNT and return only raw numbers. Reply ONLY in the format: ITEM|AMOUNT"),
		},
	}

	// 5. Set up updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("[INFO] Bot is now listening for messages...")

	// 6. Main Loop
	for update := range updates {
		if update.Message == nil || update.Message.Text == "" {
			continue
		}

		// Security: Check Authorized User ID
		if update.Message.From.ID != AuthorizedUserID {
			log.Printf("[WARN] Unauthorized attempt from UserID: %d (%s)", update.Message.From.ID, update.Message.From.UserName)
			continue
		}

		log.Printf("[INFO] Processing message from %d: %q", update.Message.From.ID, update.Message.Text)
		handleMessage(ctx, bot, model, update.Message)
	}
}

func loadEnv(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return // Ignore if file doesn't exist
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Support both ':' and '=' as delimiters
		delimiter := ":"
		if !strings.Contains(line, ":") && strings.Contains(line, "=") {
			delimiter = "="
		}

		parts := strings.SplitN(line, delimiter, 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			os.Setenv(key, value)
		}
	}
}

func replyToUser(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, text string) {
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyToMessageID = msg.MessageID
	if _, err := bot.Send(reply); err != nil {
		log.Printf("[ERROR] Failed to send Telegram message: %v", err)
	}
}

func handleMessage(ctx context.Context, bot *tgbotapi.BotAPI, model *genai.GenerativeModel, msg *tgbotapi.Message) {
	// Create a per-message timeout to prevent hanging
	msgCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Request Gemini to parse the message
	resp, err := model.GenerateContent(msgCtx, genai.Text(msg.Text))
	if err != nil {
		if msgCtx.Err() == context.DeadlineExceeded {
			log.Printf("[ERROR] Gemini request timed out for: %s", msg.Text)
			replyToUser(bot, msg, "Sorry, the request timed out. Please try again.")
		} else {
			log.Printf("[ERROR] Gemini API error: %v", err)
			replyToUser(bot, msg, "Sorry, I couldn't process that. Please try again.")
		}
		return
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Printf("[WARN] Gemini returned no results for: %s", msg.Text)
		replyToUser(bot, msg, "I couldn't understand the expense. Try: \"bought eggs for 70\"")
		return
	}

	// Extract the formatted string safely
	var responseStr string
	if part, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		responseStr = string(part)
	} else {
		log.Printf("[ERROR] Gemini returned non-text part: %v", resp.Candidates[0].Content.Parts[0])
		replyToUser(bot, msg, "Sorry, I got an unexpected response. Please try again.")
		return
	}
	responseStr = strings.TrimSpace(responseStr)

	// Split by |
	data := strings.Split(responseStr, "|")
	if len(data) != 2 {
		log.Printf("[WARN] Unexpected format from Gemini: %s", responseStr)
		replyToUser(bot, msg, "I couldn't understand the expense. Try: \"bought eggs for 70\"")
		return
	}

	item := strings.TrimSpace(data[0])
	amount := strings.TrimSpace(data[1])

	// Log to file
	if err := logExpense(item, amount); err != nil {
		log.Printf("[ERROR] File I/O error: %v", err)
		replyToUser(bot, msg, "There was a storage error. Please contact the admin.")
		return
	}

	log.Printf("[INFO] Logged expense: %s | %s", item, amount)

	// Send confirmation back to Telegram
	confirmMsg := fmt.Sprintf("✅ Logged **%s MVR** for **%s**.", amount, item)
	replyToUser(bot, msg, confirmMsg)
}

func logExpense(item, amount string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %w", err)
	}

	filePath := filepath.Join(home, "expenses.txt")
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open expenses file: %w", err)
	}
	defer f.Close()

	date := time.Now().Format("2006-01-02")
	line := fmt.Sprintf("%s | %s | %s\n", date, item, amount)

	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("failed to write to expenses file: %w", err)
	}

	return nil
}
