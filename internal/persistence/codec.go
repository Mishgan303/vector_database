package persistence

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
)

const formatVersion = 1

var crcTable = crc32.MakeTable(crc32.Castagnoli)

type envelope struct {
	Version int             `json:"version"`
	Data    json.RawMessage `json:"data"`
	CRC32C  uint32          `json:"crc32c"`
}

func encode(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envelope{Version: formatVersion, Data: data, CRC32C: crc32.Checksum(data, crcTable)})
}

func decode(data []byte, value any) error {
	var record envelope
	if err := json.Unmarshal(data, &record); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if record.Version != formatVersion {
		return fmt.Errorf("unsupported format version %d", record.Version)
	}
	if crc32.Checksum(record.Data, crcTable) != record.CRC32C {
		return fmt.Errorf("CRC32C mismatch")
	}
	if err := json.Unmarshal(record.Data, value); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	return nil
}
