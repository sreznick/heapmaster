package hprof

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

type HprofHeader struct {
	Magic          string
	IdentifierSize int32
	HighWord       int32
	LowWord        int32
}

type HprofRecord struct {
	Tag    Tag
	Time   int32
	Length int32
	Data   []byte
}

// 0 for object,
// 4 for boolean,
// 5 for char,
// 6 for float,
// 7 for double,
// 8 for byte,
// 9 for short,
// 10 for int,
// 11 for long
type BasicType byte

const (
	Object  BasicType = 0  //only in AllocSites, 0 means not an array, non-zero means an array of the given type
	Boolean BasicType = 4  //1 byte
	Char    BasicType = 5  //1 byte
	Float   BasicType = 6  //4 bytes
	Double  BasicType = 7  //8 bytes
	Byte    BasicType = 8  //1 byte
	Short   BasicType = 9  //2 bytes
	Int     BasicType = 10 //4 bytes
	Long    BasicType = 11 //8 bytes
)

type ID int64 // depends on the identifier size, but we will use int64 for now

// 0x01
type StringInUTF8 struct {
	StringId ID
	Bytes    []byte
}

// 0x02
type LoadClass struct {
	SerialNumber           int32
	ClassObjectId          ID
	StackTraceSerialNumber int32
	ClassNameStringId      ID
}

// 0x03
type UnloadClass struct {
	ClassSerialNumber int32
}

// 0x04
type StackFrame struct {
	FrameId                 ID
	MethodId                ID
	MethodSignatureStringId ID
	SourceFileNameStringId  ID
	ClassSerialNumber       int32
	// > 0 line number
	// 0 no line information
	// -1 unknown location
	// -2 compiled method
	// -3 native method
	flag int32
}

// 0x05
type StackTrace struct {
	StackTraceSerialNumber int32
	ThreadSerialNumber     int32
	NumberOfFrames         int32
	FramesID               []ID
}

// 0x06
type AllocSites struct {
	// 0x1 incremental / complete
	// 0x2 sorted by allocation / line
	// 0x4 whether to force GC
	BitMaskSize            int16
	CutoffRatio            int32
	TotalLiveBytes         int32
	TotalLiveInstances     int32
	TotalBytesAllocated    int64
	TotalInstanceAllocated int64
	NumberOfSites          int32
	Sites                  []struct {
		ArrayIndicator             BasicType
		ClassSerialNumber          int32
		StackTraceSerialNumber     int32
		NumberOfLiveBytes          int32
		NumberOfLiveInstances      int32
		NumberOfBytesAllocated     int32
		NumberOfInstancesAllocated int32
	}
}

// 0x07
type HeapSummary struct {
	LiveBytes          int32
	LiveInstances      int32
	BytesAllocated     int64
	InstancesAllocated int64
}

// 0x0A
type StartThread struct {
	ThreadSerialNumber      int32
	ThreadObjectId          ID
	StackTraceSerialNumber  int32
	ThreadNameStringId      ID
	ThreadGroupNameId       ID
	ThreadParentGroupNameId ID
}

// 0x0B
type EndThread struct {
	ThreadSerialNumber int32
}

// 0x0C or 0x1C
type HeapDump struct {
	data []byte
}

// 0x0D
type CPUSamples struct {
	TotalNumberOfSamples int32
	Traces               []struct {
		NumberOfSamples        int32
		StackTraceSerialNumber int32
	}
}

// 0x0E
type ControlSettings struct {
	// 0x1 alloc traces on/off
	// 0x2 cpu sampling on/off
	BitMask         int32
	StackTraceDepth int16
}

// subtypes in heap dump

type HeapDumpSubTag byte

