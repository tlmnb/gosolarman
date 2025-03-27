package gosolarman

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/grid-x/modbus"
)

// Mock connection for testing
type mockConn struct {
	writeBuffer bytes.Buffer
	readBuffer  bytes.Buffer
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writeBuffer.Write(b)
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.readBuffer.Read(b)
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestSend(t *testing.T) {
	mock := &mockConn{}
	handler := &solarmanTransporter{
		Address: "192.168.1.1:8899",
		conn:    mock,
	}

	aduRequest := []byte{0xA5, 0x0E, 0x00, 0x10, 0x45, 0x01, 0x00, 0x4F, 0xAD, 0x6D, 0xA5}
	mock.readBuffer.Write([]byte{0xA5, 0x0E, 0x00, 0x10, 0x15, 0x01, 0x00, 0x4F, 0xAD, 0x6D, 0xA5, 0x00, 0x15})

	aduResponse, err := handler.Send(aduRequest)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if !bytes.Equal(aduRequest, mock.writeBuffer.Bytes()) {
		t.Errorf("Expected written data %X, got %X", aduRequest, mock.writeBuffer.Bytes())
	}

	expectedResponse := []byte{0xA5, 0x0E, 0x00, 0x10, 0x15, 0x01, 0x00, 0x4F, 0xAD, 0x6D, 0xA5, 0x00, 0x15}
	if !bytes.Equal(aduResponse, expectedResponse) {
		t.Errorf("Expected response %X, got %X", expectedResponse, aduResponse)
	}
}

func TestEncode(t *testing.T) {
	packager := &solarmanPackager{
		SlaveID:      0x01,
		LoggerSerial: 0x12345678,
	}

	pdu := &modbus.ProtocolDataUnit{
		FunctionCode: 0x03,
		Data:         []byte{0x00, 0x01, 0x00, 0x02},
	}

	adu, err := packager.Encode(pdu)
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}

	if adu[0] != StartByte {
		t.Errorf("Expected StartByte 0x%02X, got 0x%02X", StartByte, adu[0])
	}

	if adu[len(adu)-1] != EndByte {
		t.Errorf("Expected EndByte 0x%02X, got 0x%02X", EndByte, adu[len(adu)-1])
	}
}

func TestDecode(t *testing.T) {
	packager := &solarmanPackager{}
	adu := []byte{
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
		0xDB, // Checksum placeholder
		0x15, // End
	}

	pdu, err := packager.Decode(adu)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if pdu.FunctionCode != 0x03 {
		t.Errorf("Expected FunctionCode 0x03, got 0x%02X", pdu.FunctionCode)
	}
	expectedData := []byte{0x02, 0x71, 0x00, 0x01}
	if !bytes.Equal(pdu.Data, expectedData) {
		t.Errorf("Expected Data %X, got %X", expectedData, pdu.Data)
	}

}

func TestVerify(t *testing.T) {
	packager := &solarmanPackager{}
	aduRequest := []byte{0xA5, 0x0E, 0x00, 0x10, 0x45, 0x01, 0x00, 0x4F, 0xAD, 0x6D, 0xA5}
	aduResponse := []byte{0xA5, 0x0E, 0x00, 0x10, 0x15, 0x01, 0x00, 0x4F, 0xAD, 0x6D, 0xA5, 0x42, 0x15}

	err := packager.Verify(aduRequest, aduResponse)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
}

func TestCheckSum(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	expectedChecksum := byte(0x0A) // 0x01 + 0x02 + 0x03 + 0x04 = 0x0A

	calculatedChecksum := CheckSum(data)
	if calculatedChecksum != expectedChecksum {
		t.Errorf("Expected CheckSum 0x%02X, got 0x%02X", expectedChecksum, calculatedChecksum)
	}
}
