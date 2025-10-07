package types

// SnapshotError describes the error that occurred while taking the snapshot.
type SnapshotError struct {
	Message string `json:"message,omitempty"`
}

// Equals checks if the snapshot error is identical to another snapshot error.
func (err SnapshotError) Equals(other SnapshotError) bool {
	return err.Message == other.Message
}