const (
	RootUnknownTag        HeapDumpSubTag = 0xFF
	RootJNIGlobalTag      HeapDumpSubTag = 0x01
	RootJNILocalTag       HeapDumpSubTag = 0x02
	RootJavaFrameTag      HeapDumpSubTag = 0x03
	RootNativeStackTag    HeapDumpSubTag = 0x04
	RootStickyClassTag    HeapDumpSubTag = 0x05
	RootThreadBlockTag    HeapDumpSubTag = 0x06
	RootMonitorUsedTag    HeapDumpSubTag = 0x07
	RootThreadObjectTag   HeapDumpSubTag = 0x08
	ClassDumpTag          HeapDumpSubTag = 0x20
	InstanceDumpTag       HeapDumpSubTag = 0x21
	ObjectArrayDumpTag    HeapDumpSubTag = 0x22
	PrimitiveArrayDumpTag HeapDumpSubTag = 0x23
)

// 0xFF
type RootUnknown struct {
	ObjectId ID
}

// 0x01
type RootJNIGlobal struct {
	ObjectId ID
	JNIRef   ID
}

// 0x02
type RootJNILocal struct {
	ObjectId           ID
	ThreadSerialNumber int32
	FrameNumber        int32 // -1 for empty
}

// 0x03
type RootJavaFrame struct {
	ObjectId           ID
	ThreadSerialNumber int32
	FrameNumber        int32 // -1 for empty
}

// 0x04
type RootNativeStack struct {
	ObjectId           ID
	ThreadSerialNumber int32
}

// 0x05
type RootStickyClass struct {
	ObjectId ID
}

// 0x06
type RootThreadBlock struct {
	ObjectId           ID
	ThreadSerialNumber int32
}

// 0x07
type RootMonitorUsed struct {
	ObjectId ID
}

// 0x08
type RootThreadObject struct {
	ObjectId               ID
	ThreadSerialNumber     int32
	StackTraceSerialNumber int32
}

// 0x20
type ClassDump struct {
	ClassObjectId            ID
	StackTraceSerialNumber   int32
	SuperClassObjectId       ID
	ClassLoaderObjectId      ID
	SignersObjectId          ID
	ProtectionDomainObjectId ID
	// reserved_1               ID
	// reserved_2               ID
	InstanceSize           int32 //in bytes
	ConstantPoolSize       int16
	ConstantPool           []ConstantPoolRecord
	NumberOfStaticFields   int16
	StaticFields           []StaticFieldRecord
	NumberOfInstanceFields int16
	InstanceFields         []InstanceFieldRecord
}

type ConstantPoolRecord struct {
	ConstantPoolIndex int16
	Type              BasicType
	Value             []byte //size depends on the type
}

type StaticFieldRecord struct {
	StaticFieldNameStringId ID
	Type                    BasicType
	Value                   []byte //size depends on the type
}

type InstanceFieldRecord struct {
	FieldNameStringId ID
	Type              BasicType
}

// 0x21
type InstanceDump struct {
	ObjectId               ID
	StackTraceSerialNumber int32
	ClassObjectId          ID
	NumberOfBytes          int32
	InstanceFieldValues    []byte
}

// 0x22
type ObjectArrayDump struct {
	ArrayObjectId          ID
	StackTraceSerialNumber int32
	NumberOfElements       int32
	ArrayClassObjectId     ID
	Elements               []ID
}

// 0x23
type PrimitiveArrayDump struct {
	ArrayObjectId          ID
	StackTraceSerialNumber int32
	NumberOfElements       int32
	ElementType            BasicType
	Elements               []byte
}

func readHeader(file *os.File) HprofHeader {
	header := HprofHeader{}

	// Read the magic number (JAVA PROFILE 1.0.2\0) 19 bytes
	magic := make([]byte, 19)
	if _, err := file.Read(magic); err != nil {
		fmt.Errorf("Error reading header text: %v\n", err)
		return header
	}
	header.Magic = string(magic)

	if err := binary.Read(file, binary.BigEndian, &header.IdentifierSize); err != nil {
		fmt.Errorf("Error reading identifier size: %v\n", err)
		return header
	}

	if err := binary.Read(file, binary.BigEndian, &header.HighWord); err != nil {
		fmt.Errorf("Error reading high word: %v\n", err)
		return header
	}

	if err := binary.Read(file, binary.BigEndian, &header.LowWord); err != nil {
		fmt.Errorf("Error reading low word: %v\n", err)
		return header
	}

	return header
}

