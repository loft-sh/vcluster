package snapshot

import "time"

type Snapshot struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

// SnapshotError describes the error that occurred while taking the snapshot.
type SnapshotError struct {
	Message string `json:"message,omitempty"`
}

func (err SnapshotError) Equals(other SnapshotError) bool {
	return err.Message == other.Message
}
