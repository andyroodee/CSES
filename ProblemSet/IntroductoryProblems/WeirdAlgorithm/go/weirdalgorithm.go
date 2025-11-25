package main

import (
	"fmt"
)

func main() {
	var n int64
	_, err := fmt.Scanf("%d", &n)
	if err != nil {
		return
	}

	for n != 1 {
		fmt.Print(n, " ")
		if n%2 == 0 {
			n /= 2
		} else {
			n = 3*n + 1
		}
	}
	fmt.Println("1 ")
}