func readRecord(file *os.File) (HprofRecord, error) {
	record := HprofRecord{}

	// Read the tag (1 byte)
	if err := binary.Read(file, binary.BigEndian, &record.Tag); err != nil {
		fmt.Errorf("Error reading tag: %v\n", err)
		return record, err
	}

	// Read the timestamp (4 bytes)
	if err := binary.Read(file, binary.BigEndian, &record.Time); err != nil {
		fmt.Errorf("Error reading timestamp: %v\n", err)
		return record, err
	}

	// Read the length (4 bytes)
	if err := binary.Read(file, binary.BigEndian, &record.Length); err != nil {
		fmt.Errorf("Error reading length: %v\n", err)
		return record, err
	}

	// Read the data based on the length
	record.Data = make([]byte, record.Length)
	if _, err := file.Read(record.Data); err != nil {
		fmt.Errorf("Error reading data: %v\n", err)
		return record, err
	}

	return record, nil
}

func readLoadClass(data []byte) interface{} {
	return LoadClass{
		SerialNumber:           int32(binary.BigEndian.Uint32(data[:4])),
		ClassObjectId:          ID(binary.BigEndian.Uint64(data[4:12])),
		StackTraceSerialNumber: int32(binary.BigEndian.Uint32(data[12:16])),
		ClassNameStringId:      ID(binary.BigEndian.Uint64(data[16:24])),
	}
}

func readStringInUTF8(data []byte) interface{} {
	return StringInUTF8{
		StringId: ID(binary.BigEndian.Uint64(data[:8])),
		Bytes:    append([]byte(nil), data[8:]...),
	}
}

func readStackFrame(data []byte) interface{} {
	return StackFrame{
		FrameId:                 ID(binary.BigEndian.Uint64(data[:8])),
		MethodId:                ID(binary.BigEndian.Uint64(data[8:16])),
		MethodSignatureStringId: ID(binary.BigEndian.Uint64(data[16:24])),
		SourceFileNameStringId:  ID(binary.BigEndian.Uint64(data[24:32])),
		ClassSerialNumber:       int32(binary.BigEndian.Uint32(data[32:36])),
		flag:                    int32(binary.BigEndian.Uint32(data[36:40])),
	}
}

func readStackTrace(data []byte) interface{} {
	stackTrace := StackTrace{
		StackTraceSerialNumber: int32(binary.BigEndian.Uint32(data[:4])),
		ThreadSerialNumber:     int32(binary.BigEndian.Uint32(data[4:8])),
		NumberOfFrames:         int32(binary.BigEndian.Uint32(data[8:12])),
	}

	// Read the frames ID
	stackTrace.FramesID = make([]ID, stackTrace.NumberOfFrames)
	for i := 0; i < int(stackTrace.NumberOfFrames); i++ {
		stackTrace.FramesID[i] = ID(binary.BigEndian.Uint64(data[12+i*8 : 12+(i+1)*8]))
	}

	return stackTrace
}

func readHeapDump(data []byte) HeapDump {
	heapDump := HeapDump{}

	heapDump.data = make([]byte, len(data))
	copy(heapDump.data, data)

	return heapDump
}

func readRootUnknown(data []byte) interface{} {
	return RootUnknown{
		ObjectId: ID(binary.BigEndian.Uint64(data[:8])),
	}
}

func readRootJNIGlobal(data []byte) interface{} {
	return RootJNIGlobal{
		ObjectId: ID(binary.BigEndian.Uint64(data[:8])),
		JNIRef:   ID(binary.BigEndian.Uint64(data[8:16])),
	}
}

func readRootJNILocal(data []byte) interface{} {
	return RootJNILocal{
		ObjectId:           ID(binary.BigEndian.Uint64(data[:8])),
		ThreadSerialNumber: int32(binary.BigEndian.Uint32(data[8:12])),
		FrameNumber:        int32(binary.BigEndian.Uint32(data[12:16])),
	}
}

