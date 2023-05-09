package main

import "sync"

var a = 0

func main() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		a = 1
		wg.Done()
	}()
	go func() {
		a = 2
		wg.Done()
	}()
	wg.Wait()
	print(a)
}
