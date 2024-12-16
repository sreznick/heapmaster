package hprof

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type HprofHeader struct {
	Magic string
	Identifier int32
	HighWord int32
	LowWord int32
}

type HprofRecord struct {
	Tag byte
	Time int32
	Length int32
	Data []byte
}

type LoadClass struct {
	SerialNumber int32
	ClassObjectId int64
	StackTraceSerialNumber int32
	ClassNameStringId int64
}

type StringInUTF8 struct {
	SerialNumber int64
	Bytes []byte
}


type StackFrame struct {
	FrameId int64
	MethodId int64
	MethodSignatureStringId int64
	SourceFileNameStringId int64
	ClassSerialNumber int32
	flag int32
}


type StackTrace struct {
	StackTraceSerialNumber int32
	ThreadSerialNumber int32
	NumberOfFrames int32
	FramesID []int64
}

type HeapDump struct {
	data []byte
}
	
//subtypes in heap dump
type RootUnknown struct {
	ObjectId int64
}

type RootJNIGlobal struct {
	ObjectId int64
	JNIRef int64
}

type RootJNILocal struct {
	ObjectId int64
	ThreadSerialNumber int32
	FrameNumber int32 // -1 for empty
}

type RootJavaFrame struct {
	ObjectId int64
	ThreadSerialNumber int32
	FrameNumber int32 // -1 for empty
}

type RootNativeStack struct {
	ObjectId int64
	ThreadSerialNumber int32
}

type RootStickyClass struct {
	ObjectId int64
}

type RootThreadBlock struct {
	ObjectId int64
	ThreadSerialNumber int32
}

type RootMonitorUsed struct {
	ObjectId int64
}

type RootThreadObject struct {
	ObjectId int64
	ThreadSerialNumber int32
	StackTraceSerialNumber int32
}

type ClassDump struct {
	ClassObjectId int64
	StackSerialNumber int32
	SuperClassObjectId int64
	ClassLoaderObjectId int64
	SignersObjectId int64
	ProtectionDomainObjectId int64
	nothing_1 int64
	nothing_2 int64
	InstanceSize int32
	ConstantPool []byte
	StaticFields []byte
	InstanceFields []byte
}

type InstanceDump struct {
	ObjectId int64
	StackTraceSerialNumber int32
	ClassObjectId int64
	NumberOfBytes int32
	Fields []byte
}

type ObjectArrayDump struct {
	ArrayObjectId int64
	StackTraceSerialNumber int32
	NumberOfElements int32
	ArrayClassObjectId int64
	Elements []int64
}

type PrimitiveArrayDump struct {
	ArrayObjectId int64
	StackTraceSerialNumber int32
	NumberOfElements int32
	ElementType byte 
	// 4 for boolean, 
	// 5 for char, 
	// 6 for float, 
	// 7 for double, 
	// 8 for byte, 
	// 9 for short, 
	// 10 for int, 
	// 11 for long
	Elements []byte
}

func readHeader(file *os.File) (HprofHeader, error) {
	header := HprofHeader{}

	// Read the magic number (JAVA PROFILE 1.0.2\0) 19 bytes
	magic := make([]byte, 19)
	if _, err := file.Read(magic); err != nil {
		return header, err
	}
	header.Magic = string(magic)

	if err := binary.Read(file, binary.BigEndian, &header.Identifier); err != nil {
		return header, err
	}

	if err := binary.Read(file, binary.BigEndian, &header.HighWord); err != nil {
		return header, err
	}

	if err := binary.Read(file, binary.BigEndian, &header.LowWord); err != nil {
		return header, err
	}

	return header, nil
}

func readRecord(file *os.File) (HprofRecord, error) {
	record := HprofRecord{}

	// Read the tag (1 byte)
	if err := binary.Read(file, binary.BigEndian, &record.Tag); err != nil {
		return record, err
	}

	// Read the timestamp (4 bytes)
	if err := binary.Read(file, binary.BigEndian, &record.Time); err != nil {
		return record, err
	}

	// Read the length (4 bytes)
	if err := binary.Read(file, binary.BigEndian, &record.Length); err != nil {
		return record, err
	}

	// Read the data based on the length
	record.Data = make([]byte, record.Length)
	if _, err := file.Read(record.Data); err != nil {
		return record, err
	}

	return record, nil
}

