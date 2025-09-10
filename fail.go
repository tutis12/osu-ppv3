package main

import (
	"fmt"
	"os"
)

func Fail(cat string, id int, reason string) {
	fmt.Printf("fail: %s, %d\n", cat, id)
	file, err := os.Create(fmt.Sprintf("../%s/%d", cat, id))
	if err != nil {
		panic(err)
	}
	defer file.Close()
	_, err = file.Write([]byte(reason))
	if err != nil {
		panic(err)
	}
}
