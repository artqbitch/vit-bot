package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"vit-bot/base"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func main() {
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	db, err := base.Connect(ctx)
	if err != nil {
		os.Exit(1)
	}
	defer db.Client.Close()
	app := &application{db: db}
	b, err := bot.New(token, bot.WithDefaultHandler(app.welcome), bot.WithErrorsHandler(func(error) {}))
	if err != nil {
		return
	}
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "box:", bot.MatchTypePrefix, app.answer)
	b.Start(ctx)
}

type application struct{ db *base.Base }

func callbackData(id string) string {
	sum := sha256.Sum256([]byte(id))
	return fmt.Sprintf("box:%x", sum[:16])
}

func (app *application) welcome(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}
	chatID := update.Message.Chat.ID
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	boxes, err := app.db.GetBoxes(queryCtx)
	if err != nil {
		sendText(ctx, b, chatID, "Не удалось загрузить вопросы. Попробуйте /start чуть позже.")
		return
	}
	var rows [][]models.InlineKeyboardButton
	for _, box := range boxes {
		label := box.Question
		if strings.TrimSpace(label) == "" {
			continue
		}
		rows = append(rows, []models.InlineKeyboardButton{{Text: label, CallbackData: callbackData(box.ID)}})
	}
	if len(rows) == 0 {
		sendText(ctx, b, chatID, "Добро пожаловать! Вопросов пока нет. Загляните позже.")
		return
	}
	for start := 0; start < len(rows); start += 50 {
		end := min(start+50, len(rows))
		text := "Добро пожаловать! Выберите интересующий вас вопрос:"
		if start > 0 {
			text = "Другие вопросы:"
		}
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: text, ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: rows[start:end]}})
		if err != nil {
			return
		}
	}
}

func (app *application) answer(ctx context.Context, b *bot.Bot, update *models.Update) {
	query := update.CallbackQuery
	if query == nil {
		return
	}
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{CallbackQueryID: query.ID})
	if query.Message.Message == nil {
		return
	}
	chatID := query.Message.Message.Chat.ID
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	boxes, err := app.db.GetBoxes(queryCtx)
	if err != nil {
		sendText(ctx, b, chatID, "Не удалось загрузить ответ. Попробуйте ещё раз чуть позже.")
		return
	}
	for _, box := range boxes {
		if callbackData(box.ID) != query.Data {
			continue
		}
		answer := strings.TrimSpace(box.Answer)
		if answer == "" {
			answer = "Ответ на этот вопрос пока не добавлен."
		}
		sendText(ctx, b, chatID, answer)
		return
	}
	sendText(ctx, b, chatID, "Этот вопрос больше недоступен. Нажмите /start, чтобы обновить список.")
}

func sendText(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	chars := []rune(text)
	for len(chars) > 0 {
		n := min(2000, len(chars))
		if _, err := b.SendMessage(ctx, &bot.SendMessageParams{ChatID: chatID, Text: string(chars[:n])}); err != nil {
			return
		}
		chars = chars[n:]
	}
}
