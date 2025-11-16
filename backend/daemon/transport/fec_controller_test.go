package transport

import "testing"

func TestFECController_Tick_NoPanic(t *testing.T) {
	updates := 0
	ctl := NewFECController(32, 3, func(k, r int, reason string) { updates++ })
	for i := 0; i < 5; i++ {
		ctl.Tick()
	}
	_ = updates
}
