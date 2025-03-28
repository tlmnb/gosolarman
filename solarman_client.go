package gosolarman

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"sync"
	"syscall"
	"time"

	"github.com/grid-x/modbus"
)

const (
	// StartByte is the start byte of a Solarman frame.
	StartByte = 0xA5

	// EndByte is the end byte of a Solarman frame.
	EndByte = 0x15

	// ControlCodeRequest is the control code for Modbus RTU requests in client mode.
	ControlCodeRequest = 0x4510

	// FrameType denotes the frame type for outgoing Modbus RTU requests (e.g., 0x02 for solar inverter).
	FrameType = 0x02

	// SensorType denotes the sensor type for outgoing requests (e.g., 0x0000).
	SensorType = 0x0000

	// TotalWorkingTime is the total working time of the data logging stick (set to 0x00000000 for outgoing requests).
	TotalWorkingTime = 0x00000000

	// PowerOnTime is the current uptime of the data logging stick (set to 0x00000000 for outgoing requests).
	PowerOnTime = 0x00000000

	// OffsetTime is the offset timestamp (set to 0x00000000 for outgoing requests).
	OffsetTime = 0x00000000

	Timeout = 5 * time.Second 
)

// SolarmanClientHandler is a handler that combines the Solarman packager and transporter.
type SolarmanClientHandler struct {
	solarmanPackager
	solarmanTransporter
}

// NewSolarmanClientHandler creates a new Solarman client handler.
//
// Parameters:
//   - Address: The address of the Solarman device (e.g., "192.168.1.1:8899").
//   - LoggerSerial: The serial number of the data logging stick.
//
// Returns:
//   - A pointer to the created SolarmanClientHandler.
func NewSolarmanClientHandler(Address string, LoggerSerial uint32) *SolarmanClientHandler {
	handler := &SolarmanClientHandler{}
	handler.Address = Address
	handler.LoggerSerial = LoggerSerial
	handler.Timeout = Timeout
	handler.ConnectDelay = 0
	return handler
}

// NewSolarmanClient creates a new Modbus client for Solarman devices.
//
// Parameters:
//   - Address: The address of the Solarman device (e.g., "192.168.1.1:8899").
//   - LoggerSerial: The serial number of the data logging stick.
//
// Returns:
//   - A Modbus client for interacting with the Solarman device.
func NewSolarmanClient(Address string, LoggerSerial uint32, SlaveID byte) modbus.Client {
	handler := NewSolarmanClientHandler(Address, LoggerSerial)
	handler.SlaveID = SlaveID
	return modbus.NewClient(handler)
}

// solarmanTransporter handles the transport layer for Solarman communication.
type solarmanTransporter struct {
	Address      string        // Address of the Solarman device.
	mu           sync.Mutex    // Mutex for thread-safe access to the connection.
	conn         net.Conn      // TCP connection to the Solarman device.
	Logger       modbus.Logger // Logger for debugging and monitoring.
	Timeout      time.Duration // Timeout for read/write operations.
	ConnectDelay time.Duration // Delay before attempting first access to the device.
}

// Send sends a Modbus RTU request and receives the response.
//
// Parameters:
//   - aduRequest: The Modbus RTU request to send.
//
// Returns:
//   - aduResponse: The Modbus RTU response received from the device.
//   - err: An error if the operation fails.
func (mb *solarmanTransporter) Send(aduRequest []byte) (aduResponse []byte, err error) {
	mb.mu.Lock()
	defer mb.mu.Unlock()
	if err = mb.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to %q: %w", mb.Address, err)
	}

	err = mb.write(aduRequest)

	if errors.Is(err, syscall.EPIPE) {
		if err = mb.reconnect(); err != nil {
			return nil, fmt.Errorf("failed to reconnect to %q: %w", mb.Address, err)
		}
		if err = mb.write(aduRequest); err != nil {
			return nil, fmt.Errorf("failed to write to %q: %w", mb.Address, err)
		}
	}
	mb.logf("SENT %s\n", hex.EncodeToString(aduRequest))
	aduResponse, err = mb.read()
	if errors.Is(err, syscall.EPIPE) {
		if err = mb.reconnect(); err != nil {
			return nil, fmt.Errorf("failed to reconnect to %q: %w", mb.Address, err)
		}
		if err = mb.write(aduRequest); err != nil {
			return nil, fmt.Errorf("failed to write to %q: %w", mb.Address, err)
		}
		if aduResponse, err = mb.read(); err != nil {
			return nil, fmt.Errorf("failed to read from %q: %w", mb.Address, err)
		}
	}
	mb.logf("RECD %s\n", hex.EncodeToString(aduResponse))
	return
}

