package gosolarman

import "github.com/grid-x/modbus"

func CRC(SlaveID byte, pdu *modbus.ProtocolDataUnit) []byte {
	data := []byte{SlaveID, pdu.FunctionCode}
	data = append(data, pdu.Data...)

	return CRCFromBytes(data)
}

func CRCFromBytes(data []byte) []byte {
	var crc uint16 = 0xFFFF
	for _, v := range data {
		crc = crc ^ uint16(v)
		for range 8 {
			if (crc & 0x01) == 0 {
				crc = crc >> 1
			} else {
				crc = crc >> 1
				crc = crc ^ 0xA001
			}
		}
	}
	lo := byte(crc)
	hi := byte(crc >> 8)
	return []byte{lo, hi}
}
