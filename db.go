package resync

// DB defines an interface for persisting Stats. Stats are simple metrics
// for each job.
type DB interface {
	// Prune removes old stats based on the configured retention value.
	Prune() error

	// List returns all stats stored.
	List() ([]Stat, error)

	// Insert adds a stat to the database.
	Insert(Stat) error
}