// write sends a request to the Solarman device.
//
// Parameters:
//   - request: The byte array representing the request.
//
// Returns:
//   - An error if the operation fails.
func (mb *solarmanTransporter) write(request []byte) error {
	n, err := mb.conn.Write(request)
	if err != nil {
		return err
	}
	if n < len(request) {
		return fmt.Errorf("error while sending data. Got %d expected %d", n, len(request))
	}
	return nil
}

// read reads a response from the Solarman device.
//
// Returns:
//   - response: The byte array representing the response.
//   - err: An error if the operation fails.
func (mb *solarmanTransporter) read() (response []byte, err error) {
	raw_response := make([]byte, 1024)
	n, err := mb.conn.Read(raw_response)
	response = raw_response[0:n]
	return
}

// Connect establishes a connection to the Solarman device.
//
// Returns:
//   - An error if the connection fails.
func (mb *solarmanTransporter) Connect() error {
	mb.mu.Lock()
	defer mb.mu.Unlock()

	return mb.connect()
}

// connect establishes a TCP connection to the Solarman device.
//
// Returns:
//   - An error if the connection fails.
func (mb *solarmanTransporter) connect() error {
	fmt.Println("Connecting to", mb.Address)
	if mb.conn == nil {
		var conn net.Conn
		var err error
		d := net.Dialer{
			Timeout: mb.Timeout,
		}
		if conn, err = d.Dial("tcp", mb.Address); err != nil {
			return err
		}
		mb.conn = conn
	}
	time.Sleep(mb.ConnectDelay)
	return nil
}

// reconnect closes the existing connection and establishes a new one.
//
// Returns:
//   - An error if the reconnection fails.
func (mb *solarmanTransporter) reconnect() error {
	if mb.conn == nil {
		return mb.connect()
	} else {
		mb.conn.Close()

		mb.conn = nil
		return mb.connect()
	}
}

// Close closes the connection to the Solarman device.
//
// Returns:
//   - An error if the operation fails.
func (mb *solarmanTransporter) Close() (err error) {
	if mb.conn != nil {
		err = mb.conn.Close()
	}
	return
}

// logf logs a formatted message if a logger is configured.
//
// Parameters:
//   - format: The format string.
//   - v: The values to format.
func (mb *solarmanTransporter) logf(format string, v ...any) {
	if mb.Logger != nil {
		mb.Logger.Printf(format, v)
	}
}

// solarmanPackager handles the encoding and decoding of Modbus RTU frames for Solarman devices.
type solarmanPackager struct {
	SlaveID      byte   // The Modbus slave ID.
	LoggerSerial uint32 // The serial number of the data logging stick.
	serial       byte   // The sequence number for requests.
}

// NewSolarmanPackager creates a new Solarman packager.
//
// Parameters:
//   - LoggerSerial: The serial number of the data logging stick.
//
// Returns:
//   - A Modbus packager for Solarman devices.
func NewSolarmanPackager(LoggerSerial uint32) modbus.Packager {
	return &solarmanPackager{
		LoggerSerial: LoggerSerial,
		serial:       0x00,
	}
}

// SetSlave sets the Modbus slave ID.
//
// Parameters:
//   - slaveID: The Modbus slave ID to set.
func (mb *solarmanPackager) SetSlave(slaveID byte) {
	mb.SlaveID = slaveID
}

// Encode encodes a Modbus Protocol Data Unit (PDU) into an Application Data Unit (ADU).
//
// Parameters:
//   - pdu: The Modbus Protocol Data Unit to encode.
//
// Returns:
//   - adu: The encoded Application Data Unit.
//   - err: An error if the encoding fails.
func (mb *solarmanPackager) Encode(pdu *modbus.ProtocolDataUnit) (adu []byte, err error) {
	payload := new(bytes.Buffer)
	payload.WriteByte(FrameType)
	payload.Write(uint16ToBytes(SensorType, binary.LittleEndian))
	payload.Write(uint32ToBytes(TotalWorkingTime, binary.LittleEndian))
	payload.Write(uint32ToBytes(PowerOnTime, binary.LittleEndian))
	payload.Write(uint32ToBytes(OffsetTime, binary.LittleEndian))

	payload.WriteByte(mb.SlaveID)
	payload.WriteByte(pdu.FunctionCode)
	payload.Write(pdu.Data)
	payload.Write(CRC(mb.SlaveID, pdu))
	payloadBytes := payload.Bytes()

	request := new(bytes.Buffer)
	request.WriteByte(StartByte)
	request.Write(uint16ToBytes(uint16(len(payloadBytes)), binary.LittleEndian))
	request.Write(uint16ToBytes(ControlCodeRequest, binary.LittleEndian))
	request.WriteByte(mb.getNextSerial())
	request.WriteByte(0x00)
	request.Write(uint32ToBytes(mb.LoggerSerial, binary.LittleEndian))
	request.Write(payloadBytes)

	request.WriteByte(CheckSum(request.Bytes()[1:]))
	request.WriteByte(EndByte)

	return request.Bytes(), nil
}

