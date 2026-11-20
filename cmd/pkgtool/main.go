// Command pkgtool generates packaging assets for the release workflow.
//
//	go run ./cmd/pkgtool icon-png <size> <out.png>
//	go run ./cmd/pkgtool icon-ico <out.ico>
package main

import (
	"fmt"
	"os"
	"strconv"

	"pc_monitor/pkg/icon"
)

func main() {
	if len(os.Args) < 3 {
		fail("usage: pkgtool icon-png <size> <out.png> | icon-ico <out.ico>")
	}
	switch os.Args[1] {
	case "icon-png":
		if len(os.Args) != 4 {
			fail("usage: pkgtool icon-png <size> <out.png>")
		}
		size, err := strconv.Atoi(os.Args[2])
		if err != nil || size < 16 || size > 1024 {
			fail("size must be 16-1024")
		}
		write(os.Args[3], icon.PNG(size))
	case "icon-ico":
		write(os.Args[2], icon.ICO())
	default:
		fail("unknown command " + os.Args[1])
	}
}

func write(path string, data []byte) {
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fail(err.Error())
	}
	fmt.Println("wrote", path)
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
