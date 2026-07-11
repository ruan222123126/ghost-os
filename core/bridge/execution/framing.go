package execution

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

const maxFrameBytes = 16 * 1024 * 1024

func writeFrame(writer io.Writer, payload any) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if len(jsonBytes) == 0 || len(jsonBytes) > maxFrameBytes {
		return fmt.Errorf("invalid frame payload size: %d", len(jsonBytes))
	}

	var lengthBuf [4]byte
	binary.BigEndian.PutUint32(lengthBuf[:], uint32(len(jsonBytes)))
	if err := writeAll(writer, lengthBuf[:]); err != nil {
		return err
	}
	return writeAll(writer, jsonBytes)
}

func readFrame(reader io.Reader, target any) error {
	var lengthBuf [4]byte
	if _, err := io.ReadFull(reader, lengthBuf[:]); err != nil {
		return err
	}

	length := binary.BigEndian.Uint32(lengthBuf[:])
	if length == 0 || length > maxFrameBytes {
		if looksLikeJSONPrefix(lengthBuf[:]) {
			return fmt.Errorf("%w: %q", errPersistentProtocolUnsupported, sanitizeFramePrefix(lengthBuf[:]))
		}
		return fmt.Errorf("invalid frame length: %d", length)
	}

	jsonBytes := make([]byte, length)
	if _, err := io.ReadFull(reader, jsonBytes); err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, target)
}

func writeAll(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		written, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrShortWrite
		}
		payload = payload[written:]
	}
	return nil
}

func looksLikeJSONPrefix(prefix []byte) bool {
	trimmed := bytes.TrimSpace(prefix)
	return len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[')
}

func sanitizeFramePrefix(prefix []byte) string {
	trimmed := bytes.TrimSpace(prefix)
	if len(trimmed) > 4 {
		trimmed = trimmed[:4]
	}
	return string(trimmed)
}