func readRootJavaFrame(data []byte) interface{} {
	return RootJavaFrame{
		ObjectId:           ID(binary.BigEndian.Uint64(data[:8])),
		ThreadSerialNumber: int32(binary.BigEndian.Uint32(data[8:12])),
		FrameNumber:        int32(binary.BigEndian.Uint32(data[12:16])),
	}
}

func readRootNativeStack(data []byte) interface{} {
	return RootNativeStack{
		ObjectId:           ID(binary.BigEndian.Uint64(data[:8])),
		ThreadSerialNumber: int32(binary.BigEndian.Uint32(data[8:12])),
	}
}

func readRootStickyClass(data []byte) interface{} {
	return RootStickyClass{
		ObjectId: ID(binary.BigEndian.Uint64(data[:8])),
	}
}

func readRootThreadBlock(data []byte) interface{} {
	return RootThreadBlock{
		ObjectId:           ID(binary.BigEndian.Uint64(data[:8])),
		ThreadSerialNumber: int32(binary.BigEndian.Uint32(data[8:12])),
	}
}

func readRootMonitorUsed(data []byte) interface{} {
	return RootMonitorUsed{
		ObjectId: ID(binary.BigEndian.Uint64(data[:8])),
	}
}

func readRootThreadObject(data []byte) interface{} {
	return RootThreadObject{
		ObjectId:               ID(binary.BigEndian.Uint64(data[:8])),
		ThreadSerialNumber:     int32(binary.BigEndian.Uint32(data[8:12])),
		StackTraceSerialNumber: int32(binary.BigEndian.Uint32(data[12:16])),
	}
}

func readClassDump(data []byte) interface{} {
	classDump := ClassDump{
		ClassObjectId:            ID(binary.BigEndian.Uint64(data[:8])),
		StackTraceSerialNumber:   int32(binary.BigEndian.Uint32(data[8:12])),
		SuperClassObjectId:       ID(binary.BigEndian.Uint64(data[12:20])),
		ClassLoaderObjectId:      ID(binary.BigEndian.Uint64(data[20:28])),
		SignersObjectId:          ID(binary.BigEndian.Uint64(data[28:36])),
		ProtectionDomainObjectId: ID(binary.BigEndian.Uint64(data[36:44])),
		InstanceSize:             int32(binary.BigEndian.Uint32(data[60:64])),
	}

	// Read the constant pool
	classDump.ConstantPoolSize = int16(binary.BigEndian.Uint16(data[64:66]))
	classDump.ConstantPool = make([]ConstantPoolRecord, classDump.ConstantPoolSize)
	offset := 66
	for i := 0; i < int(classDump.ConstantPoolSize); i++ {
		constantPoolRecord := ConstantPoolRecord{
			ConstantPoolIndex: int16(binary.BigEndian.Uint16(data[offset : offset+2])),
			Type:              BasicType(data[offset+2]),
		}
		offset += 3
		switch constantPoolRecord.Type {
		case Boolean, Byte, Char:
			constantPoolRecord.Value = make([]byte, 1)
			copy(constantPoolRecord.Value, data[offset:offset+1])
			offset++
		case Short:
			constantPoolRecord.Value = make([]byte, 2)
			copy(constantPoolRecord.Value, data[offset:offset+2])
			offset += 2
		case Float, Int:
			constantPoolRecord.Value = make([]byte, 4)
			copy(constantPoolRecord.Value, data[offset:offset+4])
			offset += 4
		case Double, Long:
			constantPoolRecord.Value = make([]byte, 8)
			copy(constantPoolRecord.Value, data[offset:offset+8])
			offset += 8
		}
		classDump.ConstantPool[i] = constantPoolRecord
	}

	// Read the static fields
	classDump.NumberOfStaticFields = int16(binary.BigEndian.Uint16(data[offset : offset+2]))
	classDump.StaticFields = make([]StaticFieldRecord, classDump.NumberOfStaticFields)
	offset += 2
	for i := 0; i < int(classDump.NumberOfStaticFields); i++ {
		staticFieldRecord := StaticFieldRecord{
			StaticFieldNameStringId: ID(binary.BigEndian.Uint64(data[offset : offset+8])),
			Type:                    BasicType(data[offset+8]),
		}
		offset += 9
		switch staticFieldRecord.Type {
		case Boolean, Byte, Char:
			staticFieldRecord.Value = make([]byte, 1)
			copy(staticFieldRecord.Value, data[offset:offset+1])
			offset++
		case Short:
			staticFieldRecord.Value = make([]byte, 2)
			copy(staticFieldRecord.Value, data[offset:offset+2])
			offset += 2
		case Float, Int:
			staticFieldRecord.Value = make([]byte, 4)
			copy(staticFieldRecord.Value, data[offset:offset+4])
			offset += 4
		case Double, Long:
			staticFieldRecord.Value = make([]byte, 8)
			copy(staticFieldRecord.Value, data[offset:offset+8])
			offset += 8
		}
		classDump.StaticFields[i] = staticFieldRecord
	}

	// Read the instance fields
	classDump.NumberOfInstanceFields = int16(binary.BigEndian.Uint16(data[offset : offset+2]))
	classDump.InstanceFields = make([]InstanceFieldRecord, classDump.NumberOfInstanceFields)
	offset += 2
	for i := 0; i < int(classDump.NumberOfInstanceFields); i++ {
		instanceFieldRecord := InstanceFieldRecord{
			FieldNameStringId: ID(binary.BigEndian.Uint64(data[offset : offset+8])),
			Type:              BasicType(data[offset+8]),
		}
		classDump.InstanceFields[i] = instanceFieldRecord
		offset += 9
	}

	return classDump
}

