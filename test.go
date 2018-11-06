package main

import "fmt"
import "math/rand"
import "time"

func main() {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 10; i++ {
		fmt.Println(r.Intn(2))
	}
}
