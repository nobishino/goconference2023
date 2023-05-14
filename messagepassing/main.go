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
	print(a, b)
	if r1 == 1 && r2 == 0 { // これは起こりうるか？
		panic("Answer: Yes")
	}
}

func exec() {
	a = 0
	b = 0
	wg.Add(2)
	defer wg.Wait()

	go f()
	go g()
}

func main() {
	for {
		exec()
	}
}
