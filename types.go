package gosolarman

import (
    "bytes"
    "encoding/binary"
    "fmt"

    "github.com/grid-x/modbus"
)

// Header represents the header of a Solarman frame.
type Header struct {
    Start              byte   // Start byte of the frame, always 0xA5.
    Length             uint16 // Length of the payload in bytes.
    ControlCode        uint16 // Control code indicating the type of frame (e.g., REQUEST, RESPONSE).
    SequenceNumber     uint16 // Sequence number for request-response matching.
    LoggerSerialNumber uint32 // Serial number of the data logging stick.
}

// ResponsePayload represents the payload of a Solarman response frame.
type ResponsePayload struct {
    FrameType        byte                  // Frame type (e.g., 0x02 for solar inverter).
    Status           byte                  // Status of the request (e.g., 0x01 for real-time data).
    TotalWorkingTime uint32                // Total working time of the data logging stick in seconds.
    PowerOnTime      uint32                // Current uptime of the data logging stick in seconds.
    OffsetTime       uint32                // Offset timestamp in seconds.
    ModbusRTUFrame   modbus.ProtocolDataUnit // Modbus RTU response frame.
}

// Response represents a complete Solarman response, including the header, payload, and checksum.
type Response struct {
    Header   *Header         // Header of the response frame.
    Payload  *ResponsePayload // Payload of the response frame.
    Checksum byte            // Checksum for verifying the integrity of the frame.
}

// Parse parses a byte array into a Response structure.
// It validates the header, payload, and checksum of the frame.
//
// Parameters:
//   - data: The byte array representing the Solarman response frame.
//
// Returns:
//   - A pointer to the parsed Response structure.
//   - An error if the parsing fails (e.g., invalid length, checksum mismatch).
func Parse(data []byte) (*Response, error) {
    // Ensure the data is at least 11 bytes long
    if len(data) < 11 {
        return nil, fmt.Errorf("data too short, expected at least 11 bytes, got %d", len(data))
    }

    header, err := ParseHeader(data[:11])
    if err != nil {
        return nil, fmt.Errorf("failed to parse header: %w", err)
    }

    // Calculate the checksum of all bytes except the last one
    calculatedChecksum := CheckSum(data[1 : len(data)-2])

    // Compare the calculated checksum with the provided checksum (last byte)
    providedChecksum := data[len(data)-2]
    if calculatedChecksum != providedChecksum {
        return nil, fmt.Errorf("checksum mismatch: calculated 0x%02X, provided 0x%02X", calculatedChecksum, providedChecksum)
    }

    responseLength := uint16(len(data[11 : len(data)-2]))
    if header.Length != responseLength {
        return nil, fmt.Errorf("invalid Length, expected %d, got %d", header.Length, responseLength)
    }

    rtu := data[25 : len(data)-2]
    if len(rtu) < 5 {
        return nil, fmt.Errorf("invalid RTU frame, expected at least 5 bytes, got %d", len(rtu))
    }
    providedCrc := rtu[len(rtu)-2:]
    expectedCrc := CRCFromBytes(rtu[:len(rtu)-2])
    if !bytes.Equal(providedCrc, expectedCrc) {
        return nil, fmt.Errorf("CRC mismatch: expected %v, got %v", expectedCrc, providedCrc)
    }

    payload := &ResponsePayload{
        FrameType:        data[11],
        Status:           data[12],
        TotalWorkingTime: binary.LittleEndian.Uint32(data[13:17]),
        PowerOnTime:      binary.LittleEndian.Uint32(data[17:21]),
        OffsetTime:       binary.LittleEndian.Uint32(data[21:25]),
        ModbusRTUFrame:   parseRTUFrame(data[25 : len(data)-2]),
    }

    return &Response{
        Header:   header,
        Payload:  payload,
        Checksum: data[len(data)-1],
    }, nil
}

// ParseHeader parses the header of a Solarman frame from a byte array.
//
// Parameters:
//   - data: The byte array representing the header (must be at least 11 bytes).
//
// Returns:
//   - A pointer to the parsed Header structure.
//   - An error if the parsing fails (e.g., invalid start byte, insufficient length).
func ParseHeader(data []byte) (*Header, error) {
    // Ensure the data is at least 11 bytes long
    if len(data) < 11 {
        return nil, fmt.Errorf("data too short, expected at least 11 bytes, got %d", len(data))
    }

    reader := bytes.NewReader(data)

    // Parse Start
    start, err := reader.ReadByte()
    if err != nil {
        return nil, fmt.Errorf("failed to read Start: %w", err)
    }
    if start != StartByte {
        return nil, fmt.Errorf("invalid Start byte, expected 0x%02X, got 0x%02X", StartByte, start)
    }

    // Parse Length
    var length uint16
    if err := binary.Read(reader, binary.LittleEndian, &length); err != nil {
        return nil, fmt.Errorf("failed to read Length: %w", err)
    }

    // Parse Control Code
    var controlCode uint16
    if err := binary.Read(reader, binary.LittleEndian, &controlCode); err != nil {
        return nil, fmt.Errorf("failed to read Control Code: %w", err)
    }

    // Parse Sequence Number
    var sequenceNumber uint16
    if err := binary.Read(reader, binary.LittleEndian, &sequenceNumber); err != nil {
        return nil, fmt.Errorf("failed to read Sequence Number: %w", err)
    }

    // Parse Logger Serial Number
    var loggerSerialNumber uint32
    if err := binary.Read(reader, binary.LittleEndian, &loggerSerialNumber); err != nil {
        return nil, fmt.Errorf("failed to read Logger Serial Number: %w", err)
    }

    // Return the parsed header
    return &Header{
        Start:              start,
        Length:             length,
        ControlCode:        controlCode,
        SequenceNumber:     sequenceNumber,
        LoggerSerialNumber: loggerSerialNumber,
    }, nil
}

// parseRTUFrame parses the Modbus RTU frame from the payload.
//
// Parameters:
//   - data: The byte array representing the Modbus RTU frame.
//
// Returns:
//   - A ProtocolDataUnit containing the function code and data.
func parseRTUFrame(data []byte) modbus.ProtocolDataUnit {
    return modbus.ProtocolDataUnit{
        FunctionCode: data[1],
        Data:         data[2 : len(data)-2],
    }
}