func readLoadClass(data []byte) (LoadClass, error) {
	loadClass := LoadClass{}

	// Read the serial number (4 bytes)
	loadClass.SerialNumber = int32(binary.BigEndian.Uint32(data[:4]))

	// Read the class object ID (8 bytes)
	loadClass.ClassObjectId = int64(binary.BigEndian.Uint64(data[4:12]))

	// Read the stack trace serial number (4 bytes)
	loadClass.StackTraceSerialNumber = int32(binary.BigEndian.Uint32(data[12:16]))

	// Read the class name string ID (8 bytes)
	loadClass.ClassNameStringId = int64(binary.BigEndian.Uint64(data[16:24]))

	return loadClass, nil
}

func readStringInUTF8(data []byte) (StringInUTF8, error) {
	stringInUTF8 := StringInUTF8{}

	// Read the serial number (8 bytes)
	stringInUTF8.SerialNumber = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the bytes
	stringInUTF8.Bytes = make([]byte, len(data)-8)
	copy(stringInUTF8.Bytes, data[8:])

	return stringInUTF8, nil
}

func readStackFrame(data []byte) (StackFrame, error) {
	stackFrame := StackFrame{}

	// Read the frame ID (8 bytes)
	stackFrame.FrameId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the method ID (8 bytes)
	stackFrame.MethodId = int64(binary.BigEndian.Uint64(data[8:16]))

	// Read the method signature string ID (8 bytes)
	stackFrame.MethodSignatureStringId = int64(binary.BigEndian.Uint64(data[16:24]))

	// Read the source file name string ID (8 bytes)
	stackFrame.SourceFileNameStringId = int64(binary.BigEndian.Uint64(data[24:32]))

	// Read the class serial number (4 bytes)
	stackFrame.ClassSerialNumber = int32(binary.BigEndian.Uint32(data[32:36]))

	// Read the flag (4 bytes)
	stackFrame.flag = int32(binary.BigEndian.Uint32(data[36:40]))

	return stackFrame, nil
}

func readStackTrace(data []byte) (StackTrace, error) {
	stackTrace := StackTrace{}

	// Read the stack trace serial number (4 bytes)
	stackTrace.StackTraceSerialNumber = int32(binary.BigEndian.Uint32(data[:4]))

	// Read the thread serial number (4 bytes)
	stackTrace.ThreadSerialNumber = int32(binary.BigEndian.Uint32(data[4:8]))

	// Read the number of frames (4 bytes)
	stackTrace.NumberOfFrames = int32(binary.BigEndian.Uint32(data[8:12]))

	// Read the frames ID
	stackTrace.FramesID = make([]int64, stackTrace.NumberOfFrames)
	for i := 0; i < int(stackTrace.NumberOfFrames); i++ {
		stackTrace.FramesID[i] = int64(binary.BigEndian.Uint64(data[12+i*8 : 12+(i+1)*8]))
	}

	return stackTrace, nil
}


func readHeapDump(data []byte) (HeapDump, error) {
	heapDump := HeapDump{}

	heapDump.data = make([]byte, len(data))
	copy(heapDump.data, data)

	return heapDump, nil
}


func readRootUnknown(data []byte) (RootUnknown, error) {
	rootUnknown := RootUnknown{}

	// Read the object ID (8 bytes)
	rootUnknown.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	return rootUnknown, nil
}

func readRootJNIGlobal(data []byte) (RootJNIGlobal, error) {
	rootJNIGlobal := RootJNIGlobal{}

	// Read the object ID (8 bytes)
	rootJNIGlobal.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the JNI ref (8 bytes)
	rootJNIGlobal.JNIRef = int64(binary.BigEndian.Uint64(data[8:16]))

	return rootJNIGlobal, nil
}

