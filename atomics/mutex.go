package main

import (
	"sync"
	"sync/atomic"
)

var x int64 = 0
var m sync.Mutex

// 単純なincrement. 並行プログラムでは意図通りに動かない
func increment() {
	x += 1
}

// Mutexを使ってgoroutine-safeにインクリメントする
func incrementByMutex() {
	m.Lock()
	x += 1
	m.Unlock()
}

// 同じことをatomicsで書く
func incrementByAtomics() {
	atomic.AddInt64(&x, 1)
}

// Go1.19からはatomics専用の型が使えて、よりコンパクトで間違えにくい書き方ができます
var y atomic.Int64

func betterIncrementByAtomics() {
	y.Add(1)
}
