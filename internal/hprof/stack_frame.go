package hprof

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	TagStackFrame    = 0x04
	TagStackTrace    = 0x05
	TagRootJavaFrame = 0x02
	TagRootJNILocal  = 0x03
	TagStartThread   = 0x0A
	TagEndThread     = 0x0B
)

type ThreadStack struct {
	ThreadID    int32
	StackFrames []StackFrame
}

func GetFrameData(file *os.File, frameID int64) ([]byte, error) {
	if file == nil {
		return nil, errors.New("File is not opened")
	}

	var frameSize int64 = 40 // 8 * 4 + 4 * 2
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

func ExtractCallStackRecords(file *os.File) ([]StackTrace, []StackFrame, []RootJavaFrame, []RootJNILocal, []StartThread, error) {
	var stackTraces []StackTrace
	var stackFrames []StackFrame
	var rootJavaFrames []RootJavaFrame
	var rootJNILocals []RootJNILocal
	var startThreads []StartThread

	for {
		record, err := readRecord(file)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, nil, nil, nil, err
		}

		switch record.Tag {
		case 0x01:
			_, err := readStringInUTF8(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
		case TagStackTrace:
			stackTrace, err := readStackTrace(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			stackTraces = append(stackTraces, stackTrace)
			for _, frameID := range stackTrace.FramesID {
				frameData, err := GetFrameData(file, frameID)
				if err != nil {
					return nil, nil, nil, nil, nil, err
				}
				frameRecord, err := readStackFrame(frameData)
				if err != nil {
					return nil, nil, nil, nil, nil, err
				}
				stackFrames = append(stackFrames, frameRecord)
			}

		case TagRootJavaFrame:
			rootJavaFrame, err := readRootJavaFrame(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			rootJavaFrames = append(rootJavaFrames, rootJavaFrame)

		case TagRootJNILocal:
			rootJNILocal, err := readRootJNILocal(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			rootJNILocals = append(rootJNILocals, rootJNILocal)

		case TagStartThread:
			startThread, err := readStartThread(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}
			startThreads = append(startThreads, startThread)

		case TagEndThread:
			_, err := readEndThread(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, err
			}

		default:
		}
	}

	return stackTraces, stackFrames, rootJavaFrames, rootJNILocals, startThreads, nil
}

func BuildThreadStacks(stackTraces []StackTrace, stackFrames []StackFrame) ([]ThreadStack, error) {
	fmt.Println("In function BuildThreadStacks")
	fmt.Printf("Number of stack traces: %d\n", len(stackTraces))
	fmt.Printf("Number of stack frames: %d\n", len(stackFrames))

	var threadStacks []ThreadStack
	for _, stackTrace := range stackTraces {
		fmt.Printf("Stack Trace for Thread %d\n", stackTrace.ThreadSerialNumber)
		var frames []StackFrame
		fmt.Printf("Processing Stack Trace %d\n", stackTrace.StackTraceSerialNumber)
		for _, frameID := range stackTrace.FramesID {
			fmt.Printf("Stack Trace Frame ID: %d\n", frameID)
			frameIndex := int(frameID) - 1
			if frameIndex >= 0 && frameIndex < len(stackFrames) {
				frames = append(frames, stackFrames[frameIndex])
			} else {
				fmt.Printf("Invalid frame ID %d\n", frameID)
			}
		}
		threadStacks = append(threadStacks, ThreadStack{
			ThreadID:    stackTrace.ThreadSerialNumber,
			StackFrames: frames,
		})
	}
	return threadStacks, nil
}
