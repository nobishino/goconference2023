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
	if r1 == 1 && r2 == 0 {
		panic("Answer: No!")
	}
}

func main() {
	wg.Add(2)
	defer wg.Wait()

	go f()
	go g()
}
