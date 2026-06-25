// gencrc reads a QRIS payload from stdin (with "XXXX" placeholder
// at the end for CRC), computes the correct CRC, and prints the
// fixed payload. Used for generating test data.
//
// Usage:
//   echo "00020101...6304XXXX" | go run cmd/gencrc/main.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 4096), 4096)

	if !scanner.Scan() {
		fmt.Fprintln(os.Stderr, "error: no input on stdin")
		os.Exit(1)
	}
	payload := strings.TrimSpace(scanner.Text())

	if !strings.HasSuffix(payload, "XXXX") {
		fmt.Fprintln(os.Stderr, "error: payload must end with XXXX placeholder")
		os.Exit(1)
	}

	prefix := payload[:len(payload)-4]
	crc := computeCRC([]byte(prefix))
	fmt.Println(prefix + fmt.Sprintf("%04X", crc))
}

func computeCRC(data []byte) uint16 {
	crc := uint16(0xFFFF)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
