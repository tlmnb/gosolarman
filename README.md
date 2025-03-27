# gosolarman
![go build](https://github.com/tlmnb/gosolarman/actions/workflows/go.yml/badge.svg)
## Introduction
`GoSolarman` is a Go library for interacting with Solarman devices using the Modbus RTU protocol by extending the [grid-x/modbus](https://github.com/grid-x/modbus) library. 
It provides functionality to encode and decode Modbus RTU frames, send requests, and validate responses.
This library is designed to simplify communication with Solarman data logging sticks.

It is based on the excellent python library [PySolarmanV5](https://github.com/jmccrohan/pysolarmanv5). The solarman protocol is described [here](https://pysolarmanv5.readthedocs.io/en/latest/solarmanv5_protocol.html).

## Usage
As stated before this library is an extension of the [grid-x/modbus](https://github.com/grid-x/modbus) library. Therefore the usage is similar

### Basic Usage
```golang
package main

import (
	"encoding/binary"
	"fmt"

	"github.com/tlmnb/gosolarman"
)

func main() {
  loggerSerial := uint32(1234567891)
  slaveId := byte(0x01)
	client := gosolarman.NewSolarmanClient("192.168.10.99:8899", loggerSerial, slaveId)

	data, err := client.ReadHoldingRegisters(625, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(binary.BigEndian.Uint16(data))
}
```

### Advanced Usage
```golang
package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"

	"github.com/grid-x/modbus"
	"github.com/tlmnb/gosolarman"
)

func main() {
  loggerSerial := uint32(1234567891)
	handler := gosolarman.NewSolarmanClientHandler("192.168.10.99:8899", loggerSerial)
	handler.SlaveID = 0x01
	handler.Logger = log.New(os.Stdout, "gosolarman: ", log.LstdFlags)
	err := handler.Connect()
	defer handler.Close()
	if err != nil {
		fmt.Println(err)
		return
	}
	client := modbus.NewClient(handler)
	data, err := client.ReadHoldingRegisters(625, 1)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(binary.BigEndian.Uint16(data))
}


```

### Contributing
Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

### License
This library is licensed under the Apache-2.0 License. See the [LICENSE](LICENSE) file for details.
