package hprof

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
)

func readID(reader io.Reader) ID {
	var id ID
	binary.Read(reader, binary.BigEndian, &id)
	return id
}

func readInt64(reader io.Reader) int64 {
	var i int64
	binary.Read(reader, binary.BigEndian, &i)
	return i
}

func readInt32(reader io.Reader) int32 {
	var i int32
	binary.Read(reader, binary.BigEndian, &i)
	return i
}

func readUint16(reader io.Reader) uint16 {
	var i uint16
	binary.Read(reader, binary.BigEndian, &i)
	return i
}

func readBasicType(reader io.Reader) BasicType {
	var bt BasicType
	binary.Read(reader, binary.BigEndian, &bt)
	return bt
}

func readArray(reader io.Reader, size int32) []byte {
	data := make([]byte, size)
	io.ReadFull(reader, data)
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

var flag = true

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

	// Get reader for the data
	record.DataReader = io.LimitReader(file, int64(record.Length))

	return record, nil
}

// Readers

func readStringInUTF8(reader io.Reader) {
	StringInUTF8 := StringInUTF8{
		StringID: readID(reader),
	}
	
	length := readInt32(reader);

	StringInUTF8.Bytes = readArray(reader, length);

	if err := SaveStringInUTF8(&StringInUTF8); err != nil {
		fmt.Errorf("Error saving StringInUTF8 to database: %v\n", err)
	}
}

func readLoadClass(reader io.Reader) {
	loadClass := LoadClass{
		ClassSerialNumber: readInt32(reader),
		ID:                readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		ClassNameStringID: readID(reader),
	}

	if err := SaveLoadClass(&loadClass); err != nil {
		fmt.Errorf("Error saving LoadClass to database: %v\n", err)
	}
}


func readUnloadClass(reader io.Reader) {
	unloadClass := UnloadClass{
		ClassSerialNumber: readInt32(reader),
	}

	if err := SaveUnloadClass(&unloadClass); err != nil {
		fmt.Errorf("Error saving UnloadClass to database: %v\n", err)
	}
}

func readStackFrame(reader io.Reader) {
	stackFrame := StackFrame{
		ID : readID(reader),
		MethodNameStringID: readID(reader),
		MethodSignatureStringID: readID(reader),
		SourceFileNameStringID: readID(reader),
		ClassSerialNumber: readInt32(reader),
		Flag: readInt32(reader),
	}

	if err := SaveStackFrame(&stackFrame); err != nil {
		fmt.Errorf("Error saving StackFrame to database: %v\n", err)
	}
}


func readStackTrace(reader io.Reader) {
	stackTrace := StackTrace{
		StackTraceSerialNumber: readInt32(reader),
		ThreadSerialNumber:     readInt32(reader),
	}

	framesCount := readInt32(reader)

	if err := SaveStackTrace(&stackTrace); err != nil {
		fmt.Errorf("Error saving StackTrace to database: %v\n", err)
		return
	}

	// Read the frames ID
	for i := int32(0); i < framesCount; i++ {
		frameId := readID(reader)
		
		if err := GetDB().
			Model(&StackFrame{}).
			Where("ID = ?", frameId).
			UpdateColumn("StackTraceSerialNumber", stackTrace.StackTraceSerialNumber).Error; err != nil {
			fmt.Errorf("Error updating StackFrame with frame ID %d: %v\n", frameId, err)
		}
	}
}

func readAllocSites(reader io.Reader) {
	allocSites := AllocSites{
		BitMaskSize: readUint16(reader),
		CutoffRatio: readInt32(reader),
		TotalLiveBytes:       readInt32(reader),
		TotalLiveInstances:   readInt32(reader),
		TotalBytesAllocated:  readInt64(reader),
		TotalInstanceAllocated: readInt64(reader),
	}

	numberOfSites := readInt32(reader)

	if err := SaveAllocSites(&allocSites); err != nil {
		fmt.Errorf("Error saving AllocSites to database: %v\n", err)
		return
	}

	// Read the sites
	for i := int32(0); i < numberOfSites; i++ {
		site := Site{
			AllocSitesID: allocSites.ID,
			ArrayIndicator: readBasicType(reader),
			ClassSerialNumber: readInt32(reader),
			StackTraceSerialNumber: readInt32(reader),
			NumberOfLiveBytes:       readInt32(reader),
			NumberOfLiveInstances:   readInt32(reader),
			NumberOfBytesAllocated:  readInt32(reader),
			NumberOfInstancesAllocated: readInt32(reader),
		}

		if err := SaveSite(&site); err != nil {
			fmt.Errorf("Error saving Site to database: %v\n", err)
			return
		}
	}
}


