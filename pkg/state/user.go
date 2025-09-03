package state

import (
	"context"
)

const (
	CurrentUserId = "CurrentUserId"
	CurrentUserIP = "CurrentIP"
)

// CurrentUser returns the current user's ID as uint from the context.
func CurrentUser(ctx context.Context) uint {
	value := ctx.Value(CurrentUserId)
	if value == nil {
		return 0
	}

	userID, ok := value.(uint)
	if !ok {
		return 0
	}

	return userID
}

func SetCurrentUser(ctx context.Context, userID uint) context.Context {
	return context.WithValue(ctx, CurrentUserId, userID)
}
