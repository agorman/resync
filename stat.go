package resync

import "time"

// Stat defines basic statistics for a single sync. Stats are stored so that historical data from past syncs
// can be viewed.
type Stat struct {
	Name     string
	Success  bool
	Start    string
	End      string
	Duration time.Duration
	format   string
	start    time.Time
	end      time.Time
}

// Finish sets the Success based on err, End based on the current time, and Duration based on Start and End.
func (s Stat) Finish(err error) Stat {
	if err == nil {
		s.Success = true
	}

	s.end = time.Now()
	s.End = s.end.Format(s.format)
	s.Duration = time.Since(s.start)
	return s
}

// NewStat creates a new Stat with Name set to name, Success set to false, and Start set to
// the current time.
func NewStat(name string, format string) Stat {
	start := time.Now()

	return Stat{
		Name:    name,
		Success: false,
		Start:   start.Format(format),
		start:   start,
		format:  format,
	}
}
