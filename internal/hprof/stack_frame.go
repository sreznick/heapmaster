package hprof

import (
	"errors"
	"io"
	"os"
)

func GetFrameData(file *os.File, frameID int64) ([]byte, error) {
	if file == nil {
		return nil, errors.New("File is not opened")
	}

	var frameSize int64 = 40
	var offset int64 = frameID * frameSize
	_, err := file.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Read data
	data := make([]byte, 64)
	_, err = file.Read(data)
	if err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func ExtractCallStackRecords(file *os.File) ([]StackTrace, []StackFrame, []RootJavaFrame, []RootJNILocal, error) {
	var stackTraces []StackTrace
	var stackFrames []StackFrame
	var rootJavaFrames []RootJavaFrame
	var rootJNILocals []RootJNILocal

	for {
		record, err := readRecord(file)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, nil, nil, err
		}

		// Stack trace with tag 0x05
		if record.Tag == 0x05 {
			stackTrace, err := readStackTrace(record.Data)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			stackTraces = append(stackTraces, stackTrace)

			for _, frameID := range stackTrace.FramesID {
				frameData, err := GetFrameData(file, frameID)
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

		// Root frames check
		if record.Tag == 0x02 {
			rootJavaFrame, err := readRootJavaFrame(record.Data)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			rootJavaFrames = append(rootJavaFrames, rootJavaFrame)
		} else if record.Tag == 0x03 {
			rootJNILocal, err := readRootJNILocal(record.Data)
			if err != nil {
				return nil, nil, nil, nil, err
			}
			rootJNILocals = append(rootJNILocals, rootJNILocal)
		}
	}

	return stackTraces, stackFrames, rootJavaFrames, rootJNILocals, nil
}
