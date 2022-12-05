package resync

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
)

// BoltDB is the default and only database for storing stats. In the future
// other databases could be added.
type BoltDB struct {
	config *Config
	mu     sync.Mutex
	db     *bolt.DB
}

// NewBoltDB creates the underlying boltdb database. If retention is less than 1 than
// the database isn't created and all calls are no-ops.
func NewBoltDB(config *Config) (*BoltDB, error) {
	if IntValue(config.Retention) < 1 {
		return &BoltDB{
			config: config,
		}, nil
	}

	if err := os.Mkdir(StringValue(config.LibPath), 0600); err != nil {
		if !os.IsExist(err) {
			return nil, fmt.Errorf("BoltDB: Failed to create resync lib directory: %w", err)
		}
	}

	db, err := bolt.Open(filepath.Join(StringValue(config.LibPath), "resync.db"), 0600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("BoltDB: Failed to open bolt database: %w", err)
	}

	boltdb := &BoltDB{
		config: config,
		mu:     sync.Mutex{},
		db:     db,
	}

	return boltdb, boltdb.Prune()
}

// Insert adds one Stat to bolt.
func (s *BoltDB) Insert(stat Stat) error {
	if IntValue(s.config.Retention) < 1 {
		return nil
	}

	// only one goroutine can do a read/write bold transaction at a time
	s.mu.Lock()
	defer s.mu.Unlock()

	err := s.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte(stat.Name))
		if err != nil {
			return fmt.Errorf("BoltDB: create bucket: %s", err)
		}

		// store stat in bolt JSON encoded
		encoded, err := json.Marshal(stat)
		if err != nil {
			return fmt.Errorf("BoltDB: marshal json: %s", err)
		}

		// store stat by sortable start time
		if err := b.Put([]byte(stat.start.Format(time.RFC3339Nano)), encoded); err != nil {
			return fmt.Errorf("BoltDB: put: %s", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("BoltDB: failed transaction: %s", err)
	}

	return s.prune()
}

// Prune removes all entries for a Sync job that exceed the rentention config.
func (s *BoltDB) Prune() error {
	// only one goroutine can do a read/write bold transaction at a time
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.prune()
}

func (s *BoltDB) prune() error {
	if IntValue(s.config.Retention) < 1 {
		return nil
	}

	err := s.db.Update(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			stats := b.Stats()
			count := stats.KeyN
			cursor := b.Cursor()
			cursor.First()

			for count > IntValue(s.config.Retention) {
				if err := cursor.Delete(); err != nil {
					return fmt.Errorf("BoltDB: failed delete: %s", err)
				}
				cursor.Next()
				count--
			}
			return nil
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("BoltDB: failed transaction: %s", err)
	}
	return nil
}

// List returns all stats stored as a map. The map keys are sync names and the values are a list of all stored stats for.
// that sysnc. Stats are returned storted by Start in descending order.
func (s *BoltDB) List() (map[string][]Stat, error) {
	statMap := make(map[string][]Stat)

	if IntValue(s.config.Retention) < 1 {
		return statMap, nil
	}

	err := s.db.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			b.ForEach(func(k, v []byte) error {
				stat := Stat{}
				if err := json.Unmarshal(v, &stat); err != nil {
					// stat can't be read so just skip it and log the error.
					// It will eventually be pruned.
					log.Error(err)
					return nil
				}

				statMap[string(name)] = append(statMap[string(name)], stat)

				return nil
			})
			return nil
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("BoltDB: failed transaction: %s", err)
	}

	// return sorted by start desc
	for _, stats := range statMap {
		sort.Slice(stats, func(i, j int) bool {
			return j < i
		})
	}

	return statMap, nil
}
