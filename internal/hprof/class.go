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

func (bt BasicType) GetSize() int32 {
	switch bt {
	case Boolean, Byte, Char:
		return 1
	case Short:
		return 2
	case Float, Int:
		return 4
	case Double, Long:
		return 8
	}
	return -1
}

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

func readID(reader *bytes.Reader) ID {
	var id ID
	binary.Read(reader, binary.BigEndian, &id)
	return id
}

func readInt64(reader *bytes.Reader) int64 {
	var i int64
	binary.Read(reader, binary.BigEndian, &i)
	return i
}

func readInt32(reader *bytes.Reader) int32 {
	var i int32
	binary.Read(reader, binary.BigEndian, &i)
	return i
}

func readInt16(reader *bytes.Reader) int16 {
	var i int16
	binary.Read(reader, binary.BigEndian, &i)
	return i
}

func readBasicType(reader *bytes.Reader) BasicType {
	var bt BasicType
	binary.Read(reader, binary.BigEndian, &bt)
	return bt
}

func readArray(reader *bytes.Reader, size int32) []byte {
	data := make([]byte, size)
	reader.Read(data)
	return data
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

func readRootUnknown(reader *bytes.Reader) interface{} {
	return RootUnknown{
		ObjectId: readID(reader),
	}
}

func readRootJNIGlobal(reader *bytes.Reader) interface{} {
	return RootJNIGlobal{
		ObjectId: readID(reader),
		JNIRef:   readID(reader),
	}
}

func readRootJNILocal(reader *bytes.Reader) interface{} {
	return RootJNILocal{
		ObjectId:           readID(reader),
		ThreadSerialNumber: readInt32(reader),
		FrameNumber:        readInt32(reader),
	}
}

func readRootJavaFrame(reader *bytes.Reader) interface{} {
	return RootJavaFrame{
		ObjectId:           readID(reader),
		ThreadSerialNumber: readInt32(reader),
		FrameNumber:        readInt32(reader),
	}
}

func readRootNativeStack(reader *bytes.Reader) interface{} {
	return RootNativeStack{
		ObjectId:           readID(reader),
		ThreadSerialNumber: readInt32(reader),
	}
}

func readRootStickyClass(reader *bytes.Reader) interface{} {
	return RootStickyClass{
		ObjectId: readID(reader),
	}
}

func readRootThreadBlock(reader *bytes.Reader) interface{} {
	return RootThreadBlock{
		ObjectId:           readID(reader),
		ThreadSerialNumber: readInt32(reader),
	}
}

func readRootMonitorUsed(reader *bytes.Reader) interface{} {
	return RootMonitorUsed{
		ObjectId: readID(reader),
	}
}

func readRootThreadObject(reader *bytes.Reader) interface{} {
	return RootThreadObject{
		ObjectId:               readID(reader),
		ThreadSerialNumber:     readInt32(reader),
		StackTraceSerialNumber: readInt32(reader),
	}
}

func readClassDump(reader *bytes.Reader) interface{} {
	classDump := ClassDump{
		ClassObjectId:            readID(reader),
		StackTraceSerialNumber:   readInt32(reader),
		SuperClassObjectId:       readID(reader),
		ClassLoaderObjectId:      readID(reader),
		SignersObjectId:          readID(reader),
		ProtectionDomainObjectId: readID(reader),
		InstanceSize:             readInt32(reader),
	}

	// Read the constant pool
	classDump.ConstantPoolSize = readInt16(reader)
	classDump.ConstantPool = make([]ConstantPoolRecord, classDump.ConstantPoolSize)
	for i := 0; i < int(classDump.ConstantPoolSize); i++ {
		constantPoolRecord := ConstantPoolRecord{
			ConstantPoolIndex: readInt16(reader),
			Type:              readBasicType(reader),
		}
		constantPoolRecord.Value = readArray(reader, constantPoolRecord.Type.GetSize())
		classDump.ConstantPool[i] = constantPoolRecord
	}

	// Read the static fields
	classDump.NumberOfStaticFields = readInt16(reader)
	classDump.StaticFields = make([]StaticFieldRecord, classDump.NumberOfStaticFields)
	for i := 0; i < int(classDump.NumberOfStaticFields); i++ {
		staticFieldRecord := StaticFieldRecord{
			StaticFieldNameStringId: readID(reader),
			Type:                    readBasicType(reader),
		}
		staticFieldRecord.Value = readArray(reader, staticFieldRecord.Type.GetSize())
		classDump.StaticFields[i] = staticFieldRecord
	}

	// Read the instance fields
	classDump.NumberOfInstanceFields = readInt16(reader)
	classDump.InstanceFields = make([]InstanceFieldRecord, classDump.NumberOfInstanceFields)
	for i := 0; i < int(classDump.NumberOfInstanceFields); i++ {
		instanceFieldRecord := InstanceFieldRecord{
			FieldNameStringId: readID(reader),
			Type:              readBasicType(reader),
		}
		classDump.InstanceFields[i] = instanceFieldRecord
	}

	return classDump
}

func readInstanceDump(reader *bytes.Reader) interface{} {
	instanceDump := InstanceDump{
		ObjectId:               readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		ClassObjectId:          readID(reader),
		NumberOfBytes:          readInt32(reader),
	}

	instanceDump.InstanceFieldValues = readArray(reader, instanceDump.NumberOfBytes)

	return instanceDump
}

func readObjectArrayDump(reader *bytes.Reader) interface{} {
	objectArrayDump := ObjectArrayDump{
		ArrayObjectId:          readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		NumberOfElements:       readInt32(reader),
		ArrayClassObjectId:     readID(reader),
	}

	objectArrayDump.Elements = make([]ID, objectArrayDump.NumberOfElements)
	for i := 0; i < int(objectArrayDump.NumberOfElements); i++ {
		objectArrayDump.Elements[i] = readID(reader)
	}

	return objectArrayDump
}

func readPrimitiveArrayDump(reader *bytes.Reader) interface{} {
	primitiveArrayDump := PrimitiveArrayDump{
		ArrayObjectId:          readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		NumberOfElements:       readInt32(reader),
		ElementType:            readBasicType(reader),
	}

	primitiveArrayDump.Elements = readArray(
		reader,
		primitiveArrayDump.NumberOfElements*primitiveArrayDump.ElementType.GetSize(),
	)

	return primitiveArrayDump
}

func parseHeapDump(heapDumpFile *os.File) {

	type readerFunction func(*bytes.Reader) interface{}

	subTagFuncMap := map[HeapDumpSubTag]readerFunction{
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

				if readerFunction, ok := subTagFuncMap[subTag]; ok {
					fmt.Printf("------%s\n", readerFunction(reader))
				} else {
					fmt.Errorf("Unknown sub tag: %d\n", subTag)
				}
			}
		}
	}
}
