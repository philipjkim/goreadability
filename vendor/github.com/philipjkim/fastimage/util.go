package fastimage

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"os"
)

func readUint16(buffer []byte) (result uint16) {
	reader := bytes.NewReader(buffer)
	binary.Read(reader, binary.BigEndian, &result)
	return
}

func readULint16(buffer []byte) (result uint16) {
	reader := bytes.NewReader(buffer)
	binary.Read(reader, binary.LittleEndian, &result)
	return
}

func readSampleFile(sf string) ([]string, error) {
	f, err := os.Open(sf)
	if err != nil {
		return nil, err
	}
	fs := bufio.NewScanner(f)
	var images []string
	for fs.Scan() {
		images = append(images, fs.Text())
	}

	return images, nil
}