func readRootJNILocal(data []byte) (RootJNILocal, error) {
	rootJNILocal := RootJNILocal{}

	// Read the object ID (8 bytes)
	rootJNILocal.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the thread serial number
	rootJNILocal.ThreadSerialNumber = int32(binary.BigEndian.Uint32(data[8:12]))

	// Read the frame number
	rootJNILocal.FrameNumber = int32(binary.BigEndian.Uint32(data[12:16]))

	return rootJNILocal, nil
}


func readRootJavaFrame(data []byte) (RootJavaFrame, error) {
	rootJavaFrame := RootJavaFrame{}

	// Read the object ID (8 bytes)
	rootJavaFrame.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the thread serial number
	rootJavaFrame.ThreadSerialNumber = int32(binary.BigEndian.Uint32(data[8:12]))

	// Read the frame number
	rootJavaFrame.FrameNumber = int32(binary.BigEndian.Uint32(data[12:16]))

	return rootJavaFrame, nil
}


func readRootNativeStack(data []byte) (RootNativeStack, error) {
	rootNativeStack := RootNativeStack{}

	// Read the object ID (8 bytes)
	rootNativeStack.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the thread serial number
	rootNativeStack.ThreadSerialNumber = int32(binary.BigEndian.Uint32(data[8:12]))

	return rootNativeStack, nil
}

func readRootStickyClass(data []byte) (RootStickyClass, error) {
	rootStickyClass := RootStickyClass{}

	// Read the object ID (8 bytes)
	rootStickyClass.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	return rootStickyClass, nil
}

func readRootThreadBlock(data []byte) (RootThreadBlock, error) {
	rootThreadBlock := RootThreadBlock{}

	// Read the object ID (8 bytes)
	rootThreadBlock.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the thread serial number
	rootThreadBlock.ThreadSerialNumber = int32(binary.BigEndian.Uint32(data[8:12]))

	return rootThreadBlock, nil
}

func readRootMonitorUsed(data []byte) (RootMonitorUsed, error) {
	rootMonitorUsed := RootMonitorUsed{}

	// Read the object ID (8 bytes)
	rootMonitorUsed.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	return rootMonitorUsed, nil
}

func readRootThreadObject(data []byte) (RootThreadObject, error) {
	rootThreadObject := RootThreadObject{}

	// Read the object ID (8 bytes)
	rootThreadObject.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the thread serial number
	rootThreadObject.ThreadSerialNumber = int32(binary.BigEndian.Uint32(data[8:12]))

	// Read the stack trace serial number
	rootThreadObject.StackTraceSerialNumber = int32(binary.BigEndian.Uint32(data[12:16]))

	return rootThreadObject, nil
}


func readClassDump(data []byte) (ClassDump, error) {
	classDump := ClassDump{}

	// Read the class object ID (8 bytes)
	classDump.ClassObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the stack serial number (4 bytes)
	classDump.StackSerialNumber = int32(binary.BigEndian.Uint32(data[8:12]))

	// Read the super class object ID (8 bytes)
	classDump.SuperClassObjectId = int64(binary.BigEndian.Uint64(data[12:20]))

	// Read the class loader object ID (8 bytes)
	classDump.ClassLoaderObjectId = int64(binary.BigEndian.Uint64(data[20:28]))

	// Read the signers object ID (8 bytes)
	classDump.SignersObjectId = int64(binary.BigEndian.Uint64(data[28:36]))

	// Read the protection domain object ID (8 bytes)
	classDump.ProtectionDomainObjectId = int64(binary.BigEndian.Uint64(data[36:44]))

	// Read the nothing_1 (8 bytes)
	classDump.nothing_1 = int64(binary.BigEndian.Uint64(data[44:52]))

	// Read the nothing_2 (8 bytes)
	classDump.nothing_2 = int64(binary.BigEndian.Uint64(data[52:60]))

	// Read the instance size (4 bytes)
	classDump.InstanceSize = int32(binary.BigEndian.Uint32(data[60:64]))

	// Read the constant pool
	constantPoolSize := int32(binary.BigEndian.Uint32(data[64:68]))
	classDump.ConstantPool = make([]byte, constantPoolSize)
	copy(classDump.ConstantPool, data[68:68+constantPoolSize])

	// Read the static fields
	staticFieldsSize := int32(binary.BigEndian.Uint32(data[68:72]))
	classDump.StaticFields = make([]byte, staticFieldsSize)
	copy(classDump.StaticFields, data[72+constantPoolSize:72+constantPoolSize+staticFieldsSize])


	// Read the instance fields
	instanceFieldsSize := int32(binary.BigEndian.Uint32(data[72:76]))
	classDump.InstanceFields = make([]byte, instanceFieldsSize)
	copy(classDump.InstanceFields, data[76+constantPoolSize+staticFieldsSize:76+constantPoolSize+staticFieldsSize+instanceFieldsSize])

	return classDump, nil

}


