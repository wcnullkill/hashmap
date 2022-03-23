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
		{1, 0},
		{2, 1},
		{3, 2},
		{15, 4},
		{1<<10 - 1, 10},
		{1 << 10, 10},
		{1<<10 + 1, 11},
	}

}
