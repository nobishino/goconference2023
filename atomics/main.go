package main

import (
	"sync"
	"sync/atomic"
)

var a, b atomic.Int64
var wg sync.WaitGroup

func f() {
	defer wg.Done()
	a.Store(1)
	b.Store(2)
}

func g() {
	defer wg.Done()
	r1 := b.Load()
	r2 := a.Load()
	if r1 == 1 && r2 == 0 { // これは発生しうるか？
		panic("Answer: Yes") // 発生したらpanicしてプログラム終了
	}
}

// 実験を1回行う関数
func exec() {
	a.Store(0) // グローバル変数aの初期化
	b.Store(0) // グローバル変数bの初期化
	wg.Add(2)
	defer wg.Wait() // f, gの完了を待ち合わせる

	go f()
	go g()
}

func main() {
	for {
		exec()
	}
}
