package client

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
)

func GetPngMetadata(r io.Reader) (map[string]string, error) {
	header := make([]byte, 8)
	_, err := io.ReadFull(r, header)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(header, []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		return nil, errors.New("not a valid PNG file")
	}

	txtChunks := make(map[string]string)

	for {
		var length uint32
		err = binary.Read(r, binary.BigEndian, &length)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		chunkType := make([]byte, 4)
		_, err = io.ReadFull(r, chunkType)
		if err != nil {
			return nil, err
		}

		if string(chunkType) == "tEXt" {
			chunkData := make([]byte, length)
			_, err = io.ReadFull(r, chunkData)
			if err != nil {
				return nil, err
			}

			keywordEnd := bytes.IndexByte(chunkData, 0)
			if keywordEnd == -1 {
				return nil, errors.New("malformed tEXt chunk")
			}

			keyword := string(chunkData[:keywordEnd])
			contentJson := string(chunkData[keywordEnd+1:])
			txtChunks[keyword] = contentJson
		} else {
			// Skip the chunk data if it's not tEXt
			_, err = io.CopyN(io.Discard, r, int64(length))
			if err != nil {
				return nil, err
			}
		}

		// Skip the CRC
		_, err = io.CopyN(io.Discard, r, 4)
		if err != nil {
			return nil, err
		}
	}

	return txtChunks, nil
}
