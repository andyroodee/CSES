package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func readInt64(reader *bufio.Reader) int64 {
	var n int64
	in, _ := reader.ReadString('\n')
	n, _ = strconv.ParseInt(strings.TrimSpace(in), 10, 64)
	return n
}

func readInt64s(reader *bufio.Reader) []int64 {
	var numbers []int64
	in, _ := reader.ReadString('\n')
	for _, s := range strings.Split(strings.TrimSpace(in), " ") {
		num, _ := strconv.ParseInt(s, 10, 64)
		numbers = append(numbers, num)
	}
	return numbers
}

func main() {
	reader := bufio.NewReader(os.Stdin)
	n := readInt64(reader)

	expected := (n * (n + 1)) / 2
	numbers := readInt64s(reader)
	for _, num := range numbers {
		expected -= num
	}
	fmt.Println(expected)
}