// func readHeapDump(data []byte) HeapDump {
// 	heapDump := HeapDump{}

// 	heapDump.data = make([]byte, len(data))
// 	copy(heapDump.data, data)

// 	return heapDump
// }


func readCPUSamples(data []byte) interface{} {
	cpuSamples := CPUSamples{
		TotalNumberOfSamples: int32(binary.BigEndian.Uint32(data[:4])),
		NumberOfTraces:       int32(binary.BigEndian.Uint32(data[4:8])),
	}

	// Read the traces
	cpuSamples.Traces = make([]struct {
		NumberOfSamples        int32
		StackTraceSerialNumber int32
	}, cpuSamples.NumberOfTraces)
	for i := 0; i < int(cpuSamples.NumberOfTraces); i++ {
		cpuSamples.Traces[i].NumberOfSamples = int32(binary.BigEndian.Uint32(data[8+i*8 : 8+(i+1)*8]))
		cpuSamples.Traces[i].StackTraceSerialNumber = int32(binary.BigEndian.Uint32(data[12+i*8 : 12+(i+1)*8]))
	}

	return cpuSamples
}

func readControlSettings(data []byte) interface{} {
	return ControlSettings{
		BitMask:         int32(binary.BigEndian.Uint32(data[:4])),
		StackTraceDepth: uint16(binary.BigEndian.Uint16(data[4:6])),
	}
}

func readRootUnknown(reader io.Reader) { 
	rootUnknown := RootUnknown{
		ID: readID(reader),
	}

	if err := SaveRootUnknown(&rootUnknown); err != nil {
		fmt.Errorf("Error saving RootUnknown to database: %v\n", err)
	}
}


func readRootJNIGlobal(reader io.Reader) {
	rootJNIGlobal := RootJNIGlobal{
		ID: readID(reader),
		JNIGlobalRef: readID(reader),
	}

	if err := SaveRootJNIGlobal(&rootJNIGlobal); err != nil {
		fmt.Errorf("Error saving RootJNIGlobal to database: %v\n", err)
	}
}

func readRootJNILocal(reader io.Reader) {
	rootJNILocal := RootJNILocal{
		ID: readID(reader),
		ThreadSerialNumber: readInt32(reader),
		FrameNumberInStackTrace: readInt32(reader),
	}

	if err := SaveRootJNILocal(&rootJNILocal); err != nil {
		fmt.Errorf("Error saving RootJNILocal to database: %v\n", err)
	}
}

func readRootJavaFrame(reader io.Reader) {
	rootJavaFrame := RootJavaFrame{
		ID: readID(reader),
		ThreadSerialNumber: readInt32(reader),
		FrameNumberInStackTrace: readInt32(reader),
	}

	if err := SaveRootJavaFrame(&rootJavaFrame); err != nil {
		fmt.Errorf("Error saving RootJavaFrame to database: %v\n", err)
	}
}

func readRootNativeStack(reader io.Reader) {
	rootNativeStack := RootNativeStack{
		ID: readID(reader),
		ThreadSerialNumber: readInt32(reader),
	}

	if err := SaveRootNativeStack(&rootNativeStack); err != nil {
		fmt.Errorf("Error saving RootNativeStack to database: %v\n", err)
	}
}


func readRootStickyClass(reader io.Reader) {
	rootStickyClass := RootStickyClass{
		ID: readID(reader),
	}

	if err := SaveRootStickyClass(&rootStickyClass); err != nil {
		fmt.Errorf("Error saving RootStickyClass to database: %v\n", err)
	}
}

func readRootThreadBlock(reader io.Reader) {
	rootThreadBlock := RootThreadBlock{
		ID: readID(reader),
		ThreadSerialNumber: readInt32(reader),
	}

	if err := SaveRootThreadBlock(&rootThreadBlock); err != nil {
		fmt.Errorf("Error saving RootThreadBlock to database: %v\n", err)
	}
}


