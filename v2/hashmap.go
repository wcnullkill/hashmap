package v2

import (
	"hash/maphash"
)

type hmap struct {
	count           uint    // map内所有元素个数
	b               uint8   // 当前设置2^b为正常桶的个数
	bucketCount     uint    // 桶的数量
	buckets         []*bmap // 正常桶
	overflowBuckets []*bmap // 溢出桶
	// oldBuckets         []*bmap      // 正常桶，扩容时使用
	// oldOverflowBuckets []*bmap      // 溢出桶，扩容时使用
	cap       uint         // 初始化时，预设的map容量
	mapHash   Hash         //hash函数
	seed      maphash.Seed // 类似于hash0
	noverflow uint16       // 大致的溢出桶个数

}

func NewHMap(cap int) *hmap {
	if cap < 0 || cap > 1<<30 {
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
	if hm.testhashGrow() {
		hm.hashGrow()
	}

	hash := hm.mapHash.Hash(key)
	bucketIndex := calbucket(hash, hm.b)
	bm := hm.buckets[bucketIndex]
	// 先从正常桶和溢出桶查找
	// 如果找到了，就直接更新
	bucket, index, ok := bm.getIndex(key, hash)
	if ok {
		bucket.vals[index] = val
		return true
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
	pre := b
	overflow := b.overflow
	for overflow != nil {
		index, ok = bmapGetFree(overflow)
		if ok {
			overflow.update(index, key, val, hash)
			overflow.count++
			return true
		}
		pre = overflow
		overflow = overflow.overflow
	}

	// 如果溢出桶也满了，就创建一个新的溢出桶
	overflow = bmapInit()

	overflow.update(index, key, val, hash)
	overflow.count++
	pre.overflow = overflow
	hm.overflowBuckets = append(hm.overflowBuckets, overflow)
	hm.incrnoverflow()

	return true
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

// 等量扩容，一次性分配
// 正常桶容量不变，将原本正常桶及其溢出桶，重新插入新的正常桶+溢出桶中，使key，val更紧密
// 触发条件，溢出桶太多
func (hm *hmap) sameSizeGrow() {
	oldbuckets := hm.buckets
	B := hm.b
	hm.buckets = bmapSliceMake(B)
	hm.overflowBuckets = make([]*bmap, 0)
	hm.noverflow = 0
	// 原来的i->i
	for i := 0; i < len(oldbuckets); i++ {
		oldbm := oldbuckets[i]
		newbm := hm.buckets[i]
		for oldbm != nil {
			for j := uint8(0); j < uint8(8); j++ {
				if !bmapEmpty(oldbm, j) {
					// 尝试找本桶空闲
					index, ok := bmapGetFree(newbm)
					if ok {
						bmapcopy(oldbm, j, newbm, index)
						newbm.count++
						continue
					}
					// 如果本桶没有空闲，则找溢出桶
					for newbm.overflow != nil {
						index, ok = bmapGetFree(newbm.overflow)
						if ok {
							// 将旧bmap里的tophash,key,hash,val复制到新bmap里
							bmapcopy(oldbm, j, newbm, index)
							newbm.count++
							continue
						}
					}
					// 新bmap已满，创建新的溢出桶
					newbmap := bmapInit()
					newbm.overflow = newbmap
					hm.overflowBuckets = append(hm.overflowBuckets, newbmap)
					newbm = newbmap
					hm.incrnoverflow()
				}
			}
			oldbm = oldbm.overflow
		}
	}
}

// 翻倍扩容，一次性分配
// 将原本buckets[i]，分流到newBuckets[i]和newBuckets[i+(1<<hm.b)]上
// 触发条件，装载因子大于6.5
func (hm *hmap) grow() {
	oldbucktes := hm.buckets
	// oldoverflow = hm.oldOverflowBuckets
	B := hm.b + 1

	hm.buckets = bmapSliceMake(B)
	hm.overflowBuckets = make([]*bmap, 0)
	hm.noverflow = 0
	// 原来的i->{i,i+2^b}
	for i := 0; i < len(oldbucktes); i++ {
		oldbm := oldbucktes[i]
		newbm1 := hm.buckets[i]
		newbm2 := hm.buckets[i+(1<<hm.b)]
		for oldbm != nil {
			for j := uint8(0); j < 8; j++ {
				if !bmapEmpty(oldbm, j) {
					var newbm *bmap
					// 由倒数第B位取决分流
					// fmt.Printf("%b,%b\r\n", oldbm.keyhash[j], 1<<B)
					if oldbm.keyhash[j]&(1<<B) == 0 {
						newbm = newbm1
					} else {
						newbm = newbm2
					}

					// 尝试找本桶空闲
					index, ok := bmapGetFree(newbm)
					if ok {
						// 将旧bmap里的tophash,key,hash,val复制到新bmap里
						bmapcopy(oldbm, j, newbm, index)
						newbm.count++
						continue
					}
					// 如果本桶没有空闲，则找溢出桶
					for newbm.overflow != nil {
						index, ok = bmapGetFree(newbm.overflow)
						if ok {
							// 将旧bmap里的tophash,key,hash,val复制到新bmap里
							bmapcopy(oldbm, j, newbm, index)
							newbm.count++
							continue
						}
					}
					// 新bmap已满，创建新的溢出桶
					newbmap := bmapInit()
					newbm.overflow = newbmap
					hm.overflowBuckets = append(hm.overflowBuckets, newbmap)
					if oldbm.keyhash[j]&(1<<B) == 0 {
						newbm1 = newbmap
					} else {
						newbm2 = newbmap
					}
					hm.incrnoverflow()
				}
			}
			oldbm = oldbm.overflow
		}
	}
	hm.b++
	hm.bucketCount = 1 << hm.b
}

// noverflow直接+1
// golang中如果b<16，则noverflow++
// 如果>=16,则有1/(1<<(b-15))的概率+1
func (hm *hmap) incrnoverflow() {
	hm.noverflow++
}

// 是否满足扩容条件
func (hm *hmap) testhashGrow() bool {
	return overLoadFactor(hm.count+1, hm.bucketCount) || testTooManyBuckets(hm.noverflow, hm.b)
}
func (hm *hmap) hashGrow() {
	if overLoadFactor(hm.count+1, hm.bucketCount) {
		// 翻倍扩容
		hm.grow()

	} else {
		// 等量扩容
		hm.sameSizeGrow()
	}

}

// 是否满足翻倍扩容条件
// 如果装载因子超过6.5，则返回true
func overLoadFactor(count uint, bucketCount uint) bool {
	// bucketcount目前最大1<<30，暂时不考虑溢出
	return count > bucketCount*13/2
}

// 是否满足等量扩容条件
// 如果b>15，返回noverflow>=1<<15
// 如果b<=15，返回noverflow>=1<<b
func testTooManyBuckets(noverflow uint16, b uint8) bool {
	if b > 15 {
		b = 15
	}
	return noverflow >= uint16(1)<<(b&15)
}

func makemap(cap uint) *hmap {
	h := new(hmap)
	h.cap = cap
	B := uint8(0)
	for overloadFactor(cap, B) {
		B++
	}
	h.b = B
	if h.b > 0 {
		h.buckets = bmapSliceMake(B)
	}
	h.overflowBuckets = make([]*bmap, 0)
	h.bucketCount = 1 << B
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
	overflow := b.overflow
	for overflow != nil && overflow.count > 0 {
		index, ok := bmapSearch(overflow, key, hash)
		if ok {
			return overflow, index, true
		}
		overflow = overflow.overflow
	}
	return nil, uint8(0), false
}
func (bm *bmap) update(index uint8, key string, val interface{}, hash uint64) {
	bm.tophash[index] = calTopHash(hash)
	bm.keyhash[index] = hash
	bm.keys[index] = key
	bm.vals[index] = val
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
	for i := uint8(0); i < 8; i++ {
		if bmapEmpty(bm, i) {
			return i, true
		}
	}
	return uint8(0), false
}

// 判断index是否为空
func bmapEmpty(bm *bmap, index uint8) bool {
	return bm.tophash[index] == 0 && bm.keyhash[index] == 0
}

func bmapSliceMake(b uint8) []*bmap {
	s := make([]*bmap, 1<<b)
	for i := 0; i < 1<<b; i++ {
		s[i] = bmapInit()
	}
	return s
}

func bmapInit() *bmap {
	bm := new(bmap)
	bm.tophash = [8]uint8{}
	bm.keyhash = [8]uint64{}
	bm.keys = [8]string{}
	bm.vals = [8]interface{}{}
	return bm
}

func bmapcopy(src *bmap, srcIndex uint8, dst *bmap, dstIndex uint8) {
	dst.tophash[dstIndex] = src.tophash[srcIndex]
	dst.keyhash[dstIndex] = src.keyhash[srcIndex]
	dst.keys[dstIndex] = src.keys[srcIndex]
	dst.vals[dstIndex] = src.vals[srcIndex]
	// fmt.Println(dst.keys[dstIndex], src.keys[srcIndex])
}

// 计算cap是否大于2^b
func overloadFactor(cap uint, b uint8) bool {
	return cap > 1<<(b+3)
}

// 计算桶的位置，也就是hash值的低b位
func calbucket(hash uint64, b uint8) uint64 {
	return hash & (1<<(b) - 1)
}

// 计算tophash，也就是hash值的高八位
func calTopHash(hash uint64) uint8 {
	return uint8(hash >> 56)
}