func readInstanceDump(data []byte) (InstanceDump, error) {
	instanceDump := InstanceDump{}

	// Read the object ID (8 bytes)
	instanceDump.ObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the stack trace serial number (4 bytes)
	instanceDump.StackTraceSerialNumber = int32(binary.BigEndian.Uint32(data[8:12]))

	// Read the class object ID (8 bytes)
	instanceDump.ClassObjectId = int64(binary.BigEndian.Uint64(data[12:20]))

	// Read the number of bytes (4 bytes)
	instanceDump.NumberOfBytes = int32(binary.BigEndian.Uint32(data[20:24]))

	// Read the fields
	instanceDump.Fields = make([]byte, instanceDump.NumberOfBytes)
	copy(instanceDump.Fields, data[24:24+instanceDump.NumberOfBytes])

	return instanceDump, nil
}

func readObjectArrayDump(data []byte) (ObjectArrayDump, error) {
	objectArrayDump := ObjectArrayDump{}

	// Read the array object ID (8 bytes)
	objectArrayDump.ArrayObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the stack trace serial number (4 bytes)
	objectArrayDump.StackTraceSerialNumber = int32(binary.BigEndian.Uint32(data[8:12]))

	// Read the number of elements (4 bytes)
	objectArrayDump.NumberOfElements = int32(binary.BigEndian.Uint32(data[12:16]))

	// Read the array class object ID (8 bytes)
	objectArrayDump.ArrayClassObjectId = int64(binary.BigEndian.Uint64(data[16:24]))

	// Read the elements
	objectArrayDump.Elements = make([]int64, objectArrayDump.NumberOfElements)
	for i := 0; i < int(objectArrayDump.NumberOfElements); i++ {
		objectArrayDump.Elements[i] = int64(binary.BigEndian.Uint64(data[24+i*8 : 24+(i+1)*8]))
	}

	return objectArrayDump, nil
}

func readPrimitiveArrayDump(data []byte) (PrimitiveArrayDump, error) {
	primitiveArrayDump := PrimitiveArrayDump{}

	// Read the array object ID (8 bytes)
	primitiveArrayDump.ArrayObjectId = int64(binary.BigEndian.Uint64(data[:8]))

	// Read the stack trace serial number (4 bytes)
	primitiveArrayDump.StackTraceSerialNumber = int32(binary.BigEndian.Uint32(data[8:12]))

	// Read the number of elements (4 bytes)
	primitiveArrayDump.NumberOfElements = int32(binary.BigEndian.Uint32(data[12:16]))

	// Read the element type (1 byte)
	primitiveArrayDump.ElementType = data[16]

	// Read the elements
	primitiveArrayDump.Elements = make([]byte, primitiveArrayDump.NumberOfElements)
	copy(primitiveArrayDump.Elements, data[17:17+primitiveArrayDump.NumberOfElements])

	return primitiveArrayDump, nil
}


