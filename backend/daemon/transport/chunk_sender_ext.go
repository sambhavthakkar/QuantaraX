package transport

// SetScheduler configures the worker pool with a priority scheduler and class.
func (p *ChunkWorkerPool) SetScheduler(s *PriorityScheduler, class PriorityClass) {
	p.scheduler = s
	p.class = class
}
