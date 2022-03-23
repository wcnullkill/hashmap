package v1

import "hash/maphash"

type hmap struct {
	count        uint // map内所有元素个数
	buckets      []*bmap
	b            uint8 //
	buckestCount uint  // 桶的数量
	cap          uint  // 初始化时，预设的map容量
	mapHash      Hash
	seed         maphash.Seed // 类似于hash0
}

func NewHMap(cap int) *hmap {
	if cap < 0 {
		panic("cap error")
	}
	return makemap(uint(cap))
}

func (hm *hmap) Set(key string, val interface{}) {
	if hm.set(key, val) {
		hm.count++
	}
}
func (hm *hmap) Get(key string) (interface{}, bool) {
	return hm.get(key)
}
func (hm *hmap) Delete(key string) {
	if hm.del(key) {
		hm.count--
	}
}
func (hm *hmap) Count() int {
	return int(hm.count)
}

// 返回值表示是否属于新增
func (hm *hmap) set(key string, val interface{}) bool {
	hash := hm.mapHash.Hash(key)
	bucketIndex := calbucket(hash, hm.b)
	bucket := hm.buckets[bucketIndex]
	return bucket.set(key, val, hash)
}
func (hm *hmap) get(key string) (interface{}, bool) {
	hash := hm.mapHash.Hash(key)
	bucketIndex := calbucket(hash, hm.b)
	bucket := hm.buckets[bucketIndex]
	return bucket.get(key, hash)
}
func (hm *hmap) del(key string) bool {
	hash := hm.mapHash.Hash(key)
	bucketIndex := calbucket(hash, hm.b)
	bucket := hm.buckets[bucketIndex]
	return bucket.del(key, hash)
}

func makemap(cap uint) *hmap {
	h := new(hmap)
	h.cap = cap
	B := uint8(0)
	for !overloadFactor(cap, B) {
		B++
	}
	h.b = B
	h.buckestCount = 1 << B
	h.buckets = make([]*bmap, h.buckestCount)
	h.seed = maphash.MakeSeed()
	hash := newMapHash(h.seed)
	h.mapHash = hash
	return h
}

type bmap struct {
	count    uint8 // 有效元素个数
	tophash  [8]uint8
	keyhash  [8]uint64
	keys     [8]string
	vals     [8]interface{}
	overflow *bmap
}

func (bm *bmap) get(key string, hash uint64) (interface{}, bool) {
	b, index, ok := bm.getIndex(key, hash)
	if ok {
		return b.vals[index], true
	}
	return nil, false
}

// 返回值表示是否属于新增
func (bm *bmap) set(key string, val interface{}, hash uint64) bool {
	// 首先尝试搜索
	bucket, index, ok := bm.getIndex(key, hash)
	if ok {
		bucket.vals[index] = val
		return false
	}

	// 没有找到，将值插入一个空闲处
	// 先从正常桶插入
	b := bm
	index, ok = bmapGetFree(b)
	if ok {
		b.update(index, key, val, hash)
		b.count++
		return true
	}
	// 再找溢出桶
	b = bm.overflow
	for b != nil {
		index, ok = bmapGetFree(b)
		if ok {
			b.update(index, key, val, hash)
			b.count++
			return true
		}
	}
	// 如果溢出桶也满了，就创建一个新的溢出桶
	overflow := new(bmap)
	b.overflow = overflow

	overflow.update(index, key, val, hash)
	overflow.count++
	return true
}

func (bm *bmap) del(key string, hash uint64) bool {
	b, index, ok := bm.getIndex(key, hash)
	if ok {
		b.update(index, "", nil, 0)
		b.count--
		return true
	}
	return false
}

// 通过遍历正常桶与溢出桶查找
func (bm *bmap) getIndex(key string, hash uint64) (*bmap, uint8, bool) {
	b := bm
	// 从正常桶搜索
	index, ok := bmapSearch(bm, key, hash)
	if ok {
		return b, index, true
	}
	// 从溢出桶搜索
	b = b.overflow
	for b != nil && b.count > 0 {
		index, ok := bmapSearch(bm, key, hash)
		if ok {
			return b, index, true
		}
	}
	return nil, uint8(0), false
}

// 从桶中查询
func bmapSearch(bm *bmap, key string, hash uint64) (uint8, bool) {
	tophash := calTopHash(hash)
	for i := 0; i < 8; i++ {
		if tophash == bm.tophash[i] && hash == bm.keyhash[i] && key == bm.keys[i] {
			return uint8(i), true
		}
	}
	return uint8(0), false
}

// 从桶中找一个空闲处
func bmapGetFree(bm *bmap) (uint8, bool) {
	if bm.count == 8 {
		return uint8(0), false
	}
	for i := 0; i < 8; i++ {
		if bm.keyhash[i] == 0 {
			return uint8(i), true
		}
	}
	return uint8(0), false
}

func (bm *bmap) update(index uint8, key string, val interface{}, hash uint64) {
	bm.tophash[index] = calTopHash(hash)
	bm.keyhash[index] = hash
	bm.keys[index] = key
	bm.vals[index] = val
	bm.count++
}

// 计算cap是否小于等于2^b
func overloadFactor(cap uint, b uint8) bool {
	return cap <= 1<<b
}

// 计算桶的位置，也就是hash值的低b位
func calbucket(hash uint64, b uint8) uint64 {
	return hash & (1<<(b+1) - 1)
}

// 计算tophash，也就是hash值的高八位
func calTopHash(hash uint64) uint8 {
	return uint8(hash >> 46)
}
