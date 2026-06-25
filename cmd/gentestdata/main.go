// gentestdata generates valid QRIS testdata files programmatically
// using auto-computed length and CRC.
package main

import (
	"fmt"
	"os"
	"strings"
)

type sample struct {
	filename string
	parts    []string // each part is already TLV-encoded
}

func main() {
	samples := []sample{
		{
			filename: "static_minimal.txt",
			parts: []string{
				tlv("00", "01"),
				tlv("01", "11"),
				tlv("51", tlv("00", "ID.CO.QRIS.WWW")+
					tlv("01", "9360091500000000000")+
					tlv("02", "ID1020000000000")+
					tlv("03", "UMI")),
				tlv("52", "4812"),
				tlv("53", "360"),
				tlv("58", "ID"),
				tlv("59", "MINIMAL TEST"),
				tlv("60", "JAKARTA"),
			},
		},
		{
			filename: "static_psp_only.txt",
			parts: []string{
				tlv("00", "01"),
				tlv("01", "11"),
				tlv("26", tlv("00", "ID.DANA.WWW")+
					tlv("01", "9360082500000000000")+
					tlv("02", "DANA0000000000000")+
					tlv("03", "UMI")),
				tlv("52", "5999"),
				tlv("53", "360"),
				tlv("58", "ID"),
				tlv("59", "DANA MERCHANT"),
				tlv("60", "JAKARTA"),
			},
		},
		{
			filename: "static_qris_only.txt",
			parts: []string{
				tlv("00", "01"),
				tlv("01", "11"),
				tlv("51", tlv("00", "ID.CO.QRIS.WWW")+
					tlv("01", "9360091500001234567")+
					tlv("02", "ID1020012345678")+
					tlv("03", "UMI")),
				tlv("52", "4812"),
				tlv("53", "360"),
				tlv("58", "ID"),
				tlv("59", "WARUNG SAMPLE"),
				tlv("60", "JAKARTA"),
			},
		},
		{
			filename: "dynamic_with_amount.txt",
			parts: []string{
				tlv("00", "01"),
				tlv("01", "12"), // dynamic
				tlv("51", tlv("00", "ID.CO.QRIS.WWW")+
					tlv("01", "9360091500001234567")+
					tlv("02", "ID1020012345678")+
					tlv("03", "URE")),
				tlv("52", "4812"),
				tlv("53", "360"),
				tlv("54", "100000"),
				tlv("58", "ID"),
				tlv("59", "DYNAMIC MERCH"),
				tlv("60", "JAKARTA"),
			},
		},
		{
			filename: "static_with_additional.txt",
			parts: []string{
				tlv("00", "01"),
				tlv("01", "11"),
				tlv("51", tlv("00", "ID.CO.QRIS.WWW")+
					tlv("01", "9360091500001234567")+
					tlv("02", "ID1020012345678")+
					tlv("03", "UMI")),
				tlv("52", "4812"),
				tlv("53", "360"),
				tlv("58", "ID"),
				tlv("59", "WITH ADDITIONAL"),
				tlv("60", "JAKARTA"),
				tlv("62", tlv("03", "STORE01")+
					tlv("07", "T-A01")),
			},
		},
	}

	for _, s := range samples {
		body := strings.Join(s.parts, "") + "6304"
		payload := body + fmt.Sprintf("%04X", computeCRC([]byte(body)))

		path := "testdata/valid/" + s.filename
		if err := os.WriteFile(path, []byte(payload), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "writing %s: %v\n", path, err)
			os.Exit(1)
		}
		fmt.Printf("✓ %s (%d bytes)\n", path, len(payload))
	}
}

func tlv(tag, value string) string {
	return fmt.Sprintf("%s%02d%s", tag, len(value), value)
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