func readInstanceDump(data []byte) interface{} {
	instanceDump := InstanceDump{
		ObjectId:               ID(binary.BigEndian.Uint64(data[:8])),
		StackTraceSerialNumber: int32(binary.BigEndian.Uint32(data[8:12])),
		ClassObjectId:          ID(binary.BigEndian.Uint64(data[12:20])),
		NumberOfBytes:          int32(binary.BigEndian.Uint32(data[20:24])),
	}

	// Read the fields
	instanceDump.InstanceFieldValues = make([]byte, instanceDump.NumberOfBytes)
	copy(instanceDump.InstanceFieldValues, data[24:24+instanceDump.NumberOfBytes])

	return instanceDump
}

func readObjectArrayDump(data []byte) interface{} {
	objectArrayDump := ObjectArrayDump{
		ArrayObjectId:          ID(binary.BigEndian.Uint64(data[:8])),
		StackTraceSerialNumber: int32(binary.BigEndian.Uint32(data[8:12])),
		NumberOfElements:       int32(binary.BigEndian.Uint32(data[12:16])),
		ArrayClassObjectId:     ID(binary.BigEndian.Uint64(data[16:24])),
	}

	// Read the elements
	objectArrayDump.Elements = make([]ID, objectArrayDump.NumberOfElements)
	for i := 0; i < int(objectArrayDump.NumberOfElements); i++ {
		objectArrayDump.Elements[i] = ID(binary.BigEndian.Uint64(data[24+i*8 : 24+(i+1)*8]))
	}

	return objectArrayDump
}

func readPrimitiveArrayDump(data []byte) interface{} {
	primitiveArrayDump := PrimitiveArrayDump{
		ArrayObjectId:          ID(binary.BigEndian.Uint64(data[:8])),
		StackTraceSerialNumber: int32(binary.BigEndian.Uint32(data[8:12])),
		NumberOfElements:       int32(binary.BigEndian.Uint32(data[12:16])),
		ElementType:            BasicType(data[16]),
	}

	var elementSize int32
	switch primitiveArrayDump.ElementType {
	case Boolean, Byte, Char:
		elementSize = 1
	case Short:
		elementSize = 2
	case Float, Int:
		elementSize = 4
	case Double, Long:
		elementSize = 8
	}

	// Read the elements
	primitiveArrayDump.Elements = make([]byte, primitiveArrayDump.NumberOfElements*elementSize)
	copy(primitiveArrayDump.Elements, data[17:17+(primitiveArrayDump.NumberOfElements*elementSize)])

	return primitiveArrayDump
}

