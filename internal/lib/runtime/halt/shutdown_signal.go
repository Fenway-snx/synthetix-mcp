package halt

import "sync"

func NewShutdownSignal(triggerFn func()) (<-chan struct{}, func()) {
	shutdownSignal := make(chan struct{}, 1)
	var shutdownOnce sync.Once

	requestShutdown := func() {
		shutdownOnce.Do(func() {
			if triggerFn != nil {
				triggerFn()
			}
			close(shutdownSignal)
		})
	}

	return shutdownSignal, requestShutdown
}
