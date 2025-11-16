package service

import (
	"github.com/quantarax/backend/daemon/manager"
	"time"
)

// RunCASGC is a stub wrapper that would be used when BoltCAS is wired into transport CAS backend.
// Currently, transport uses an in-memory CAS; this is provided for future integration.
func RunCASGC(retention time.Duration) error {
	// Example: cas, err := manager.OpenBoltCAS("/var/lib/quantarax/cas.db")
	// if err != nil { return err }
	// defer cas.Close()
	// _, err = cas.GC(retention)
	// return err
	_, _ = manager.OpenBoltCAS("")
	_ = retention
	return nil
}
