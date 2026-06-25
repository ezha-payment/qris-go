package qris

import "fmt"

// computeCRC calculates CRC-16/CCITT-FALSE checksum.
//
// Algorithm parameters per ITU-T V.41 / CCITT specification:
//   - Polynomial: 0x1021
//   - Initial value: 0xFFFF
//   - No input reflection
//   - No output reflection
//   - No XOR output
//
// Per QRIS spec, CRC is computed over the entire payload INCLUDING
// the CRC tag header "6304" but EXCLUDING the CRC value itself.
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

// formatCRC returns CRC as 4-character uppercase hexadecimal string.
func formatCRC(crc uint16) string {
	return fmt.Sprintf("%04X", crc)
}
