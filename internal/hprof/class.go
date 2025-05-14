package hprof

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
)

func readID(reader *bytes.Reader) ID {
	var id ID
	binary.Read(reader, binary.BigEndian, &id)
	return id
}

func readInt64(reader *bytes.Reader) uint64 {
	var i uint64
	binary.Read(reader, binary.BigEndian, &i)
	return i
}

func readInt32(reader *bytes.Reader) int32 {
	var i int32
	binary.Read(reader, binary.BigEndian, &i)
	return i
}

func readUint16(reader *bytes.Reader) uint16 {
	var i uint16
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
		ClassSerialNumber:      int32(binary.BigEndian.Uint32(data[:4])),
		ClassObjectId:          ID(binary.BigEndian.Uint64(data[4:12])),
		StackTraceSerialNumber: int32(binary.BigEndian.Uint32(data[12:16])),
		ClassNameStringId:      ID(binary.BigEndian.Uint64(data[16:24])),
	}
}

func readUnloadClass(data []byte) interface{} {
	return UnloadClass{
		ClassSerialNumber: int32(binary.BigEndian.Uint32(data[:4])),
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

func readAllocSites(data []byte) interface{} {
	allocSites := AllocSites{
		BitMaskSize:            uint16(binary.BigEndian.Uint16(data[:2])),
		CutoffRatio:            int32(binary.BigEndian.Uint32(data[2:6])),
		TotalLiveBytes:         int32(binary.BigEndian.Uint32(data[6:10])),
		TotalLiveInstances:     int32(binary.BigEndian.Uint32(data[10:14])),
		TotalBytesAllocated:    int64(binary.BigEndian.Uint64(data[14:22])),
		TotalInstanceAllocated: int64(binary.BigEndian.Uint64(data[22:30])),
		NumberOfSites:          int32(binary.BigEndian.Uint32(data[30:34])),
	}

	// Read the sites
	allocSites.Sites = make([]Site, allocSites.NumberOfSites)
	for i := 0; i < int(allocSites.NumberOfSites); i++ {
		site := Site{
			ArrayIndicator:             BasicType(data[34+i*25]),
			ClassSerialNumber:          int32(binary.BigEndian.Uint32(data[35+i*25 : 39+i*25])),
			StackTraceSerialNumber:     int32(binary.BigEndian.Uint32(data[39+i*25 : 43+i*25])),
			NumberOfLiveBytes:          int32(binary.BigEndian.Uint32(data[43+i*25 : 47+i*25])),
			NumberOfLiveInstances:      int32(binary.BigEndian.Uint32(data[47+i*25 : 51+i*25])),
			NumberOfBytesAllocated:     int32(binary.BigEndian.Uint32(data[51+i*25 : 55+i*25])),
			NumberOfInstancesAllocated: int32(binary.BigEndian.Uint32(data[55+i*25 : 59+i*25])),
		}
		allocSites.Sites[i] = site
	}

	return allocSites
}

func readHeapDump(data []byte) HeapDump {
	heapDump := HeapDump{}

	heapDump.data = make([]byte, len(data))
	copy(heapDump.data, data)

	return heapDump
}

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
		reserved_1:               readID(reader),
		reserved_2:               readID(reader),
		InstanceSize:             readInt32(reader),
	}

	// Read the constant pool
	classDump.ConstantPoolSize = readUint16(reader)
	classDump.ConstantPool = make([]ConstantPoolRecord, classDump.ConstantPoolSize)
	for i := 0; i < int(classDump.ConstantPoolSize); i++ {
		constantPoolRecord := ConstantPoolRecord{
			ConstantPoolIndex: readUint16(reader),
			Type:              readBasicType(reader),
		}
		constantPoolRecord.Value = readArray(reader, constantPoolRecord.Type.GetSize())
		classDump.ConstantPool[i] = constantPoolRecord
	}

	// Read the static fields
	classDump.NumberOfStaticFields = readUint16(reader)
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
	classDump.NumberOfInstanceFields = readUint16(reader)
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

var (
	IDtoStringInUTF8                = make(map[ID]string)
	IDtoSizeClassDump               = make(map[ID]int64)
	ClassObjectIdToClassNameID      = make(map[ID]ID)
	IDtoStackFrame                  = make(map[ID]StackFrame)
	StackTraceIdToStackFrameIds     = make(map[int32][]ID)
	ClassObjectIdToCountInstances   = make(map[ID]int32)
	IDtoClassLoaderID               = make(map[ID]ID)
	ObjectIdToInstanceDump          = make(map[ID]InstanceDump)
	ObjectIdToInstanceDumpMap       = make(map[ID]InstanceDump)
	ClassObjectIdToClassDumpMap     = make(map[ID]ClassDump)
	ObjectIdToObjectArrayDumpMap    = make(map[ID]ObjectArrayDump)
	ObjectIdToPrimitiveArrayDumpMap = make(map[ID]PrimitiveArrayDump)
	StringUtf8Map                   = make(map[ID]StringInUTF8)
	ClassObjectIdToLoadClassMap     = make(map[ID]LoadClass)
)

const ArrayHeaderSize = int32(16)

func ParseHeapDump(heapDumpFile *os.File) {

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
			stringInUTF8 := readStringInUTF8(record.Data).(StringInUTF8)
			IDtoStringInUTF8[stringInUTF8.StringId] = string(stringInUTF8.Bytes)
			StringUtf8Map[stringInUTF8.StringId] = stringInUTF8
		case LoadClassTag:
			loadClass := readLoadClass(record.Data).(LoadClass)
			ClassObjectIdToClassNameID[loadClass.ClassObjectId] = loadClass.ClassNameStringId
			ClassObjectIdToLoadClassMap[loadClass.ClassObjectId] = loadClass
		case UnloadClassTag:
			unloadClass := readUnloadClass(record.Data).(UnloadClass)
			_ = unloadClass
		case StackFrameTag:
			stackFrame := readStackFrame(record.Data).(StackFrame)
			IDtoStackFrame[stackFrame.FrameId] = stackFrame
		case StackTraceTag:
			stackTrace := readStackTrace(record.Data).(StackTrace)
			StackTraceIdToStackFrameIds[stackTrace.StackTraceSerialNumber] = stackTrace.FramesID
		case AllocSitesTag:
			allocSites := readAllocSites(record.Data).(AllocSites)
			_ = allocSites
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
					fmt.Errorf("Error reading sub tag: %v\n", err)
					break
				}

				if readerFunction, ok := subTagFuncMap[subTag]; ok {
					switch subTag {
					case InstanceDumpTag:
						instanceDump := readerFunction(reader).(InstanceDump)
						IDtoSizeClassDump[instanceDump.ClassObjectId] += int64(instanceDump.NumberOfBytes)
						ClassObjectIdToCountInstances[instanceDump.ClassObjectId]++
						ObjectIdToInstanceDumpMap[instanceDump.ObjectId] = instanceDump
					case ClassDumpTag:
						classDump := readerFunction(reader).(ClassDump)
						IDtoSizeClassDump[classDump.ClassObjectId] += int64(classDump.InstanceSize)
						IDtoClassLoaderID[classDump.ClassObjectId] = classDump.ClassLoaderObjectId
						ClassObjectIdToClassDumpMap[classDump.ClassObjectId] = classDump
					case ObjectArrayDumpTag:
						objectArrayDump := readerFunction(reader).(ObjectArrayDump)
						ObjectIdToObjectArrayDumpMap[objectArrayDump.ArrayObjectId] = objectArrayDump
					case PrimitiveArrayDumpTag:
						primitiveArrayDump := readerFunction(reader).(PrimitiveArrayDump)
						ObjectIdToPrimitiveArrayDumpMap[primitiveArrayDump.ArrayObjectId] = primitiveArrayDump
					default:
						_ = readerFunction(reader)
					}
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
