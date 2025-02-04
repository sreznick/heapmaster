package hprof

import (
	"fmt"
	"io"
	"os"
)

const (
	TagStringInUTF8 Tag = 0x01
	// TagLoadClass       Tag = 0x02 // declared in record.go
	TagStackFrame      Tag = 0x04
	TagStackTrace      Tag = 0x05
	TagStartThread     Tag = 0x0A
	TagEndThread       Tag = 0x0B
	TagHeapDumpSegment Tag = 0x1C
	TagHeapDumpEnd     Tag = 0x2C
)

type HeapSubTag uint8

const (
	SubTagRootJavaFrame HeapSubTag = 0x03
	SubTagRootJNILocal  HeapSubTag = 0x02
)

func ProcessRecords(file *os.File, IDtoStringInUTF8 map[int64]string) ([]StackTrace, []StackFrame, []RootJavaFrame, []RootJNILocal, []StartThread, []EndThread, error) {
	var (
		stackTraces    []StackTrace
		stackFrames    []StackFrame
		rootJavaFrames []RootJavaFrame
		rootJNILocals  []RootJNILocal
		startThreads   []StartThread
		endThreads     []EndThread
	)

	for {
		record, err := readRecord(file)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading record: %v", err)
		}

		switch Tag(record.Tag) {
		case TagStringInUTF8:
			stringInUTF8, err := readStringInUTF8(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			IDtoStringInUTF8[stringInUTF8.SerialNumber] = string(stringInUTF8.Bytes)

		case TagLoadClass:
			if _, err := readLoadClass(record.Data); err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}

		case TagStackFrame:
			stackFrame, err := readStackFrame(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			stackFrames = append(stackFrames, stackFrame)

		case TagStackTrace:
			stackTrace, err := readStackTrace(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			stackTraces = append(stackTraces, stackTrace)

		case TagStartThread:
			startThread, err := readStartThread(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			startThreads = append(startThreads, startThread)

		case TagEndThread:
			endThread, err := readEndThread(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}
			endThreads = append(endThreads, endThread)

		case TagHeapDumpSegment:
			if err := processHeapDumpSegment(record.Data, &rootJavaFrames, &rootJNILocals); err != nil {
				return nil, nil, nil, nil, nil, nil, err
			}

		default:
			// Skip other tags
		}
	}

	return stackTraces, stackFrames, rootJavaFrames, rootJNILocals, startThreads, endThreads, nil
}

func processHeapDumpSegment(data []byte, rootJavaFrames *[]RootJavaFrame, rootJNILocals *[]RootJNILocal) error {
	offset := 0
	for offset < len(data) {
		subTag := HeapSubTag(data[offset])
		offset++

		switch subTag {
		case SubTagRootJavaFrame:
			frame, err := readRootJavaFrame(data[offset:])
			if err != nil {
				return err
			}
			*rootJavaFrames = append(*rootJavaFrames, frame)
			offset += 12

		case SubTagRootJNILocal:
			local, err := readRootJNILocal(data[offset:])
			if err != nil {
				return err
			}
			*rootJNILocals = append(*rootJNILocals, local)
			offset += 12

		default:
			offset++
		}
	}

	return nil
}