func main() {
	subTagMap := map[byte]func(data []byte) (interface{}, error) {
		0xFF: func(data []byte) (interface{}, error) { return readRootUnknown(data) },
		0x01: func(data []byte) (interface{}, error) { return readRootJNIGlobal(data) },
		0x02: func(data []byte) (interface{}, error) { return readRootJNILocal(data) },
		0x03: func(data []byte) (interface{}, error) { return readRootJavaFrame(data) },
		0x04: func(data []byte) (interface{}, error) { return readRootNativeStack(data) },
		0x05: func(data []byte) (interface{}, error) { return readRootStickyClass(data) },
		0x06: func(data []byte) (interface{}, error) { return readRootThreadBlock(data) },
		0x07: func(data []byte) (interface{}, error) { return readRootMonitorUsed(data) },
		0x08: func(data []byte) (interface{}, error) { return readRootThreadObject(data) },
		0x20: func(data []byte) (interface{}, error) { return readClassDump(data) },
		0x21: func(data []byte) (interface{}, error) { return readInstanceDump(data) },
		0x22: func(data []byte) (interface{}, error) { return readObjectArrayDump(data) },
		0x23: func(data []byte) (interface{}, error) { return readPrimitiveArrayDump(data) },
	}
	subTagMap[0x01](nil) // just to avoid unused error


	IDtoStringInUTF8 := make(map[int64]string)

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run parser.go <file.hprof>")
		return
	}

	filePath := os.Args[1]
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	// Read the header
	header, err := readHeader(file)
	if err != nil {
		fmt.Printf("Error reading header: %v\n", err)
		return
	}
	fmt.Printf("Header: %+v\n", header)



	// Read records
	takeOnlyOneRecord := true
	currentClassLoad := LoadClass{}
	for i := 0; i < 20;  {
		record, err := readRecord(file)
		if err == io.EOF {
			fmt.Println("Reached end of file.")
			break
		} else if err != nil {
			fmt.Printf("Error reading record: %v\n", err)
			break
		}

		if record.Tag == 0x01 {
			stringInUTF8, err := readStringInUTF8(record.Data)
			if err != nil {
				fmt.Printf("Error reading StringInUTF8: %v\n", err)
				break
			}
			IDtoStringInUTF8[stringInUTF8.SerialNumber] = string(stringInUTF8.Bytes)
			continue;
		}


		if record.Tag == 0x02 && takeOnlyOneRecord {
			takeOnlyOneRecord = false
			// fmt.Printf("Record: %+v\n", record)
			loadClass, err := readLoadClass(record.Data)
			currentClassLoad = loadClass
			if err != nil {
				fmt.Printf("Error reading LoadClass: %v\n", err)
				break
			}
			fmt.Printf("----LoadClass: %d - %s\n", currentClassLoad.ClassObjectId, IDtoStringInUTF8[currentClassLoad.ClassNameStringId])
		}

		if record.Tag == 0x04 {
			stackFrame, err := readStackFrame(record.Data)
			if err != nil {
				fmt.Printf("Error reading StackFrame: %v\n", err)
				break
			}
			fmt.Printf("----StackFrame: %d - %s\n", stackFrame.MethodId, IDtoStringInUTF8[stackFrame.MethodSignatureStringId])
		}

		if record.Tag == 0x05 {
			stackTrace, err := readStackTrace(record.Data)
			if err != nil {
				fmt.Printf("Error reading StackTrace: %v\n", err)
				break
			}
			fmt.Printf("----StackTrace: %d\n", stackTrace.StackTraceSerialNumber)
			for _, frameID := range stackTrace.FramesID {
				fmt.Printf("--------FrameID: %d\n", frameID)
			}
		}

	}

	output, err :=  os.OpenFile(os.Args[2], os.O_CREATE|os.O_WRONLY, 0644)
	defer output.Close()
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}

	// for k, v := range dict {
	// 	// fmt.Printf("%d: %s\n", k, v)
	// 	output.Write([]byte(fmt.Sprintf("%d: %s\n", k, v)))
	// }

}