func readRootMonitorUsed(reader io.Reader) {
	rootMonitorUsed := RootMonitorUsed{
		ID: readID(reader),
	}

	if err := SaveRootMonitorUsed(&rootMonitorUsed); err != nil {
		fmt.Errorf("Error saving RootMonitorUsed to database: %v\n", err)
	}
}


func readRootThreadObject(reader io.Reader) {
	rootThreadObject := RootThreadObject{
		ID: readID(reader),
		ThreadSerialNumber: readInt32(reader),
		StackTraceSerialNumber: readInt32(reader),
	}

	if err := SaveRootThreadObject(&rootThreadObject); err != nil {
		fmt.Errorf("Error saving RootThreadObject to database: %v\n", err)
	}
}


func readClassDump(reader io.Reader) {
	classDump := ClassDump{
		ID: readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		SuperClassObjectID: readID(reader),
		ClassLoaderObjectID: readID(reader),
		SignersObjectID: readID(reader),
		ProtectionDomainObjectID: readID(reader),
		Reserved1: readID(reader),
		Reserved2: readID(reader),
		InstanceSize: readInt32(reader),
	}

	if err := SaveClassDump(&classDump); err != nil {
		fmt.Errorf("Error saving ClassDump to database: %v\n", err)
		return
	}

	// Read the constant pool
	constantPoolSize := readUint16(reader)
	for i := 0; i < int(constantPoolSize); i++ {
		constantPoolRecord := ConstantPoolRecord{
			ClassDumpID: classDump.ID,
			ConstantPoolIndex: readUint16(reader),
			Type: readBasicType(reader),
		}
		
		constantPoolRecord.Value = readArray(reader, constantPoolRecord.Type.GetSize())

		if err := SaveConstantPoolRecord(&constantPoolRecord); err != nil {
			fmt.Errorf("Error saving ConstantPoolRecord to database: %v\n", err)
			return
		}
	}

	// Read the static fields
	numberOfStaticFields := readUint16(reader)
	for i := 0; i < int(numberOfStaticFields); i++ {
		staticFieldRecord := StaticFieldRecord{
			ClassDumpID: classDump.ID,
			StaticFieldNameStringID: readID(reader),
			Type: readBasicType(reader),
		}
		
		staticFieldRecord.Value = readArray(reader, staticFieldRecord.Type.GetSize())
	
		if err := SaveStaticFieldRecord(&staticFieldRecord); err != nil {
			fmt.Errorf("Error saving StaticFieldRecord to database: %v\n", err)
			return
		}
	}


	// Read the instance fields
	numberOfInstanceFields := readUint16(reader)
	for i := 0; i < int(numberOfInstanceFields); i++ {
		instanceFieldRecord := InstanceFieldRecord{
			ClassDumpID: classDump.ID,
			FieldNameStringID: readID(reader),
			Type: readBasicType(reader),
		}
		
		if err := SaveInstanceFieldRecord(&instanceFieldRecord); err != nil {
			fmt.Errorf("Error saving InstanceFieldRecord to database: %v\n", err)
			return
		}
	}
}


// After all we need to parse Data from InstanceDump, because we dont know the type of each value in Data. 
// ClassDump.InstanceFileds will help us to understand the type of each value in Data.
func readInstanceDump(reader io.Reader) {
	instanceDump := InstanceDump{
		ID: readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		ClassObjectID: readID(reader),
		NumberOfBytes: readInt32(reader),
	}

	instanceDump.Data = readArray(reader, instanceDump.NumberOfBytes)

	if err := SaveInstanceDump(&instanceDump); err != nil {
		fmt.Errorf("Error saving InstanceDump to database: %v\n", err)
		return
	}
}


