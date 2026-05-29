//go:build linux

package telegramcommands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/idolum-ai/aphelion/core"
	"github.com/idolum-ai/aphelion/session"
	"github.com/idolum-ai/aphelion/telegram"
)

const telegramThreadReminderManualMax = 3

func sendTelegramThreadReminders(ctx context.Context, sender commandSender, router commandThreadRouter, msg core.InboundMessage) (bool, error) {
	threads, err := router.TelegramThreads(msg.ChatID)
	if err != nil {
		return true, err
	}
	reminders, err := router.TelegramThreadReminders(msg.ChatID, "", 100)
	if err != nil {
		return true, err
	}
	selected := selectTelegramThreadReminderCandidates(threads, reminders, time.Now().UTC(), session.DefaultTelegramThreadReminderPolicy(), telegramThreadReminderManualMax)
	if len(selected) == 0 {
		return sendTelegramThreadText(ctx, sender, msg, "No stale side threads need reminders right now.")
	}
	sent := 0
	for _, thread := range selected {
		if _, err := sendTelegramThreadReminder(ctx, sender, router, msg.ChatID, msg.SenderID, thread); err != nil {
			return true, err
		}
		sent++
	}
	return sendTelegramThreadText(ctx, sender, msg, fmt.Sprintf("Sent %d stale-thread reminder%s.", sent, pluralSuffix(sent)))
}

func selectTelegramThreadReminderCandidates(threads []session.TelegramThread, reminders []session.TelegramThreadReminder, now time.Time, policy session.TelegramThreadReminderPolicy, limit int) []session.TelegramThread {
	if limit <= 0 {
		limit = telegramThreadReminderManualMax
	}
	latestByThread := map[int64]session.TelegramThreadReminder{}
	for _, reminder := range reminders {
		if reminder.ThreadID <= 0 {
			continue
		}
		current, ok := latestByThread[reminder.ThreadID]
		if !ok || reminder.UpdatedAt.After(current.UpdatedAt) || reminder.ID > current.ID && reminder.UpdatedAt.Equal(current.UpdatedAt) {
			latestByThread[reminder.ThreadID] = reminder
		}
	}
	selected := make([]session.TelegramThread, 0, limit)
	for _, thread := range threads {
		eligibility := thread.ReminderEligibility(now, policy)
		if !eligibility.Eligible {
			continue
		}
		if reminderSuppressesTelegramThreadReminder(thread, latestByThread[thread.ThreadID]) {
			continue
		}
		selected = append(selected, thread)
		if len(selected) >= limit {
			break
		}
	}
	return selected
}

func reminderSuppressesTelegramThreadReminder(thread session.TelegramThread, reminder session.TelegramThreadReminder) bool {
	if reminder.ID == 0 || reminder.ThreadID != thread.ThreadID {
		return false
	}
	lastActivity := telegramThreadLastActiveAt(thread)
	if lastActivity.IsZero() {
		return false
	}
	if reminder.SourceLastActivityAt.IsZero() || reminder.SourceLastActivityAt.Before(lastActivity) {
		return false
	}
	switch reminder.Status {
	case session.TelegramThreadReminderStatusPending, session.TelegramThreadReminderStatusIgnored, session.TelegramThreadReminderStatusResumed:
		return true
	default:
		return false
	}
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

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
