package track

type Watcher interface {

	// Top-level path which we're monitoring
	Root() string

	// Channel of paths that have changed
	Changes() chan []string

	// Stop the watcher
	Stop()
}