func readObjectArrayDump(reader io.Reader) {
	objectArrayDump := ObjectArrayDump{
		ID: readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		NumberOfElements: readInt32(reader),
		ArrayClassObjectID: readID(reader),
	}

	if err := SaveObjectArrayDump(&objectArrayDump); err != nil {
		fmt.Errorf("Error saving ObjectArrayDump to database: %v\n", err)
		return
	}

	for i := int32(0); i < objectArrayDump.NumberOfElements; i++ {
		arrayElement := ObjectArrayElement{
			ObjectArrayDumpID: objectArrayDump.ID,
			Index: i,
			InstanceDumpID: readID(reader),
		}

		if err := SaveObjectArrayElement(&arrayElement); err != nil {
			fmt.Errorf("Error saving ObjectArrayElement to database: %v\n", err)
			return
		}
	}
}


func readPrimitiveArrayDump(reader io.Reader) {
	primitiveArrayDump := PrimitiveArrayDump{
		ID: readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		NumberOfElements: readInt32(reader),
		Type: readBasicType(reader),
	}

	if err := SavePrimitiveArrayDump(&primitiveArrayDump); err != nil {
		fmt.Errorf("Error saving PrimitiveArrayDump to database: %v\n", err)
		return
	}

	elementSize := primitiveArrayDump.Type.GetSize()
	for i := int32(0); i < primitiveArrayDump.NumberOfElements; i++ {
		element := PrimitiveArrayElement{
			PrimitiveArrayDumpID: primitiveArrayDump.ID,
			Index: i,
		}

		element.Value = readArray(reader, elementSize)

		if err := SavePrimitiveArrayElement(&element); err != nil {
			fmt.Errorf("Error saving PrimitiveArrayElement to database: %v\n", err)
			return
		}
	}

}

// var (
// 	IDtoStringInUTF8                = make(map[ID]string)
// 	IDtoSizeClassDump               = make(map[ID]int64)
// 	ClassObjectIdToClassNameID      = make(map[ID]ID)
// 	IDtoStackFrame                  = make(map[ID]StackFrame)
// 	StackTraceIdToStackFrameIds     = make(map[int32][]ID)
// 	ClassObjectIdToCountInstances   = make(map[ID]int32)
// 	IDtoClassLoaderID               = make(map[ID]ID)
// 	ObjectIdToInstanceDump          = make(map[ID]InstanceDump)
// 	ObjectIdToInstanceDumpMap       = make(map[ID]InstanceDump)
// 	ClassObjectIdToClassDumpMap     = make(map[ID]ClassDump)
// 	ObjectIdToObjectArrayDumpMap    = make(map[ID]ObjectArrayDump)
// 	ObjectIdToPrimitiveArrayDumpMap = make(map[ID]PrimitiveArrayDump)
// 	StringUtf8Map                   = make(map[ID]StringInUTF8)
// 	ClassObjectIdToLoadClassMap     = make(map[ID]LoadClass)
// )

const ArrayHeaderSize = int32(16)

func ParseHeapDump(heapDumpFile *os.File) {
	type readerFunction func(io.Reader)

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

	// Read the header
	header := readHeader(heapDumpFile)
	fmt.Printf("Header: %+v\n", header)

	// Read records
	for {
		record, err := readRecord(heapDumpFile)
		if err == io.EOF {
			fmt.Printf("Reached end of file.\n\n\n")
			break
		} else if err != nil {
			fmt.Errorf("Error reading record: %v\n", err)
			break
		}

		switch record.Tag {
		case StringUtf8Tag:
			readStringInUTF8(record.DataReader)
		case LoadClassTag:
			readLoadClass(record.DataReader)
		case UnloadClassTag:
			readUnloadClass(record.DataReader)
		case StackFrameTag:
			readStackFrame(record.DataReader)
		case StackTraceTag:
			readStackTrace(record.DataReader)
		case AllocSitesTag:
			readAllocSites(record.DataReader)
		case HeapDumpTag, HeapDumpSegmentTag:
			reader := record.DataReader
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
					readerFunction(reader)
				} else {
					fmt.Errorf("Unknown sub tag: %d\n", subTag)
					break
				}
			}
		}
	}
}

type AnalyzeResult struct {
	Header string
	Body   []string
}

func (result AnalyzeResult) Print() {
	fmt.Println("==================================================")
	fmt.Print(result.Header)
	for _, line := range result.Body {
		fmt.Print(line)
	}
	fmt.Println("==================================================")
}