// Decode decodes an Application Data Unit (ADU) into a Modbus Protocol Data Unit (PDU).
//
// Parameters:
//   - adu: The Application Data Unit to decode.
//
// Returns:
//   - pdu: The decoded Modbus Protocol Data Unit.
//   - err: An error if the decoding fails.
func (mb *solarmanPackager) Decode(adu []byte) (pdu *modbus.ProtocolDataUnit, err error) {
	response, err := Parse(adu)
	if err != nil {
		return nil, err
	}
	return &response.Payload.ModbusRTUFrame, nil
}

// Verify verifies that a Modbus RTU response matches the corresponding request.
//
// Parameters:
//   - aduRequest: The Modbus RTU request.
//   - aduResponse: The Modbus RTU response.
//
// Returns:
//   - err: An error if the verification fails.
func (mb *solarmanPackager) Verify(aduRequest []byte, aduResponse []byte) (err error) {
	// Ensure the response is at least the minimum length (header + checksum)
	if len(aduResponse) < 11 {
		return fmt.Errorf("response too short, expected at least 11 bytes, got %d", len(aduResponse))
	}

	// Verify the Start byte
	if aduResponse[0] != StartByte {
		return fmt.Errorf("invalid start byte, expected 0x%02X, got 0x%02X", StartByte, aduResponse[0])
	}

	// Verify the checksum
	calculatedChecksum := CheckSum(aduResponse[1 : len(aduResponse)-2]) // Exclude Start and End bytes
	providedChecksum := aduResponse[len(aduResponse)-2]
	if calculatedChecksum != providedChecksum {
		return fmt.Errorf("checksum mismatch: calculated 0x%02X, provided 0x%02X", calculatedChecksum, providedChecksum)
	}

	// Verify the sequence number
	requestSequence := aduRequest[5:6]   // Sequence number in the request
	responseSequence := aduResponse[5:6] // Sequence number in the response
	if !bytes.Equal(requestSequence, responseSequence) {
		return fmt.Errorf("sequence number mismatch: request 0x%04X, response 0x%04X",
			binary.BigEndian.Uint16(requestSequence), binary.BigEndian.Uint16(responseSequence))
	}

	// Verify the control code
	requestControlCode := binary.LittleEndian.Uint16(aduRequest[3:5])
	responseControlCode := binary.LittleEndian.Uint16(aduResponse[3:5])
	expectedResponseControlCode := requestControlCode - 0x3000 // Response code is request code - 0x3000
	if responseControlCode != expectedResponseControlCode {
		return fmt.Errorf("control code mismatch: expected 0x%04X, got 0x%04X",
			expectedResponseControlCode, responseControlCode)
	}

	// Verify the Logger Serial Number
	requestLoggerSerial := aduRequest[7:11]
	responseLoggerSerial := aduResponse[7:11]
	if !bytes.Equal(requestLoggerSerial, responseLoggerSerial) {
		return fmt.Errorf("logger serial number mismatch: request 0x%X, response 0x%X",
			requestLoggerSerial, responseLoggerSerial)
	}

	// If all checks pass, return nil
	return nil
}

// getNextSerial increments and returns the next sequence number for requests.
//
// Returns:
//   - The next sequence number.
func (mb *solarmanPackager) getNextSerial() byte {
	mb.serial++
	return mb.serial
}

// uint32ToBytes converts a uint32 value to a byte array.
//
// Parameters:
//   - i: The uint32 value to convert.
//   - byteorder: The byte order to use (e.g., binary.LittleEndian).
//
// Returns:
//   - A byte array representing the uint32 value.
func uint32ToBytes(i uint32, byteorder binary.ByteOrder) []byte {
	bytes := make([]byte, 4)
	byteorder.PutUint32(bytes, i)
	return bytes
}

// uint16ToBytes converts a uint16 value to a byte array.
//
// Parameters:
//   - i: The uint16 value to convert.
//   - byteorder: The byte order to use (e.g., binary.LittleEndian).
//
// Returns:
//   - A byte array representing the uint16 value.
func uint16ToBytes(i uint16, byteorder binary.ByteOrder) []byte {
	bytes := make([]byte, 2)
	byteorder.PutUint16(bytes, i)
	return bytes
}

// CheckSum calculates the checksum of a byte array.
//
// Parameters:
//   - b: The byte array to calculate the checksum for.
//
// Returns:
//   - The calculated checksum as a byte.
func CheckSum(b []byte) byte {
	checksum := byte(0)
	for _, v := range b {
		checksum += v & 0xFF
	}
	return checksum & 0xFF
}