func parseHeapDump(heapDumpFile *os.File) {

	type readerFunction func([]byte) interface{}

	subTagFuncMap := map[HeapDumpSubTag]readerFunction{
		RootUnknownTag:      readRootUnknown,
		RootJNIGlobalTag:    readRootJNIGlobal,
		RootJNILocalTag:     readRootJNILocal,
		RootJavaFrameTag:    readRootJavaFrame,
		RootNativeStackTag:  readRootNativeStack,
		RootStickyClassTag:  readRootStickyClass,
		RootThreadBlockTag:  readRootThreadBlock,
		RootMonitorUsedTag:  readRootMonitorUsed,
		RootThreadObjectTag: readRootThreadObject,
	}

	subTagSizeMap := map[HeapDumpSubTag]int{
		RootUnknownTag:      8,
		RootJNIGlobalTag:    16,
		RootJNILocalTag:     16,
		RootJavaFrameTag:    16,
		RootNativeStackTag:  12,
		RootStickyClassTag:  8,
		RootThreadBlockTag:  12,
		RootMonitorUsedTag:  8,
		RootThreadObjectTag: 16,
	}

	IDtoStringInUTF8 := make(map[ID]string)

	// Read the header
	header := readHeader(heapDumpFile)
	fmt.Printf("Header: %+v\n", header)

	// Read records
	for {
		record, err := readRecord(heapDumpFile)
		if err == io.EOF {
			fmt.Errorf("Reached end of file.")
			break
		} else if err != nil {
			fmt.Errorf("Error reading record: %v\n", err)
			break
		}

		if record.Tag == StringUtf8Tag {
			stringInUTF8 := readStringInUTF8(record.Data).(StringInUTF8)
			IDtoStringInUTF8[stringInUTF8.StringId] = string(stringInUTF8.Bytes)
			continue
		}

		if record.Tag == LoadClassTag {
			classLoad := readLoadClass(record.Data).(LoadClass)
			fmt.Printf("----LoadClass: %d - %s\n", classLoad.ClassObjectId, IDtoStringInUTF8[classLoad.ClassNameStringId])
			continue
		}

		if record.Tag == UnloadClassTag {
			//TODO
		}

		if record.Tag == StackFrameTag {
			stackFrame := readStackFrame(record.Data).(StackFrame)
			fmt.Printf("----StackFrame: %d - %s\n", stackFrame.MethodId, IDtoStringInUTF8[stackFrame.MethodSignatureStringId])
			continue
		}

		if record.Tag == StackTraceTag {
			stackTrace := readStackTrace(record.Data).(StackTrace)
			fmt.Printf("----StackTrace: %d\n", stackTrace.StackTraceSerialNumber)
		}

		if record.Tag == AllocSitesTag {
			//TODO
		}

		if record.Tag == HeapDumpTag || record.Tag == HeapDumpSegmentTag {
			heapDump := readHeapDump(record.Data)
			fmt.Printf("----HeapDump: %d\n", len(heapDump.data))
			reader := bytes.NewReader(heapDump.data)
			for {
				var subTag HeapDumpSubTag
				err := binary.Read(reader, binary.BigEndian, &subTag)
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Errorf("Error reading sub tag: %v\n", err)
					break
				}

				switch subTag {
				case ClassDumpTag:
				//TODO
				case InstanceDumpTag:
				//TODO
				case ObjectArrayDumpTag:
				//TODO
				case PrimitiveArrayDumpTag:
				//TODO
				default:
					if readerFunction, ok := subTagFuncMap[subTag]; ok {
						subTagSize := subTagSizeMap[subTag]
						data := make([]byte, subTagSize)
						_, err := reader.Read(data)
						if err != nil {
							fmt.Errorf("Error reading sub tag data: %v\n", err)
						}
						fmt.Printf("------%s\n", readerFunction(data))
					} else {
						fmt.Errorf("Unknown sub tag: %d\n", subTag)
					}
				}
			}
		}
	}
}
