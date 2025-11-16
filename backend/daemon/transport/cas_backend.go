package transport

type CASBackend interface {
	HasChunk(hash string) bool
	PutChunk(hash string, length int) error
}

var casBackend CASBackend

func SetCASBackend(b CASBackend) { casBackend = b }

func casHas(hash string) bool {
	if casBackend == nil {
		return false
	}
	return casBackend.HasChunk(hash)
}

func casPut(hash string, length int) {
	if casBackend == nil {
		return
	}
	_ = casBackend.PutChunk(hash, length)
}
