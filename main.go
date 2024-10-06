package main

import (
	"fmt"
	"runtime"
)

func main() {
	fmt.Printf("This is a publish-test app: %s %s\n", runtime.GOOS, runtime.GOARCH)
}
