package hprof

type Tag uint8

const (
	StringUtf8Tag      Tag = 0x01
	LoadClassTag       Tag = 0x02
	UnloadClassTag     Tag = 0x03
	StackFrameTag      Tag = 0x04
	StackTraceTag      Tag = 0x05
	AllocSitesTag      Tag = 0x06
	HeapSummaryTag     Tag = 0x07
	StartThreadTag     Tag = 0x0A
	EndThreadTag       Tag = 0x0B
	HeapDumpTag        Tag = 0x0C
	HeapDumpSegmentTag Tag = 0x1C
	CPUSamplesTag      Tag = 0x0D
	ControlSettingsTag Tag = 0x0E
	HeapDumpEndTag     Tag = 0x2C
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

// 0 for non-array,
// 2 for object,
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
	NonArray BasicType = 0 //only in AllocSites, 0 means not an array, non-zero means an array of the given type
	Object   BasicType = 2
	Boolean  BasicType = 4  //1 byte
	Char     BasicType = 5  //2 byte
	Float    BasicType = 6  //4 bytes
	Double   BasicType = 7  //8 bytes
	Byte     BasicType = 8  //1 byte
	Short    BasicType = 9  //2 bytes
	Int      BasicType = 10 //4 bytes
	Long     BasicType = 11 //8 bytes
)

func (bt BasicType) GetSize() int32 {
	switch bt {
	case Boolean, Byte:
		return 1
	case Short, Char:
		return 2
	case Float, Int:
		return 4
	case Double, Long, Object:
		return 8
	}
	return 0
}

func (bt BasicType) GetName() string {
	switch bt {
	case Boolean:
		return "bool"
	case Char:
		return "char"
	case Float:
		return "float"
	case Double:
		return "double"
	case Byte:
		return "byte"
	case Short:
		return "short"
	case Int:
		return "int"
	case Long:
		return "long"
	case Object:
		return "object"
	}
	return "unknown"
}

type ID int64 // depends on the identifier size, but we will use int64 for now

// 0x01
type StringInUTF8 struct {
	StringId ID
	Bytes    []byte
}

// 0x02
type LoadClass struct {
	ClassSerialNumber      int32
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
	BitMaskSize            uint16
	CutoffRatio            int32
	TotalLiveBytes         int32
	TotalLiveInstances     int32
	TotalBytesAllocated    int64
	TotalInstanceAllocated int64
	NumberOfSites          int32
	Sites                  []Site
}

type Site struct {
	ArrayIndicator             BasicType
	ClassSerialNumber          int32
	StackTraceSerialNumber     int32
	NumberOfLiveBytes          int32
	NumberOfLiveInstances      int32
	NumberOfBytesAllocated     int32
	NumberOfInstancesAllocated int32
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
	NumberOfTraces       int32
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
	StackTraceDepth uint16
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

func (hdst HeapDumpSubTag) String() string {
	switch hdst {
	case RootUnknownTag:
		return "RootUnknown"
	case RootJNIGlobalTag:
		return "RootJNIGlobal"
	case RootJNILocalTag:
		return "RootJNILocal"
	case RootJavaFrameTag:
		return "RootJavaFrame"
	case RootNativeStackTag:
		return "RootNativeStack"
	case RootStickyClassTag:
		return "RootStickyClass"
	case RootThreadBlockTag:
		return "RootThreadBlock"
	case RootMonitorUsedTag:
		return "RootMonitorUsed"
	case RootThreadObjectTag:
		return "RootThreadObject"
	case ClassDumpTag:
		return "ClassDump"
	case InstanceDumpTag:
		return "InstanceDump"
	case ObjectArrayDumpTag:
		return "ObjectArrayDump"
	case PrimitiveArrayDumpTag:
		return "PrimitiveArrayDump"
	}
	return "Unknown"
}

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
	reserved_1               ID
	reserved_2               ID
	InstanceSize             int32 //in bytes
	ConstantPoolSize         uint16
	ConstantPool             []ConstantPoolRecord
	NumberOfStaticFields     uint16
	StaticFields             []StaticFieldRecord
	NumberOfInstanceFields   uint16
	InstanceFields           []InstanceFieldRecord
}

type ConstantPoolRecord struct {
	ConstantPoolIndex uint16
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