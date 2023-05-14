package main

import (
	"fmt"
	"sync"
	"time"
)

var a = 0

func exec() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		a = 1
		wg.Done()
	}()
	go func() {
		a = 2
		time.Sleep(1 * time.Millisecond) // ※入れないと再現しづらいので入れておく
		recorder.Store(a, struct{}{})
		wg.Done()
	}()
	wg.Wait()
}

var recorder sync.Map

func main() {
	const TRIAL = 10000
	for i := 0; i < TRIAL; i++ {
		exec()
	}
	// a = 1, a = 2は起きるがa = 0は起きないことを確認する
	_, zero := recorder.Load(0)
	_, one := recorder.Load(1)
	_, two := recorder.Load(2)
	if (zero) || !one || !two {
		panic(fmt.Sprintf("zero = %t, one = %t, two = %t", zero, one, two))
	}
	fmt.Println("OK, confirmed")
}
