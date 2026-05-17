//go:build linux

package runtime

import "github.com/idolum-ai/aphelion/principal"

const telegramMissionOwnerPrefix = "telegram:"

func (r *Runtime) IsTelegramAdmin(userID int64) bool {
	if r == nil || r.resolver == nil || userID <= 0 {
		return false
	}
	actor, ok := r.resolver.ResolveTelegramUser(userID)
	return ok && actor.Role == principal.RoleAdmin
}
