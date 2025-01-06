package hprof

import (
	"errors"
	"io"
	"os"
	"sync"
)

var (
	globalFile *os.File
	fileMutex  sync.Mutex
)

func initGlobalFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	globalFile = file
	return nil
}

func closeGlobalFile() error {
	if globalFile != nil {
		return globalFile.Close()
	}
	return nil
}

func getFrameData(frameID int64) ([]byte, error) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	if globalFile == nil {
		return nil, errors.New("Global file is not opened")
	}

	var offset int64 = frameID * 64
	_, err := globalFile.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// read data
	data := make([]byte, 64)
	_, err = globalFile.Read(data)
	if err != nil {
		if err == io.EOF {
			return nil, errors.New("Can't read data")
		}
		return nil, err
	}
	return data, nil
}

func extractCallStackRecords() ([]StackTrace, []StackFrame, []RootJavaFrame, []RootJNILocal, error) {
	var stackTraces []StackTrace
	var stackFrames []StackFrame
	var rootJavaFrames []RootJavaFrame
	var rootJNILocals []RootJNILocal
	// read dump
	for {
		record, err := readRecord(globalFile)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, nil, nil, err
		}

		// stack trace
		if record.Tag == 0x05 {
			stackTrace, err := readStackTrace(record.Data)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			stackTraces = append(stackTraces, stackTrace)

			for _, frameID := range stackTrace.FramesID {

				frameData, err := getFrameData(frameID)
				if err != nil {
					return nil, nil, nil, nil, err
				}
				frameRecord, err := readStackFrame(frameData)
				if err != nil {
					return nil, nil, nil, nil, err
				}
				stackFrames = append(stackFrames, frameRecord)
			}
		}

		// need to check root
		if record.Tag == 0x02 {
			rootJavaFrame, err := readRootJavaFrame(record.Data)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			rootJavaFrames = append(rootJavaFrames, rootJavaFrame)
		} else if record.Tag == 0x03 { // Тег для RootJNILocal
			rootJNILocal, err := readRootJNILocal(record.Data)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			rootJNILocals = append(rootJNILocals, rootJNILocal)
		}
	}

	return stackTraces, stackFrames, rootJavaFrames, rootJNILocals, nil
}
