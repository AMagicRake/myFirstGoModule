package main

import (
	"fmt"

	"github.com/AMagicRake/toolkit"
)

func main() {
	tk := &toolkit.Tools{}

	s := tk.RandomString(10)

	fmt.Println(s)
}
