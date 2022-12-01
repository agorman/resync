package resync

type Notifier interface {
	Notify(Stat) error
	NotifyHistory() error
}
