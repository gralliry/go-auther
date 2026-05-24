package auther

// Adapter defines the persistence interface for Auther policies.
//
// Implementations must be safe for concurrent use. The Load method is called
// once during Authorizer construction; Save is called after every mutation
// (write-through pattern).
type Adapter interface {
	// Load returns the full policy snapshot from storage.
	// If no data exists yet, return nil for the snapshot with no error.
	Load() (*PolicySnapshot, error)

	// Save persists the full policy snapshot to storage.
	// Implementations should use atomic writes (e.g., temp file + rename)
	// to prevent data corruption.
	Save(snapshot *PolicySnapshot) error
}
