package main

import "sync"

var a, b int
var wg sync.WaitGroup

func f() {
	defer wg.Done()
	a = 1
	b = 1
}

func g() {
	defer wg.Done()
	r1 := b
	r2 := a
	if r1 == 1 && r2 == 0 { // これは発生しうるか？
		panic("Answer: Yes") // 発生したらpanicしてプログラム終了
	}
}

// 実験を1回行う関数
func exec() {
	a = 0 // グローバル変数aの初期化
	b = 0 // グローバル変数bの初期化
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
