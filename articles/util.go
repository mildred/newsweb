package articles

import (
	"bytes"
	"encoding/binary"
	"errors"
	"log"
)

func panicIfError(err error) {
	if err != nil {
		log.Panic(err)
	}
}

func btoi(b []byte) (i int64, err error) {
	err = binary.Read(bytes.NewReader(b), binary.BigEndian, &i)
	return
}

func itob(i int64) []byte {
	var buf = new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, i)
	panicIfError(err)
	return buf.Bytes()
}

func encodeIntKey(prefix string, num int64) []byte {
	return append([]byte(prefix), itob(num)...)
}

func encodeStrKey(prefix, data string) []byte {
	return []byte(prefix + data)
}

var ErrMismatchPrefix = errors.New("Mismatch prefix")

func decodeIntKey(prefix string, data []byte) (int64, error) {
	if !bytes.HasPrefix(data, []byte(prefix)) {
		return 0, ErrMismatchPrefix
	}
	return btoi(data[len([]byte(prefix)):])
}

func decodeStrKey(prefix string, data []byte) (string, error) {
	if !bytes.HasPrefix(data, []byte(prefix)) {
		return "", ErrMismatchPrefix
	}
	return string(data[len([]byte(prefix)):]), nil
}
