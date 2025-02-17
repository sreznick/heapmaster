package hprof

import (
	"bytes"
	"encoding/binary"
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

func ProcessRecords(file *os.File, IDtoStringInUTF8 map[ID]string) ([]StackTrace, []StackFrame, map[int32]ID, []StartThread, []EndThread, []RootJNILocal, []RootNativeStack, error) {
	var (
		stackTraces      []StackTrace
		stackFrames      []StackFrame
		rootJavaFrames   []RootJavaFrame
		rootJNIGlobals   []RootJNIGlobal
		rootJNILocals    []RootJNILocal
		startThreads     []StartThread
		endThreads       []EndThread
		rootNativeStacks []RootNativeStack
	)

	var ClassSerialToNameId = make(map[int32]ID)

	subTagFuncMap := map[HeapDumpSubTag]func(*bytes.Reader) interface{}{
		RootUnknownTag:        readRootUnknown,
		RootJNIGlobalTag:      readRootJNIGlobal,
		RootJNILocalTag:       readRootJNILocal,
		RootJavaFrameTag:      readRootJavaFrame,
		RootNativeStackTag:    readRootNativeStack,
		RootStickyClassTag:    readRootStickyClass,
		RootThreadBlockTag:    readRootThreadBlock,
		RootMonitorUsedTag:    readRootMonitorUsed,
		RootThreadObjectTag:   readRootThreadObject,
		ClassDumpTag:          readClassDump,
		InstanceDumpTag:       readInstanceDump,
		ObjectArrayDumpTag:    readObjectArrayDump,
		PrimitiveArrayDumpTag: readPrimitiveArrayDump,
	}

	for {
		record, err := readRecord(file)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("error reading record: %v", err)
		}

		switch Tag(record.Tag) {
		case TagStringInUTF8:
			stringInUTF8, ok := readStringInUTF8(record.Data).(StringInUTF8)
			if !ok {
				return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("incorrect StringInUTF8 format")
			}
			IDtoStringInUTF8[ID(stringInUTF8.StringId)] = string(stringInUTF8.Bytes)

		case LoadClassTag:
			loadClass, ok := readLoadClass(record.Data).(LoadClass)
			if !ok {
				return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("incorrect LoadClass format")
			}
			ClassSerialToNameId[loadClass.ClassSerialNumber] = loadClass.ClassNameStringId

		case TagStackFrame:
			stackFrame, ok := readStackFrame(record.Data).(StackFrame)
			if !ok {
				return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("incorrect StackFrame format")
			}
			stackFrames = append(stackFrames, stackFrame)

		case TagStackTrace:
			stackTrace, ok := readStackTrace(record.Data).(StackTrace)
			if !ok {
				return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("incorrect StackTrace format")
			}
			stackTraces = append(stackTraces, stackTrace)

		case HeapDumpTag, HeapDumpSegmentTag:
			heapDump := readHeapDump(record.Data)
			reader := bytes.NewReader(heapDump.data)

			for {
				var subTag HeapDumpSubTag
				err := binary.Read(reader, binary.BigEndian, &subTag)
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Printf("Error while reading subtag: %v\n", err)
					break
				}
				if readerFunction, ok := subTagFuncMap[subTag]; ok {
					switch subTag {
					case RootJNIGlobalTag:
						result := readerFunction(reader)
						if global, valid := result.(RootJNIGlobal); valid {
							rootJNIGlobals = append(rootJNIGlobals, global)
						}

					case RootJNILocalTag:
						result := readerFunction(reader)
						if local, valid := result.(RootJNILocal); valid {
							rootJNILocals = append(rootJNILocals, local)
						}

					case RootNativeStackTag:
						result := readerFunction(reader)
						if stack, valid := result.(RootNativeStack); valid {
							rootNativeStacks = append(rootNativeStacks, stack)
						}
					default:
						_ = readerFunction(reader)
					}
				} else {
					fmt.Printf("Undefined subtag: %d\n", subTag)
					break
				}
			}

		case StartThreadTag:
			startThread, ok := readStartThread(record.Data).(StartThread)
			if !ok {
				return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("incorrect StartThread format")
			}
			startThreads = append(startThreads, startThread)

		case EndThreadTag:
			endThread, ok := readEndThread(record.Data).(EndThread)
			if !ok {
				return nil, nil, nil, nil, nil, nil, nil, fmt.Errorf("incorrect EndThread format")
			}
			endThreads = append(endThreads, endThread)

		default:
			fmt.Printf("Undefined tag: %#X (%d)\n", record.Tag, record.Tag)
		}
	}

	fmt.Println("Root Java Frames:")
	for _, frame := range rootJavaFrames {
		fmt.Printf("%+v\n", frame)
	}

	fmt.Println("Root JNI Globals:")
	for _, global := range rootJNIGlobals {
		fmt.Printf("%+v\n", global)
	}

	// fmt.Println("Root JNI Locals:")
	// for _, local := range rootJNILocals {
	// 	fmt.Printf("%+v\n", local)
	// }

	return stackTraces, stackFrames, ClassSerialToNameId, startThreads, endThreads, rootJNILocals, rootNativeStacks, nil
}