func (result AnalyzeResult) ToHTML() string {
	var buf bytes.Buffer
	buf.WriteString("<h1>" + result.Header + "</h1>")
	buf.WriteString("<ul>")
	for _, line := range result.Body {
		buf.WriteString("<li>" + line + "</li>")
	}
	buf.WriteString("</ul>")
	return buf.String()
}
	

func PrintSizeClasses(max int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: fmt.Sprintf("\n\nTop %d classes by size\n", max), 
		Body:   make([]string, max),
	}

	type IdSize struct {
		id   ID
		size int64
	}

	pairs := make([]IdSize, 0, len(IDtoSizeClassDump))
	for id, size := range IDtoSizeClassDump {
		pairs = append(pairs, IdSize{id, size})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].size > pairs[j].size
	})

	for i, p := range pairs {
		if i == max {
			break
		}
		result.Body[i] = fmt.Sprintf("%d. Class ID: %d, Size: %d, Name: %s\n", (i + 1), p.id, p.size, IDtoStringInUTF8[ClassObjectIdToClassNameID[p.id]])
	}

	return result
}

func PrintCountInstances(max int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: fmt.Sprintf("\n\nTop %d classes by instance count\n", max), 
		Body:   make([]string, max),
	}
	type IdCount struct {
		id    ID
		count int32
	}

	countPairs := make([]IdCount, 0, len(ClassObjectIdToCountInstances))
	for id, count := range ClassObjectIdToCountInstances {
		countPairs = append(countPairs, IdCount{id, count})
	}

	sort.Slice(countPairs, func(i, j int) bool {
		return countPairs[i].count > countPairs[j].count
	})

	for i, p := range countPairs {
		if i == max {
			break
		}
		result.Body[i] = fmt.Sprintf("%d. Class ID: %d, Count: %d, Name: %s\n", (i + 1), p.id, p.count, IDtoStringInUTF8[ClassObjectIdToClassNameID[p.id]])
	}
	return result
}

func PrintObjectLoadersInfo(max int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: "\n\nObject loaders info\n",
		Body:   make([]string, 0),
	}

	loaderObjectsMap := make(map[ID]([]ID))
	for object, loader := range IDtoClassLoaderID {
		loaderObjectsMap[loader] = append(loaderObjectsMap[loader], object)
	}

	fmt.Printf("\n\nObject loaders info\n")
	for loader, classes := range loaderObjectsMap {
		loaderName := getNameByClassObjectId(ObjectIdToInstanceDumpMap[loader].ClassObjectId)
		if loader == 0 {
			loaderName = "Bootstrap ClassLoader (System)"
		}
		result.Body = append(result.Body, fmt.Sprintf("Loader ID: %d, Name: %s, Number of objects: %d\n", loader, loaderName, len(classes)))
		for i, obj := range classes {
			if i == max {
				result.Body = append(result.Body, "\t\t...\n")
				break
			}
			result.Body = append(result.Body, fmt.Sprintf("\t\tClass ID: %d, Name: %s\n", obj, getNameByClassObjectId(obj)))
		}
	}
	return result
}

func PrintFullClassSize(max int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: fmt.Sprintf("\n\nTop %d classes by full size (with all depends object)\n", max),
		Body:   make([]string, max),
	}

	classStatsMap := CalculateClassSizes()

	type IdStats struct {
		id   ID
		stat ClassStats
	}

	pairs := make([]IdStats, 0, len(classStatsMap))
	for id, stat := range classStatsMap {
		pairs = append(pairs, IdStats{id, stat})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].stat.TotalSize > pairs[j].stat.TotalSize
	})

	for i, p := range pairs {
		if i == max {
			break
		}
		result.Body[i] = fmt.Sprintf("%d. Class ID: %d, Size: %d, Name: %s\n", (i + 1), p.id, p.stat.TotalSize, p.stat.ClassName)
	}
	return result
}

