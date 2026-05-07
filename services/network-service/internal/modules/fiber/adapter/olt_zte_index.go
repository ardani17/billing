package adapter

import (
	"fmt"
)

// ZTEAddress merepresentasikan alamat fisik ZTE yang dipakai untuk SNMP index dan CLI.
type ZTEAddress struct {
	Shelf int
	Board int
	PON   int
	ONT   int
}

// zteDefaultAddressForPort mengubah port index API 0-based menjadi alamat ZTE C320 1-based.
func zteDefaultAddressForPort(portIndex int) (ZTEAddress, error) {
	if portIndex < 0 {
		return ZTEAddress{}, fmt.Errorf("zte port index tidak valid: %d", portIndex)
	}
	return ZTEAddress{
		Shelf: 1,
		Board: 1,
		PON:   portIndex + 1,
	}, nil
}

func zteOLTIndexForPort(portIndex int) (int, error) {
	addr, err := zteDefaultAddressForPort(portIndex)
	if err != nil {
		return 0, err
	}
	return zteCalculateOLTIndex(addr.Board, addr.PON), nil
}

func zteCLIInterfaceForPort(portIndex int) (string, error) {
	addr, err := zteDefaultAddressForPort(portIndex)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("gpon-olt_%d/%d", addr.Shelf, addr.PON), nil
}
