package resync

// DB defines an interface for persisting Stats. Stats are simple metrics
// for each job.
type DB interface {
	// Prune removes old stats based on the configured retention value.
	Prune() error

	// List returns all stats stored as a map. The map keys are sync names and the values are a list of all stored stats for.
	//that sysnc. Stats should be returned storted by Start in descending order.
	List() (map[string][]Stat, error)

	// Insert adds a stat to the database.
	Insert(Stat) error

	// Closes the connection the database
	Close() error
}
