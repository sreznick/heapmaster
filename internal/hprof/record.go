package hprof

import (
	"bytes"
	"encoding/binary"
	"io"
	"time"
)

type Tag uint8

const (
	Utf8         Tag = 1
	TagLoadClass Tag = 2
)

type Record struct {
	Header     *Header
	Tag        Tag
	SinceStart time.Duration
	RecordSize uint32
}

type RecordUtf8 struct {
	Record *Record
	Id     uint64
	Value  string
}

type RecordLoadClass struct {
	Record           *Record
	ClassSerial      uint32
	ObjectId         uint64
	StackTraceSerial uint32
	NameId           uint64
}

func (r *Record) ReadId(rdr io.Reader) (uint64, error) {
	if r.Header.IdSize == 8 {
		var v uint64
		err := binary.Read(rdr, binary.BigEndian, &v)
		if err != nil {
			return 0, err
		}
		return v, nil
	}
	return 0, nil
}

func (r *RecordUtf8) Init(blob []byte) error {
	rdr := bytes.NewReader(blob)
	id, _ := r.Record.ReadId(rdr)
	r.Id = id
	r.Value = string(blob[8:])
	return nil
}

func (r *RecordLoadClass) Init(blob []byte) error {
	rdr := bytes.NewReader(blob)
	err := binary.Read(rdr, binary.BigEndian, &r.ClassSerial)
	if err != nil {
		return err
	}
	err = binary.Read(rdr, binary.BigEndian, &r.ObjectId)
	if err != nil {
		return err
	}
	err = binary.Read(rdr, binary.BigEndian, &r.StackTraceSerial)
	if err != nil {
		return err
	}
	err = binary.Read(rdr, binary.BigEndian, &r.NameId)
	if err != nil {
		return err
	}

	return nil
}

func ReadRecord(rdr io.Reader, header *Header) (*Record, []byte, error) {
	result := &Record{Header: header}

	var tag Tag
	err := binary.Read(rdr, binary.BigEndian, &tag)
	if err != nil {
		return nil, nil, err
	}
	result.Tag = tag

	var tsd uint32
	err = binary.Read(rdr, binary.BigEndian, &tsd)
	if err != nil {
		return nil, nil, err
	}
	result.SinceStart = time.Duration(tsd)

	var rSize uint32
	err = binary.Read(rdr, binary.BigEndian, &rSize)
	if err != nil {
		return nil, nil, err
	}
	result.RecordSize = rSize

	blob := make([]byte, rSize)
	_, err = io.ReadFull(rdr, blob)
	if err != nil {
		return nil, nil, err
	}
	return result, blob, nil
}
