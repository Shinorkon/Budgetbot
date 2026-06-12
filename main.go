package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// AuthorizedUserID is the Telegram ID of the user allowed to use this bot.
// Set this to your actual Telegram User ID.
const AuthorizedUserID int64 = 0

func main() {
	// Load .env file manually
	loadEnv(".env")

	// 1. Load Environment Variables
	botToken := os.Getenv("BOT_API_KEY")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if botToken == "" || geminiKey == "" {
		log.Fatal("Missing environment variables: BOT_API_KEY and GEMINI_API_KEY are required in .env or environment.")
	}

	// 2. Initialize Telegram Bot
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatalf("Failed to initialize Telegram Bot: %v", err)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	// 3. Initialize Gemini Client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(geminiKey))
	if err != nil {
		log.Fatalf("Failed to initialize Gemini Client: %v", err)
	}
	defer client.Close()

	model := client.GenerativeModel("gemini-1.5-flash")
	model.SetTemperature(0.0)
	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text("Extract the ITEM and the AMOUNT from the user's message. Capitalize the first letter of the ITEM. Strip all currency symbols from the AMOUNT and return only raw numbers. Reply ONLY in the format: ITEM|AMOUNT"),
		},
	}

	// 4. Set up updates
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("Bot is now listening for messages...")

	// 5. Main Loop
	for update := range updates {
		if update.Message == nil || update.Message.Text == "" {
			continue
		}

		// Security: Check Authorized User ID
		if update.Message.From.ID != AuthorizedUserID {
			log.Printf("Unauthorized attempt from UserID: %d (%s)", update.Message.From.ID, update.Message.From.UserName)
			continue
		}

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

func handleMessage(ctx context.Context, bot *tgbotapi.BotAPI, model *genai.GenerativeModel, msg *tgbotapi.Message) {
	// Request Gemini to parse the message
	resp, err := model.GenerateContent(ctx, genai.Text(msg.Text))
	if err != nil {
		log.Printf("Gemini API error: %v", err)
		return
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		log.Printf("Gemini returned no results for: %s", msg.Text)
		return
	}

	// Extract the formatted string safely
	var responseStr string
	if part, ok := resp.Candidates[0].Content.Parts[0].(genai.Text); ok {
		responseStr = string(part)
	} else {
		log.Printf("Gemini returned non-text part: %v", resp.Candidates[0].Content.Parts[0])
		return
	}
	responseStr = strings.TrimSpace(responseStr)

	// Split by |
	data := strings.Split(responseStr, "|")
	if len(data) != 2 {
		log.Printf("Unexpected format from Gemini: %s", responseStr)
		return
	}

	item := strings.TrimSpace(data[0])
	amount := strings.TrimSpace(data[1])

	// Log to file
	if err := logExpense(item, amount); err != nil {
		log.Printf("File I/O error: %v", err)
		return
	}

	// Send confirmation back to Telegram
	confirmMsg := fmt.Sprintf("Logged %s MVR for %s.", amount, item)
	reply := tgbotapi.NewMessage(msg.Chat.ID, confirmMsg)
	reply.ReplyToMessageID = msg.MessageID

	if _, err := bot.Send(reply); err != nil {
		log.Printf("Failed to send Telegram message: %v", err)
	}
}

func logExpense(item, amount string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not find home directory: %v", err)
	}

	filePath := filepath.Join(home, "expenses.txt")
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	date := time.Now().Format("2006-01-02")
	line := fmt.Sprintf("%s | %s | %s\n", date, item, amount)

	if _, err := f.WriteString(line); err != nil {
		return err
	}

	return nil
}
