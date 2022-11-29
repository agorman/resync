package resync

import (
	"io"
	"os"
)

// Logger defines an interface for persisting log files. Each rsync command logs the processes
// output from both STDOUT and STDERR.
type Logger interface {
	// Rotates the log files and keeps old logs based on the configured retention value.
	Rotate(string) (io.WriteCloser, io.WriteCloser, error)

	// Stdout returns the last STDOUT from the rsync command.
	Stdout(string) (io.ReadCloser, error)

	// Stderr returns the last STDERR from the rsync command.
	Stderr(string) (io.ReadCloser, error)

	// Zip returns file that contains all of the stored logs for all syncs.
	Zip() (*os.File, error)
}
