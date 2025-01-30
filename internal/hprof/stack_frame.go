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
	IsAlive     bool
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

func ExtractCallStackRecords(file *os.File) ([]StackTrace, []StackFrame, map[int32]bool, error) {
	IDtoStringInUTF8 := make(map[int64]string)
	stackTraces, stackFrames, _, _, startThreads, endThreads, err := ProcessRecords(file, IDtoStringInUTF8)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error processing records: %v", err)
	}

	// Thread status map
	threadStatus := make(map[int32]bool)
	for _, startThread := range startThreads {
		threadStatus[startThread.ThreadSerialNumber] = true
	}
	for _, endThread := range endThreads {
		threadStatus[endThread.ThreadSerialNumber] = false
	}

	return stackTraces, stackFrames, threadStatus, nil
}

func BuildThreadStacks(stackTraces []StackTrace, stackFrames []StackFrame, threadStatus map[int32]bool) ([]ThreadStack, error) {
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
			IsAlive:     threadStatus[stackTrace.ThreadSerialNumber], // Используем мапу
		})
	}

	return threadStacks, nil
}

// main from class.go
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
			fmt.Println("Reached end of file.")
			break
		} else if err != nil {
			return nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading record: %v", err)
		}

		switch record.Tag {
		case 0x01:
			stringInUTF8, err := readStringInUTF8(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading StringInUTF8: %v", err)
			}
			IDtoStringInUTF8[stringInUTF8.SerialNumber] = string(stringInUTF8.Bytes)

		case 0x02:
			loadClass, err := readLoadClass(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading LoadClass: %v", err)
			}
			className, ok := IDtoStringInUTF8[loadClass.ClassNameStringId]
			if !ok {
				className = "Unknown ClassName"
			}
			fmt.Printf("----LoadClass: ClassObjectId=%d, ClassName=%s\n", loadClass.ClassObjectId, className)

		case 0x04:
			stackFrame, err := readStackFrame(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading StackFrame: %v", err)
			}
			stackFrames = append(stackFrames, stackFrame)
			fmt.Printf("----StackFrame: MethodId=%d, Signature=%s\n", stackFrame.MethodId, IDtoStringInUTF8[stackFrame.MethodSignatureStringId])

		case 0x05:
			stackTrace, err := readStackTrace(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading StackTrace: %v", err)
			}
			stackTraces = append(stackTraces, stackTrace)
			fmt.Printf("----StackTrace: SerialNumber=%d\n", stackTrace.StackTraceSerialNumber)
			for _, frameID := range stackTrace.FramesID {
				fmt.Printf("--------FrameID: %d\n", frameID)
			}

		case 0x07:
			rootMonitor, err := readRootMonitorUsed(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading RootMonitorUsed: %v", err)
			}
			rootJavaFrames = append(rootJavaFrames, RootJavaFrame{
				ObjectId: rootMonitor.ObjectId,
			})
			fmt.Printf("----RootMonitorUsed: ObjectID=%d\n", rootMonitor.ObjectId)

		case TagStartThread:
			startThread, err := readStartThread(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading StartThread: %v", err)
			}
			startThreads = append(startThreads, startThread)
			fmt.Printf("----StartThread: ThreadSerial=%d\n", startThread.ThreadSerialNumber)

		case TagEndThread:
			endThread, err := readEndThread(record.Data)
			if err != nil {
				return nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading EndThread: %v", err)
			}
			endThreads = append(endThreads, endThread)
			fmt.Printf("----EndThread: ThreadSerial=%d\n", endThread.ThreadSerialNumber)

		default:
			fmt.Printf("Unknown tag encountered: 0x%x\n", record.Tag)
		}
	}

	return stackTraces, stackFrames, rootJavaFrames, rootJNILocals, startThreads, endThreads, nil
}
