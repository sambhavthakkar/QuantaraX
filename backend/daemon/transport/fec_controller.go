package transport

import (
	"time"
)

type LossEstimator struct {
	windowSent int64
	windowLost int64
	lastUpdate time.Time
}

func (le *LossEstimator) OnSent(n int64)   { le.windowSent += n }
func (le *LossEstimator) OnLost(n int64)   { le.windowLost += n }
func (le *LossEstimator) Estimate() float64 {
	if le.windowSent == 0 { return 0 }
	return float64(le.windowLost) / float64(le.windowSent)
}

// FECController adapts K/R based on loss

type FECController struct {
	k, r   int
	loss   *LossEstimator
	lastMsg time.Time
	update func(k, r int, reason string)
}

func NewFECController(initK, initR int, update func(k, r int, reason string)) *FECController {
	return &FECController{k: initK, r: initR, loss: &LossEstimator{}, update: update}
}

func (fc *FECController) Tick() {
	loss := fc.loss.Estimate()
	if loss > 0.10 && fc.r < 12 {
		fc.r += 2
		fc.update(fc.k, fc.r, "loss>10%")
	} else if loss > 0.03 && fc.r < 8 {
		fc.r += 1
		fc.update(fc.k, fc.r, "loss>3%")
	} else if loss < 0.01 && fc.r > 2 {
		fc.r -= 1
		fc.update(fc.k, fc.r, "loss<1%")
	}
}
