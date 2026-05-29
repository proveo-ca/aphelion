//go:build linux

package telegramcommands

import (
	"context"
	"fmt"
	"strings"

	"github.com/idolum-ai/aphelion/session"
	"github.com/idolum-ai/aphelion/telegram"
)

func sendTelegramThreadReminder(ctx context.Context, sender commandSender, router commandThreadRouter, chatID int64, senderID int64, thread session.TelegramThread) (int64, error) {
	if sender == nil || router == nil || chatID == 0 || thread.ThreadID <= 0 {
		return 0, nil
	}
	rendered := renderTelegramThreadReminder(thread)
	rows := telegramThreadReminderRows(thread.ThreadID)
	messageID, err := sender.SendInlineKeyboard(ctx, chatID, rendered, rows, nil)
	if err != nil {
		return 0, err
	}
	lastActivity := telegramThreadLastActiveAt(thread)
	if err := router.RecordTelegramThreadReminderMessage(chatID, thread.ThreadID, messageID, telegramThreadReminderSummary(thread), session.TelegramThreadReminderSummaryKind(thread), lastActivity, senderID); err != nil {
		return 0, err
	}
	return messageID, nil
}

func renderTelegramThreadReminder(thread session.TelegramThread) string {
	label := telegramThreadDisplayLabel(thread)
	summary := telegramThreadReminderSummary(thread)
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", label)
	if strings.TrimSpace(summary) == "" {
		summary = "this side thread"
	}
	fmt.Fprintf(&b, "Last time we chatted here, we were talking about %s.\n", summary)
	b.WriteString("Want to pick this thread up again?\n\n")
	b.WriteString("Reply to this message to continue.\n\n")
	b.WriteString("You can also ignore this reminder, or absorb the thread into the main conversation.")
	return b.String()
}

func telegramThreadReminderSummary(thread session.TelegramThread) string {
	if session.TelegramThreadReminderSummaryKind(thread) == "privacy_softened" {
		return "a personal conversation"
	}
	preview := compactThreadPreview(thread.CreatedText)
	if preview == "" {
		return "this side thread"
	}
	return preview
}

func telegramThreadReminderRows(threadID int64) [][]telegram.InlineButton {
	return [][]telegram.InlineButton{{
		{Text: "ignore", CallbackData: encodeTelegramThreadReminderIgnoreCallback(threadID)},
		{Text: "absorb", CallbackData: encodeTelegramThreadReminderAbsorbCallback(threadID)},
	}}
}
