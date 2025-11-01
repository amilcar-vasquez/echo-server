// Filename: cmd/web/main.go

package main

import (
	"fmt"
)

func printWebSockets() string {
    return "Hello, RFC6455!"
}

func main() {
    greeting := printWebSockets()
    fmt.Println(greeting)
}
