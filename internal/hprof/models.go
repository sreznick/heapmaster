package hprof

import "io"

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
	Tag        Tag
	Time       int32
	Length     int32
	DataReader io.Reader
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
// type HeapDump struct {
// 	data []byte
// }

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

// Model definitions

// 0x01
type StringInUTF8 struct {
	StringID ID     `gorm:"primaryKey;column:StringID"`
	Bytes    []byte `gorm:"column:Bytes"`
}

func (StringInUTF8) TableName() string { return "StringInUTF8" }

// 0x02
type LoadClass struct {
	ClassSerialNumber      int32 `gorm:"primaryKey;column:ClassSerialNumber"`
	ClassObjectID          ID    `gorm:"column:ClassObjectID"`
	StackTraceSerialNumber int32 `gorm:"column:StackTraceSerialNumber"`
	ClassNameStringID      ID    `gorm:"column:ClassNameStringID"`
}

func (LoadClass) TableName() string { return "LoadClass" }

// 0x03
type UnloadClass struct {
	ClassSerialNumber int32 `gorm:"primaryKey;column:ClassSerialNumber"`
}

func (UnloadClass) TableName() string { return "UnloadClass" }

// 0x04
type StackFrame struct {
	ID                      ID    `gorm:"primaryKey;column:ID"`
	MethodNameStringID      ID    `gorm:"column:MethodNameStringID"`
	MethodSignatureStringID ID    `gorm:"column:MethodSignatureStringID"`
	SourceFileNameStringID  ID    `gorm:"column:SourceFileNameStringID"`
	ClassSerialNumber       int32 `gorm:"column:ClassSerialNumber"`
	// > 0 line number
	// 0 no line information
	// -1 unknown location
	// -2 compiled method
	// -3 native method
	Flag int32 `gorm:"column:Flag"`

	StackTraceSerialNumber int32 `gorm:"column:StackTraceSerialNumber"`
}

func (StackFrame) TableName() string { return "StackFrame" }

// 0x05
type StackTrace struct {
	StackTraceSerialNumber int32 `gorm:"primaryKey;column:StackTraceSerialNumber"`
	ThreadSerialNumber     int32 `gorm:"column:ThreadSerialNumber"`
}

func (StackTrace) TableName() string { return "StackTrace" }

// 0x06
type AllocSites struct {
	ID ID `gorm:"primaryKey;column:ID;autoIncrement"`
	// 0x1 incremental / complete
	// 0x2 sorted by allocation / line
	// 0x4 whether to force GC
	BitMaskSize            uint16 `gorm:"column:BitMaskSize"`
	CutoffRatio            int32  `gorm:"column:CutoffRatio"`
	TotalLiveBytes         int32  `gorm:"column:TotalLiveBytes"`
	TotalLiveInstances     int32  `gorm:"column:TotalLiveInstances"`
	TotalBytesAllocated    int64  `gorm:"column:TotalBytesAllocated"`
	TotalInstanceAllocated int64  `gorm:"column:TotalInstanceAllocated"`
}

func (AllocSites) TableName() string { return "AllocSites" }

type Site struct {
	ID                         ID        `gorm:"primaryKey;column:ID;autoIncrement"`
	AllocSitesID               ID        `gorm:"column:AllocSitesID"`
	ArrayIndicator             BasicType `gorm:"column:ArrayIndicator"`
	ClassSerialNumber          int32     `gorm:"column:ClassSerialNumber"`
	StackTraceSerialNumber     int32     `gorm:"column:StackTraceSerialNumber"`
	NumberOfLiveBytes          int32     `gorm:"column:NumberOfLiveBytes"`
	NumberOfLiveInstances      int32     `gorm:"column:NumberOfLiveInstances"`
	NumberOfBytesAllocated     int32     `gorm:"column:NumberOfBytesAllocated"`
	NumberOfInstancesAllocated int32     `gorm:"column:NumberOfInstancesAllocated"`
}

func (Site) TableName() string { return "Site" }

// Heap Dump Segment roots

// 0xFF
type RootUnknown struct {
	ID ID `gorm:"primaryKey;column:ID"`
}

func (RootUnknown) TableName() string { return "RootUnknown" }

// 0x01
type RootJNIGlobal struct {
	ID           ID `gorm:"primaryKey;column:ID"`
	JNIGlobalRef ID `gorm:"column:JNIGlobalRef"`
}

func (RootJNIGlobal) TableName() string { return "RootJNIGlobal" }

// 0x02
type RootJNILocal struct {
	ID                      ID    `gorm:"primaryKey;column:ID"`
	ThreadSerialNumber      int32 `gorm:"column:ThreadSerialNumber"`
	FrameNumberInStackTrace int32 `gorm:"column:FrameNumberInStackTrace"`
}

func (RootJNILocal) TableName() string { return "RootJNILocal" }

// 0x03
type RootJavaFrame struct {
	ID                      ID    `gorm:"primaryKey;column:ID;autoIncrement"`
	ObjectID                ID    `gorm:"column:ObjectID"`
	ThreadSerialNumber      int32 `gorm:"column:ThreadSerialNumber"`
	FrameNumberInStackTrace int32 `gorm:"column:FrameNumberInStackTrace"`
}

func (RootJavaFrame) TableName() string { return "RootJavaFrame" }

// 0x04
type RootNativeStack struct {
	ID                 ID    `gorm:"primaryKey;column:ID"`
	ThreadSerialNumber int32 `gorm:"column:ThreadSerialNumber"`
}

func (RootNativeStack) TableName() string { return "RootNativaStack" }

// 0x05
type RootStickyClass struct {
	ID ID `gorm:"primaryKey;column:ID"`
}

func (RootStickyClass) TableName() string { return "RootStickyClass" }

