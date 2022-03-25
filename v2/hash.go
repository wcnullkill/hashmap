package v2

import (
	"hash/maphash"
)

type Hash interface {
	Seed() maphash.Seed
	Hash(string) uint64
}

type mapHash struct {
	h *maphash.Hash
}

func newMapHash(seed maphash.Seed) *mapHash {
	mh := &mapHash{
		h: new(maphash.Hash),
	}
	mh.h.SetSeed(seed)
	return mh
}

func (hash *mapHash) Seed() maphash.Seed {
	return hash.h.Seed()
}

func (hash *mapHash) Hash(key string) uint64 {
	hash.h.WriteString(key)
	s := hash.h.Sum64()
	// 清空缓存
	hash.h.Reset()
	return s
}
