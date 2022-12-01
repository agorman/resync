package resync

// Notifier defines an interface for sending notifications.
type Notifier interface {
	// Notify sends a notification about a single stat.
	Notify(Stat) error

	// Notify History sends a notification with the sync history.
	NotifyHistory() error
}
