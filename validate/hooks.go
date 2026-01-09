package validate

import "sync"

// Hooks configures validation callbacks.
type Hooks struct {
	OnError func(value any, err error)
}

var (
	hooksMu sync.RWMutex
	hooks   Hooks
)

// SetHooks replaces validation hooks.
func SetHooks(next Hooks) {
	hooksMu.Lock()
	hooks = next
	hooksMu.Unlock()
}

func notifyHooks(value any, err error) {
	hooksMu.RLock()
	current := hooks
	hooksMu.RUnlock()

	if current.OnError != nil {
		current.OnError(value, err)
	}
}
