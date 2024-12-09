package hprof

import (
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

var hprofMark = "JAVA PROFILE 1.0.2"

type Header struct {
	IdSize uint32
	TimeStamp time.Time
}

func IsHprofStart(data []byte) bool {
	return len(data) >= 19 && string(data[:18]) == "JAVA PROFILE 1.0.2" && data[18] == 0
}

func ReadHeader(rdr io.Reader) (*Header, error) {
        b1 := make([]byte, 19)
        _, err := io.ReadFull(rdr, b1)
        if err != nil {
                return nil, err
        }
	if !IsHprofStart(b1) {
		return nil, fmt.Errorf("it is not hprof dump")
	}

        var idSize uint32
        err = binary.Read(rdr, binary.BigEndian, &idSize)
        if err != nil {
                return nil, err
        }

        var ts int64
        err = binary.Read(rdr, binary.BigEndian, &ts)
        if err != nil {
		return nil, err
        }

	return &Header{
		IdSize: idSize,
		TimeStamp: time.Unix(ts/1000, ts%1000),	
	}, nil
}

