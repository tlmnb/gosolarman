package gosolarman

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestParseHeader(t *testing.T) {
	data := []byte{
		0xA5,       // Start
		0x0E, 0x00, // Length (14 bytes)
		0x10, 0x45, // Control Code
		0x01, 0x02, // Sequence Number
		0x12, 0x34, 0x56, 0x78, // Logger Serial Number
	}

	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("ParseHeader failed: %v", err)
	}

	if header.Start != 0xA5 {
		t.Errorf("Expected Start 0xA5, got 0x%02X", header.Start)
	}
	if header.Length != 14 {
		t.Errorf("Expected Length 14, got %d", header.Length)
	}
	if header.ControlCode != 0x4510 {
		t.Errorf("Expected ControlCode 0x4510, got 0x%04X", header.ControlCode)
	}
	if header.SequenceNumber != 0x0201 {
		t.Errorf("Expected SequenceNumber 0x0201, got 0x%04X", header.SequenceNumber)
	}
	if header.LoggerSerialNumber != 0x78563412 {
		t.Errorf("Expected LoggerSerialNumber 0x78563412, got 0x%08X", header.LoggerSerialNumber)
	}
}

func TestParse(t *testing.T) {
	data := []byte{
		0xA5,       // Start
		0x16, 0x00, // Length (22 bytes)
		0x10, 0x45, // Control Code)
		0x01, 0x02, // Sequence Number
		0x12, 0x34, 0x56, 0x78, // Logger Serial Number
		0x02,                   // Frame Type
		0x01,                   // Status
		0x10, 0x00, 0x00, 0x00, // Total Working Time
		0x20, 0x00, 0x00, 0x00, // Power On Time
		0x30, 0x00, 0x00, 0x00, // Offset Time
		0x01, 0x03, 0x02, 0x71, 0x00, 0x01, 0xd5, 0xa9, // Modbus RTU Frame
		0x00, // Checksum placeholder
		0x15, // End
	}
	// Calculate and set the checksum
	data[len(data)-2] = CheckSum(data[1 : len(data)-2])
	response, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if response.Header.Start != 0xA5 {
		t.Errorf("Expected Start 0xA5, got 0x%02X", response.Header.Start)
	}
	if response.Header.Length != 22 {
		t.Errorf("Expected Length 22, got %d", response.Header.Length)
	}
	if response.Header.ControlCode != 0x4510 {
		t.Errorf("Expected ControlCode 0x4510, got 0x%04X", response.Header.ControlCode)
	}
	if response.Payload.FrameType != 0x02 {
		t.Errorf("Expected FrameType 0x02, got 0x%02X", response.Payload.FrameType)
	}
	if response.Payload.Status != 0x01 {
		t.Errorf("Expected Status 0x01, got 0x%02X", response.Payload.Status)
	}
	if response.Payload.TotalWorkingTime != 0x10 {
		t.Errorf("Expected TotalWorkingTime 0x10, got 0x%08X", response.Payload.TotalWorkingTime)
	}
	if response.Payload.PowerOnTime != 0x20 {
		t.Errorf("Expected PowerOnTime 0x20, got 0x%08X", response.Payload.PowerOnTime)
	}
	if response.Payload.OffsetTime != 0x30 {
		t.Errorf("Expected OffsetTime 0x30, got 0x%08X", response.Payload.OffsetTime)
	}
}

func TestUint16ToBytes(t *testing.T) {
	value := uint16(0x1234)
	ret := uint16ToBytes(value, binary.LittleEndian)

	if !bytes.Equal(ret, []byte{0x34, 0x12}) {
		t.Errorf("Expected [0x34, 0x12], got %v", ret)
	}
}

func TestUint32ToBytes(t *testing.T) {
	value := uint32(0x12345678)
	ret := uint32ToBytes(value, binary.LittleEndian)

	if !bytes.Equal(ret, []byte{0x78, 0x56, 0x34, 0x12}) {
		t.Errorf("Expected [0x78, 0x56, 0x34, 0x12], got %v", ret)
	}
}
