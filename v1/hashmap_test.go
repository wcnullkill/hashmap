package v1

import (
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

func TestNewHMap2(t *testing.T) {
	assert := assert.New(t)
	table := []struct {
		cap int
		b   uint8
	}{
		{0, 0},
		{1 << 0, 0},
		{1 << 1, 1},
		{1<<2 - 1, 2},
		{1<<4 - 1, 4},
		{1<<10 - 1, 10},
		{1 << 10, 10},
		{1<<10 + 1, 11},
	}
	for _, v := range table {
		m := NewHMap(v.cap)
		assert.Equal(v.b, m.b, v.cap)
	}
}

func TestSetGet(t *testing.T) {
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

func TestSetGet1(t *testing.T) {
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