// 0x06
type RootThreadBlock struct {
	ID                 ID    `gorm:"primaryKey;column:ID"`
	ThreadSerialNumber int32 `gorm:"column:ThreadSerialNumber"`
}

func (RootThreadBlock) TableName() string { return "RootThreadBlock" }

// 0x07
type RootMonitorUsed struct {
	ID ID `gorm:"primaryKey;column:ID"`
}

func (RootMonitorUsed) TableName() string { return "RootMonitorUsed" }

// 0x08
type RootThreadObject struct {
	ID                     ID    `gorm:"primaryKey;column:ID"`
	ThreadSerialNumber     int32 `gorm:"column:ThreadSerialNumber"`
	StackTraceSerialNumber int32 `gorm:"column:StackTraceSerialNumber"`
}

func (RootThreadObject) TableName() string { return "RootThreadObject" }

// 0x20
type ClassDump struct {
	ID                       ID    `gorm:"primaryKey;column:ID"`
	StackTraceSerialNumber   int32 `gorm:"column:StackTraceSerialNumber"`
	SuperClassObjectID       ID    `gorm:"column:SuperClassObjectID"`
	ClassLoaderObjectID      ID    `gorm:"column:ClassLoaderObjectID"`
	SignersObjectID          ID    `gorm:"column:SignersObjectID"`
	ProtectionDomainObjectID ID    `gorm:"column:ProtectionDomainObjectID"`
	Reserved1                ID    `gorm:"column:Reserved1"`
	Reserved2                ID    `gorm:"column:Reserved2"`
	InstanceSize             int32 `gorm:"column:InstanceSize"`
}

func (ClassDump) TableName() string { return "ClassDump" }

type ConstantPoolRecord struct {
	ID                ID        `gorm:"primaryKey;column:ID;autoIncrement"`
	ClassDumpID       ID        `gorm:"column:ClassDumpID"`
	ConstantPoolIndex uint16    `gorm:"column:ConstantPoolIndex"`
	Type              BasicType `gorm:"column:Type"`
	Value             []byte    `gorm:"column:Value"`
}

func (ConstantPoolRecord) TableName() string { return "ConstantPoolRecord" }

type StaticFieldRecord struct {
	ID                      ID        `gorm:"primaryKey;column:ID;autoIncrement"`
	ClassDumpID             ID        `gorm:"column:ClassDumpID"`
	StaticFieldNameStringID ID        `gorm:"column:StaticFieldNameStringID"`
	Type                    BasicType `gorm:"column:Type"`
	Value                   []byte    `gorm:"column:Value"`
}

func (StaticFieldRecord) TableName() string { return "StaticFieldRecord" }

type InstanceFieldRecord struct {
	ID                ID        `gorm:"primaryKey;column:ID;autoIncrement"`
	ClassDumpID       ID        `gorm:"column:ClassDumpID"`
	FieldNameStringID ID        `gorm:"column:FieldNameStringID"`
	Type              BasicType `gorm:"column:Type"`
}

func (InstanceFieldRecord) TableName() string { return "InstanceFieldRecord" }

// 0x21
type InstanceDump struct {
	ID                     ID     `gorm:"primaryKey;column:ID"`
	StackTraceSerialNumber int32  `gorm:"column:StackTraceSerialNumber"`
	ClassObjectID          ID     `gorm:"column:ClassObjectID"`
	NumberOfBytes          int32  `gorm:"column:NumberOfBytes"`
	Data                   []byte `gorm:"column:Data"` // ATTENTION ClassDump should be used to decode this
}

func (InstanceDump) TableName() string { return "InstanceDump" }

type InstanceFieldValues struct {
	ID             ID        `gorm:"primaryKey;column:ID;autoIncrement"`
	InstanceDumpID ID        `gorm:"column:InstanceDumpID"`
	Index          int32     `gorm:"column:Index"` // This is the index of the field in the class dump
	Type           BasicType `gorm:"column:Type"`
	Value          []byte    `gorm:"column:Value"`
}

func (InstanceFieldValues) TableName() string { return "InstanceFieldValues" }

// 0x22
type ObjectArrayDump struct {
	ID                     ID    `gorm:"primaryKey;column:ID"`
	StackTraceSerialNumber int32 `gorm:"column:StackTraceSerialNumber"`
	NumberOfElements       int32 `gorm:"column:NumberOfElements"`
	ArrayClassObjectID     ID    `gorm:"column:ArrayClassObjectID"`
}

func (ObjectArrayDump) TableName() string { return "ObjectArrayDump" }

type ObjectArrayElement struct {
	ID                ID    `gorm:"primaryKey;column:ID;autoIncrement"`
	ObjectArrayDumpID ID    `gorm:"column:ObjectArrayDumpID"`
	Index             int32 `gorm:"column:Index"`
	InstanceDumpID    ID    `gorm:"column:InstanceDumpID"`
}

func (ObjectArrayElement) TableName() string { return "ObjectArrayElement" }

// 0x23
type PrimitiveArrayDump struct {
	ID                     ID        `gorm:"primaryKey;column:ID"`
	StackTraceSerialNumber int32     `gorm:"column:StackTraceSerialNumber"`
	NumberOfElements       int32     `gorm:"column:NumberOfElements"`
	Type                   BasicType `gorm:"column:Type"`
}

func (PrimitiveArrayDump) TableName() string { return "PrimitiveArrayDump" }

type PrimitiveArrayElement struct {
	ID                   ID     `gorm:"primaryKey;column:ID;autoIncrement"`
	PrimitiveArrayDumpID ID     `gorm:"column:PrimitiveArrayDumpID"`
	Index                int32  `gorm:"column:Index"`
	Value                []byte `gorm:"column:Value"`
}

func (PrimitiveArrayElement) TableName() string { return "PrimitiveArrayElement" }
