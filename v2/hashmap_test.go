package v2

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// 初始化HMap，cap为负数
func TestNewHMap1(t *testing.T) {
	assert := assert.New(t)
	table := []struct {
		cap   int
		panic bool
	}{
		{1, false},
		{0, false},
		{-1, true},
	}
	for _, v := range table {

		fc := func() {
			NewHMap(v.cap)
		}
		if v.panic {
			assert.Panics(fc, v.cap)
		}
	}
}

// 测试hmap.b
func TestNewHMap2(t *testing.T) {
	assert := assert.New(t)
	table := []struct {
		cap         int
		b           uint8
		bucketCount uint
	}{
		{0, 0, 1 << 0},
		{1 << 0, 0, 1 << 0},
		{1 << 1, 0, 1 << 0},
		{1<<2 - 1, 0, 1 << 0},
		{1<<4 - 1, 1, 1 << 1},
		{1<<10 - 1, 7, 1 << 7},
		{1 << 10, 7, 1 << 7},
		{1<<10 + 1, 8, 1 << 8},
	}
	for _, v := range table {
		m := NewHMap(v.cap)
		assert.Equal(v.b, m.b, v.cap)
		assert.Equal(v.bucketCount, m.bucketCount, v.cap)
	}
}

// 测试Set，Get各种类型
func TestSetGet1(t *testing.T) {
	type user struct {
		Name string
		Age  int
	}
	assert := assert.New(t)
	table := []struct {
		key string
		val interface{}
	}{
		{"a", 1},
		{"a1", 2},
		{"a2", 4},
		{"a3", -8},
		{"a4", "16"},
		{"a5", "32"},
		{"a6", 3.14},
		{"a7", -3.14},
		{"a8", true},
		{"a9", false},
		{"a10", 'a'},
		{"a11", '\r'},
		{"a12", "啊啊啊"},
		{"a13", []int{1, 2, 3}},
		{"a14", []string{"1", "2", "3"}},
		{"a15", map[string]int{"ab": 1, "bed": 2}},
		{"a16", map[string]string{"ab": "aaaa", "bed": "asdff"}},
		{"a17", user{"wc", 88}},
		{"a18", &user{"wc", 88}},
	}
	m := NewHMap(100)
	for _, v := range table {
		m.Set(v.key, v.val)
	}
	for _, v := range table {
		val, ok := m.Get(v.key)
		assert.True(ok, v.key)
		assert.Equal(v.val, val, v.key)
	}
}

// 测试 重复Set
func TestSetGet2(t *testing.T) {
	assert := assert.New(t)
	table := []struct {
		key    string
		val    interface{}
		exists bool
	}{
		{"a", 1, false},
		{"a", 2, true},
		{"a1", 3, false},
		{"a1", 4, false},
		{"a1", 5, true},
	}
	m := NewHMap(100)
	for _, v := range table {
		m.Set(v.key, v.val)
	}
	for _, v := range table {
		val, ok := m.Get(v.key)
		if v.exists {
			assert.True(ok, v.key)
			assert.Equal(v.val, val, v.key)
		}
	}
}

// 测试Count
func TestCount1(t *testing.T) {
	type user struct {
		Name string
		Age  int
	}
	assert := assert.New(t)
	table := []struct {
		key string
		val interface{}
	}{
		{"a", 1},
		{"a1", 2},
		{"a2", 4},
		{"a3", -8},
		{"a4", "16"},
		{"a5", "32"},
		{"a6", 3.14},
		{"a7", -3.14},
		{"a8", true},
		{"a9", false},
		{"a10", 'a'},
		{"a11", '\r'},
		{"a12", "啊啊啊"},
		{"a13", []int{1, 2, 3}},
		{"a14", []string{"1", "2", "3"}},
		{"a15", map[string]int{"ab": 1, "bed": 2}},
		{"a16", map[string]string{"ab": "aaaa", "bed": "asdff"}},
		{"a17", user{"wc", 88}},
		{"a18", &user{"wc", 88}},
	}
	m := NewHMap(100)
	for _, v := range table {
		m.Set(v.key, v.val)
	}
	assert.Equal(len(table), m.Count())
}

// 测试Del
func TestDel(t *testing.T) {
	type user struct {
		Name string
		Age  int
	}
	assert := assert.New(t)
	table := []struct {
		key    string
		val    interface{}
		delete bool
	}{
		{"a", 1, false},
		{"a1", 2, false},
		{"a2", 4, false},
		{"a3", -8, false},
		{"a4", "16", false},
		{"a5", "32", true},
		{"a6", 3.14, false},
		{"a7", -3.14, false},
		{"a8", true, true},
		{"a9", false, false},
		{"a10", 'a', false},
		{"a11", '\r', false},
		{"a12", "啊啊啊", false},
		{"a13", []int{1, 2, 3}, false},
		{"a14", []string{"1", "2", "3"}, true},
		{"a15", map[string]int{"ab": 1, "bed": 2}, true},
		{"a16", map[string]string{"ab": "aaaa", "bed": "asdff"}, false},
		{"a17", user{"wc", 88}, false},
		{"a18", &user{"wc", 88}, true},
	}
	m := NewHMap(100)
	for _, v := range table {
		m.Set(v.key, v.val)
	}
	for _, v := range table {
		if v.delete {
			m.Delete(v.key)
		}
	}
	for _, v := range table {
		val, ok := m.Get(v.key)
		if v.delete {
			assert.False(ok, v.key)
			assert.Nil(val, v.key)
		} else {
			assert.True(ok, v.key)
			assert.Equal(v.val, val, v.key)
		}
	}
}

// 测试溢出
func TestOverflow(t *testing.T) {
	assert := assert.New(t)
	m := NewHMap(1 << 4)
	count := 1 << 15
	for i := 0; i < count; i++ {
		m.Set(strconv.Itoa(i), i)
	}
	// 防止hash值相等，被覆盖
	if count == m.Count() {
		for i := 0; i < count; i++ {
			val, ok := m.Get(strconv.Itoa(i))
			assert.True(ok)
			assert.Equal(i, val, i)
		}
	}
}