func PrintArrayInfo(max int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: fmt.Sprintf("\n\nTop %d arrays by size\n", max),
		Body:   make([]string, max),
	}
	type nameSize struct {
		name string
		size int32
	}

	nameSizeMap := make(map[string]int32)

	for _, array := range ObjectIdToObjectArrayDumpMap {
		name := getNameByClassObjectId(array.ArrayClassObjectId)
		nameSizeMap[string(name)] += getObjectSize(array)
	}

	for _, array := range ObjectIdToPrimitiveArrayDumpMap {
		name := array.ElementType.GetName()
		nameSizeMap[string(name)] += getObjectSize(array)
	}

	pairs := make([]nameSize, 0, len(nameSizeMap))
	for name, size := range nameSizeMap {
		pairs = append(pairs, nameSize{name, size})
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].size > pairs[j].size
	})

	for i, p := range pairs {
		if i == max {
			break
		}
		result.Body[i] = fmt.Sprintf("%d. Array: %s, Size: %d\n", (i + 1), p.name, p.size)
	}
	return result
}

func getNameByClassObjectId(id ID) string {
	return IDtoStringInUTF8[ClassObjectIdToClassNameID[id]]
}

type ClassStats struct {
	ClassName string
	TotalSize int32
}

func CalculateClassSizes() map[ID]ClassStats {
	result := make(map[ID]ClassStats)

	for classObjectId, classDump := range ClassObjectIdToClassDumpMap {
		visited := make(map[ID]bool)
		queue := make([]ID, 0)
		var totalSize int32

		for objId, instance := range ObjectIdToInstanceDumpMap {
			if instance.ClassObjectId == classObjectId {
				if visited[objId] {
					continue
				}
				visited[objId] = true
				queue = append(queue, objId)
				totalSize += getObjectSize(instance)
			}
		}

		for _, sf := range classDump.StaticFields {
			if sf.Type == Object {
				refId := ID(binary.BigEndian.Uint64(sf.Value))
				if visited[refId] {
					continue
				}
				visited[refId] = true
				queue = append(queue, refId)
				if obj := getObject(refId); obj != nil {
					totalSize += getObjectSize(obj)
				}
			} else {
				totalSize += sf.Type.GetSize()
			}
		}

		for len(queue) > 0 {
			currentId := queue[0]
			queue = queue[1:]

			obj := getObject(currentId)
			if obj == nil {
				continue
			}

			for _, refId := range getReferences(obj) {
				if visited[refId] {
					continue
				}
				visited[refId] = true
				queue = append(queue, refId)
				if refObj := getObject(refId); refObj != nil {
					totalSize += getObjectSize(refObj)
				}
			}
		}

		className := getNameByClassObjectId(classObjectId)

		result[classObjectId] = ClassStats{
			ClassName: className,
			TotalSize: totalSize,
		}
	}

	return result
}

func getObject(objectId ID) interface{} {
	if obj, ok := ObjectIdToInstanceDumpMap[objectId]; ok {
		return obj
	}
	if obj, ok := ObjectIdToObjectArrayDumpMap[objectId]; ok {
		return obj
	}
	if obj, ok := ObjectIdToPrimitiveArrayDumpMap[objectId]; ok {
		return obj
	}
	return nil
}

func getReferences(obj interface{}) []ID {
	switch v := obj.(type) {
	case InstanceDump:
		return parseInstanceReferences(v)
	case ObjectArrayDump:
		return v.Elements
	}
	return nil
}

func parseInstanceReferences(instance InstanceDump) []ID {
	currentClassId := instance.ClassObjectId
	var allFields []InstanceFieldRecord

	for {
		classDump, ok := ClassObjectIdToClassDumpMap[currentClassId]
		if !ok {
			break
		}
		allFields = append(classDump.InstanceFields, allFields...)
		currentClassId = classDump.SuperClassObjectId
		if currentClassId == 0 {
			break
		}
	}

	var refs []ID
	offset := 0
	for _, field := range allFields {
		if field.Type == Object {
			start := offset
			end := offset + 8
			if end > len(instance.InstanceFieldValues) {
				break
			}
			refId := ID(binary.BigEndian.Uint64(instance.InstanceFieldValues[start:end]))
			refs = append(refs, refId)
		}
		offset += int(field.Type.GetSize())
	}
	return refs
}

func getObjectSize(obj interface{}) int32 {
	switch v := obj.(type) {
	case InstanceDump:
		return v.NumberOfBytes
	case ObjectArrayDump:
		return ArrayHeaderSize + v.NumberOfElements*8
	case PrimitiveArrayDump:
		return ArrayHeaderSize + v.NumberOfElements*v.ElementType.GetSize()
	}
	return 0
}
