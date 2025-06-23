package hprof

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
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
	if size < 0 {
		fmt.Printf("Error: negative array size %d\n", size)
		return nil
	}
	if size == 0 {
		return []byte{}
	}
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

func readStringInUTF8(reader io.Reader, length int32) {
	StringInUTF8 := StringInUTF8{
		StringID: readID(reader),
	}

	// The length includes the StringID (8 bytes), so subtract that
	dataLength := length - 8
	if dataLength < 0 {
		fmt.Printf("Error: invalid string data length %d\n", dataLength)
		return
	}

	StringInUTF8.Bytes = readArray(reader, dataLength)

	if err := SaveStringInUTF8(&StringInUTF8); err != nil {
		fmt.Printf("Error saving StringInUTF8 to database: %v\n", err)
	}
}

func readLoadClass(reader io.Reader) {
	loadClass := LoadClass{
		ClassSerialNumber:      readInt32(reader),
		ClassObjectID:          readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		ClassNameStringID:      readID(reader),
	}

	if err := SaveLoadClass(&loadClass); err != nil {
		fmt.Printf("Error saving LoadClass to database: %v\n", err)
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
		ID:                      readID(reader),
		MethodNameStringID:      readID(reader),
		MethodSignatureStringID: readID(reader),
		SourceFileNameStringID:  readID(reader),
		ClassSerialNumber:       readInt32(reader),
		Flag:                    readInt32(reader),
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
			Where("\"ID\" = ?", frameId).
			UpdateColumn("\"StackTraceSerialNumber\"", stackTrace.StackTraceSerialNumber).Error; err != nil {
			fmt.Errorf("Error updating StackFrame with frame ID %d: %v\n", frameId, err)
		}
	}
}

func readAllocSites(reader io.Reader) {
	allocSites := AllocSites{
		BitMaskSize:            readUint16(reader),
		CutoffRatio:            readInt32(reader),
		TotalLiveBytes:         readInt32(reader),
		TotalLiveInstances:     readInt32(reader),
		TotalBytesAllocated:    readInt64(reader),
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
			AllocSitesID:               allocSites.ID,
			ArrayIndicator:             readBasicType(reader),
			ClassSerialNumber:          readInt32(reader),
			StackTraceSerialNumber:     readInt32(reader),
			NumberOfLiveBytes:          readInt32(reader),
			NumberOfLiveInstances:      readInt32(reader),
			NumberOfBytesAllocated:     readInt32(reader),
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
		ID:           readID(reader),
		JNIGlobalRef: readID(reader),
	}

	if err := SaveRootJNIGlobal(&rootJNIGlobal); err != nil {
		fmt.Errorf("Error saving RootJNIGlobal to database: %v\n", err)
	}
}

func readRootJNILocal(reader io.Reader) {
	rootJNILocal := RootJNILocal{
		ID:                      readID(reader),
		ThreadSerialNumber:      readInt32(reader),
		FrameNumberInStackTrace: readInt32(reader),
	}

	if err := SaveRootJNILocal(&rootJNILocal); err != nil {
		fmt.Errorf("Error saving RootJNILocal to database: %v\n", err)
	}
}

func readRootJavaFrame(reader io.Reader) {
	rootJavaFrame := RootJavaFrame{
		ObjectID:                readID(reader),
		ThreadSerialNumber:      readInt32(reader),
		FrameNumberInStackTrace: readInt32(reader),
	}

	if err := SaveRootJavaFrame(&rootJavaFrame); err != nil {
		fmt.Errorf("Error saving RootJavaFrame to database: %v\n", err)
	}
}

func readRootNativeStack(reader io.Reader) {
	rootNativeStack := RootNativeStack{
		ID:                 readID(reader),
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
		ID:                 readID(reader),
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
		ID:                     readID(reader),
		ThreadSerialNumber:     readInt32(reader),
		StackTraceSerialNumber: readInt32(reader),
	}

	if err := SaveRootThreadObject(&rootThreadObject); err != nil {
		fmt.Errorf("Error saving RootThreadObject to database: %v\n", err)
	}
}

func readClassDump(reader io.Reader) {
	classDump := ClassDump{
		ID:                       readID(reader),
		StackTraceSerialNumber:   readInt32(reader),
		SuperClassObjectID:       readID(reader),
		ClassLoaderObjectID:      readID(reader),
		SignersObjectID:          readID(reader),
		ProtectionDomainObjectID: readID(reader),
		Reserved1:                readID(reader),
		Reserved2:                readID(reader),
		InstanceSize:             readInt32(reader),
	}

	if err := SaveClassDump(&classDump); err != nil {
		fmt.Errorf("Error saving ClassDump to database: %v\n", err)
		return
	}

	// Read the constant pool
	constantPoolSize := readUint16(reader)
	for i := 0; i < int(constantPoolSize); i++ {
		constantPoolRecord := ConstantPoolRecord{
			ClassDumpID:       classDump.ID,
			ConstantPoolIndex: readUint16(reader),
			Type:              readBasicType(reader),
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
			ClassDumpID:             classDump.ID,
			StaticFieldNameStringID: readID(reader),
			Type:                    readBasicType(reader),
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
			ClassDumpID:       classDump.ID,
			FieldNameStringID: readID(reader),
			Type:              readBasicType(reader),
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
		ID:                     readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		ClassObjectID:          readID(reader),
		NumberOfBytes:          readInt32(reader),
	}

	instanceDump.Data = readArray(reader, instanceDump.NumberOfBytes)

	if err := SaveInstanceDump(&instanceDump); err != nil {
		fmt.Errorf("Error saving InstanceDump to database: %v\n", err)
		return
	}
}

func readObjectArrayDump(reader io.Reader) {
	objectArrayDump := ObjectArrayDump{
		ID:                     readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		NumberOfElements:       readInt32(reader),
		ArrayClassObjectID:     readID(reader),
	}

	// Add validation for reasonable array size
	if objectArrayDump.NumberOfElements < 0 {
		fmt.Printf("Error: negative array elements count %d\n", objectArrayDump.NumberOfElements)
		return
	}

	if objectArrayDump.NumberOfElements > 10000 {
		fmt.Printf("Processing ObjectArrayDump with %d elements\n", objectArrayDump.NumberOfElements)
	}

	// Limit processing of extremely large arrays
	const maxElementsToProcess = 10000000 // 10 million elements max
	if objectArrayDump.NumberOfElements > maxElementsToProcess {
		fmt.Printf("Warning: Object array too large (%d elements), skipping element processing\n", objectArrayDump.NumberOfElements)

		// Save the array metadata
		if err := SaveObjectArrayDump(&objectArrayDump); err != nil {
			fmt.Printf("Error saving ObjectArrayDump to database: %v\n", err)
			return
		}

		// Skip the data without processing individual elements
		for i := int32(0); i < objectArrayDump.NumberOfElements; i++ {
			readID(reader) // Skip each ID
		}
		return
	}

	if err := SaveObjectArrayDump(&objectArrayDump); err != nil {
		fmt.Printf("Error saving ObjectArrayDump to database: %v\n", err)
		return
	}

	// Process elements in batches for better performance
	const batchSize = 10000
	elements := make([]ObjectArrayElement, 0, batchSize)

	for i := int32(0); i < objectArrayDump.NumberOfElements; i++ {
		arrayElement := ObjectArrayElement{
			ObjectArrayDumpID: objectArrayDump.ID,
			Index:             i,
			InstanceDumpID:    readID(reader),
		}

		elements = append(elements, arrayElement)

		// Save in batches
		if len(elements) >= batchSize || i == objectArrayDump.NumberOfElements-1 {
			if err := GetDB().CreateInBatches(elements, batchSize).Error; err != nil {
				fmt.Printf("Error saving ObjectArrayElement batch to database: %v\n", err)
				return
			}

			// Show progress for large arrays
			if objectArrayDump.NumberOfElements > 100000 {
				progress := float64(i+1) / float64(objectArrayDump.NumberOfElements) * 100
				fmt.Printf("ObjectArray Progress: %.1f%% (%d/%d elements)\n", progress, i+1, objectArrayDump.NumberOfElements)
			}

			elements = elements[:0] // Reset slice
		}
	}
}

func readPrimitiveArrayDump(reader io.Reader) {
	primitiveArrayDump := PrimitiveArrayDump{
		ID:                     readID(reader),
		StackTraceSerialNumber: readInt32(reader),
		NumberOfElements:       readInt32(reader),
		Type:                   readBasicType(reader),
	}

	// Add validation for reasonable array size
	if primitiveArrayDump.NumberOfElements < 0 {
		fmt.Printf("Error: negative array elements count %d\n", primitiveArrayDump.NumberOfElements)
		return
	}

	if primitiveArrayDump.NumberOfElements > 10000 {
		fmt.Printf("Processing PrimitiveArrayDump with %d elements (type: %s)\n",
			primitiveArrayDump.NumberOfElements, primitiveArrayDump.Type.GetName())
	}

	// Limit processing of extremely large arrays
	const maxElementsToProcess = 1000000 // 1 million elements max
	if primitiveArrayDump.NumberOfElements > maxElementsToProcess {
		fmt.Printf("Warning: Array too large (%d elements), skipping element processing\n", primitiveArrayDump.NumberOfElements)

		// Save the array metadata
		if err := SavePrimitiveArrayDump(&primitiveArrayDump); err != nil {
			fmt.Printf("Error saving PrimitiveArrayDump to database: %v\n", err)
			return
		}

		// Skip the data without processing individual elements
		totalDataSize := primitiveArrayDump.NumberOfElements * primitiveArrayDump.Type.GetSize()
		readArray(reader, totalDataSize)
		return
	}

	if err := SavePrimitiveArrayDump(&primitiveArrayDump); err != nil {
		fmt.Printf("Error saving PrimitiveArrayDump to database: %v\n", err)
		return
	}

	// Read all array data at once instead of element by element
	elementSize := primitiveArrayDump.Type.GetSize()
	totalDataSize := primitiveArrayDump.NumberOfElements * elementSize
	allData := readArray(reader, totalDataSize)

	if allData == nil {
		fmt.Printf("Error reading array data\n")
		return
	}

	// Process elements in batches for better performance
	const batchSize = 10000
	elements := make([]PrimitiveArrayElement, 0, batchSize)

	for i := int32(0); i < primitiveArrayDump.NumberOfElements; i++ {
		start := i * elementSize
		end := start + elementSize

		if int(end) > len(allData) {
			fmt.Printf("Error: array data truncated at element %d\n", i)
			break
		}

		element := PrimitiveArrayElement{
			PrimitiveArrayDumpID: primitiveArrayDump.ID,
			Index:                i,
			Value:                allData[start:end],
		}

		elements = append(elements, element)

		// Save in batches and show progress
		if len(elements) >= batchSize || i == primitiveArrayDump.NumberOfElements-1 {
			if err := GetDB().CreateInBatches(elements, batchSize).Error; err != nil {
				fmt.Printf("Error saving PrimitiveArrayElement batch to database: %v\n", err)
				return
			}

			// Show progress for large arrays
			if primitiveArrayDump.NumberOfElements > 100000 {
				progress := float64(i+1) / float64(primitiveArrayDump.NumberOfElements) * 100
				fmt.Printf("PrimitiveArray Progress: %.1f%% (%d/%d elements)\n", progress, i+1, primitiveArrayDump.NumberOfElements)
			}

			elements = elements[:0] // Reset slice
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
	t := 0
	i := 0
	fmt.Printf("Reading records...\n")
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
			readStringInUTF8(record.DataReader, record.Length)
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

				i++
				if i%500 == 0 {
					fmt.Printf("\tProcessed %d sub tags\n", i)
				}
			}
		}
		t++

		if t%1000 == 0 {
			fmt.Printf("Processed %d records\n", t)
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

	type ClassSizeInfo struct {
		ClassID       ID     `gorm:"column:class_id"`
		ClassName     string `gorm:"column:class_name"`
		InstanceSize  int64  `gorm:"column:instance_size"`
		InstancesSize int64  `gorm:"column:instances_size"`
		TotalSize     int64  `gorm:"column:total_size"`
	}

	// Один оптимизированный запрос для получения всех данных
	var classSizeInfos []ClassSizeInfo
	query := `
		SELECT 
			cd."ID" as class_id,
			COALESCE(REPLACE(convert_from(s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || cd."ID"::text) as class_name,
			cd."InstanceSize" as instance_size,
			COALESCE(SUM(id."NumberOfBytes"), 0) as instances_size,
			cd."InstanceSize" + COALESCE(SUM(id."NumberOfBytes"), 0) as total_size
		FROM "ClassDump" cd
		LEFT JOIN "InstanceDump" id ON cd."ID" = id."ClassObjectID"
		LEFT JOIN "LoadClass" lc ON cd."ID" = lc."ClassObjectID"
		LEFT JOIN "StringInUTF8" s ON lc."ClassNameStringID" = s."StringID"
		GROUP BY cd."ID", cd."InstanceSize", s."Bytes"
		ORDER BY total_size DESC
		LIMIT ?
	`

	if err := GetDB().Raw(query, max).Scan(&classSizeInfos).Error; err != nil {
		fmt.Printf("Error getting class size information: %v\n", err)
		return result
	}

	// Заполняем результат
	for i, info := range classSizeInfos {
		if i >= max {
			break
		}
		result.Body[i] = fmt.Sprintf("%d. Class ID: %d, Size: %d, Name: %s\n",
			(i + 1), info.ClassID, info.TotalSize, info.ClassName)
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
		count int64
		name  string
	}

	// Получаем количество экземпляров для каждого класса из базы данных
	var classInstanceCounts []struct {
		ClassObjectID ID
		Count         int64
	}

	if err := GetDB().Table("InstanceDump").
		Select("\"ClassObjectID\", COUNT(*) as count").
		Group("\"ClassObjectID\"").
		Scan(&classInstanceCounts).Error; err != nil {
		fmt.Printf("Error getting instance counts: %v\n", err)
		return result
	}

	countPairs := make([]IdCount, 0, len(classInstanceCounts))
	for _, record := range classInstanceCounts {
		className := getClassNameFromDB(record.ClassObjectID)
		countPairs = append(countPairs, IdCount{record.ClassObjectID, record.Count, className})
	}

	sort.Slice(countPairs, func(i, j int) bool {
		return countPairs[i].count > countPairs[j].count
	})

	for i, p := range countPairs {
		if i == max {
			break
		}
		result.Body[i] = fmt.Sprintf("%d. Class ID: %d, Count: %d, Name: %s\n", (i + 1), p.id, p.count, p.name)
	}
	return result
}

func PrintObjectLoadersInfo(max int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: "\n\nObject loaders info\n",
		Body:   make([]string, 0),
	}

	type LoaderInfo struct {
		LoaderID   ID     `gorm:"column:loader_id"`
		LoaderName string `gorm:"column:loader_name"`
		ClassCount int64  `gorm:"column:class_count"`
	}

	var loaderInfos []LoaderInfo
	query := `
		SELECT 
			cd."ClassLoaderObjectID" as loader_id,
			CASE 
				WHEN cd."ClassLoaderObjectID" = 0 THEN 'Bootstrap ClassLoader (System)'
				ELSE COALESCE(REPLACE(convert_from(s."Bytes", 'UTF8'), '/', '.'), 'Unknown loader ' || cd."ClassLoaderObjectID"::text)
			END as loader_name,
			COUNT(*) as class_count
		FROM "ClassDump" cd
		LEFT JOIN "InstanceDump" id ON cd."ClassLoaderObjectID" = id."ID"
		LEFT JOIN "LoadClass" lc ON id."ClassObjectID" = lc."ClassObjectID"
		LEFT JOIN "StringInUTF8" s ON lc."ClassNameStringID" = s."StringID"
		GROUP BY cd."ClassLoaderObjectID", loader_name
		ORDER BY class_count DESC
	`

	if err := GetDB().Raw(query).Scan(&loaderInfos).Error; err != nil {
		fmt.Printf("Error getting loader info: %v\n", err)
		return result
	}

	for _, loaderInfo := range loaderInfos {
		result.Body = append(result.Body, fmt.Sprintf("Loader ID: %d, Name: %s, Number of classes: %d\n",
			loaderInfo.LoaderID, loaderInfo.LoaderName, loaderInfo.ClassCount))

		type ClassInfo struct {
			ClassID   ID     `gorm:"column:class_id"`
			ClassName string `gorm:"column:class_name"`
		}

		var classInfos []ClassInfo
		classQuery := `
			SELECT 
				cd."ID" as class_id,
				COALESCE(REPLACE(convert_from(s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || cd."ID"::text) as class_name
			FROM "ClassDump" cd
			LEFT JOIN "LoadClass" lc ON cd."ID" = lc."ClassObjectID"
			LEFT JOIN "StringInUTF8" s ON lc."ClassNameStringID" = s."StringID"
			WHERE cd."ClassLoaderObjectID" = ?
			LIMIT ?
		`

		if err := GetDB().Raw(classQuery, loaderInfo.LoaderID, max).Scan(&classInfos).Error; err != nil {
			fmt.Printf("Error getting classes for loader %d: %v\n", loaderInfo.LoaderID, err)
			continue
		}

		for i, classInfo := range classInfos {
			if i == max {
				result.Body = append(result.Body, "\t\t...\n")
				break
			}
			result.Body = append(result.Body, fmt.Sprintf("\t\tClass ID: %d, Name: %s\n", classInfo.ClassID, classInfo.ClassName))
		}

		if len(classInfos) == max && loaderInfo.ClassCount > int64(max) {
			result.Body = append(result.Body, "\t\t...\n")
		}
	}
	return result
}

func PrintFullClassSize(max int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: fmt.Sprintf("\n\nTop %d classes by full size (with all depends object)\n", max),
		Body:   make([]string, max),
	}

	classStatsMap := CalculateClassSizesFromDB()

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

	type ArraySizeInfo struct {
		ArrayType string `gorm:"column:array_type"`
		TotalSize int64  `gorm:"column:total_size"`
	}

	var arraySizeInfos []ArraySizeInfo

	objectArrayQuery := `
		SELECT 
			COALESCE(REPLACE(convert_from(s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad."ArrayClassObjectID"::text) || '[]' as array_type,
			SUM(? + oad."NumberOfElements" * 8) as total_size
		FROM "ObjectArrayDump" oad
		LEFT JOIN "LoadClass" lc ON oad."ArrayClassObjectID" = lc."ClassObjectID"
		LEFT JOIN "StringInUTF8" s ON lc."ClassNameStringID" = s."StringID"
		GROUP BY oad."ArrayClassObjectID", s."Bytes"
		ORDER BY total_size DESC
	`

	var objectArrayResults []ArraySizeInfo
	if err := GetDB().Raw(objectArrayQuery, ArrayHeaderSize).Scan(&objectArrayResults).Error; err != nil {
		fmt.Printf("Error getting ObjectArrayDump size info: %v\n", err)
	} else {
		arraySizeInfos = append(arraySizeInfos, objectArrayResults...)
	}

	primitiveArrayQuery := `
		SELECT 
			CASE pad."Type"
				WHEN 2 THEN 'object[]'
				WHEN 4 THEN 'bool[]'
				WHEN 5 THEN 'char[]'
				WHEN 6 THEN 'float[]'
				WHEN 7 THEN 'double[]'
				WHEN 8 THEN 'byte[]'
				WHEN 9 THEN 'short[]'
				WHEN 10 THEN 'int[]'
				WHEN 11 THEN 'long[]'
				ELSE 'unknown[]'
			END as array_type,
			SUM(
				? + pad."NumberOfElements" * 
				CASE pad."Type"
					WHEN 4 THEN 1    -- bool: 1 byte
					WHEN 8 THEN 1    -- byte: 1 byte
					WHEN 5 THEN 2    -- char: 2 bytes
					WHEN 9 THEN 2    -- short: 2 bytes
					WHEN 6 THEN 4    -- float: 4 bytes
					WHEN 10 THEN 4   -- int: 4 bytes
					WHEN 2 THEN 8    -- object: 8 bytes
					WHEN 7 THEN 8    -- double: 8 bytes
					WHEN 11 THEN 8   -- long: 8 bytes
					ELSE 0
				END
			) as total_size
		FROM "PrimitiveArrayDump" pad
		GROUP BY pad."Type"
		ORDER BY total_size DESC
	`

	var primitiveArrayResults []ArraySizeInfo
	if err := GetDB().Raw(primitiveArrayQuery, ArrayHeaderSize).Scan(&primitiveArrayResults).Error; err != nil {
		fmt.Printf("Error getting PrimitiveArrayDump size info: %v\n", err)
	} else {
		arraySizeInfos = append(arraySizeInfos, primitiveArrayResults...)
	}

	// Сортируем все результаты по размеру
	sort.Slice(arraySizeInfos, func(i, j int) bool {
		return arraySizeInfos[i].TotalSize > arraySizeInfos[j].TotalSize
	})

	// Заполняем результат
	for i, info := range arraySizeInfos {
		if i >= max {
			break
		}
		result.Body[i] = fmt.Sprintf("%d. Array: %s, Size: %d\n", (i + 1), info.ArrayType, info.TotalSize)
	}

	return result
}

type ClassStats struct {
	ClassName string
	TotalSize int32
}

func CalculateClassSizesFromDB() map[ID]ClassStats {
	result := make(map[ID]ClassStats)

	// Получаем все классы из базы данных
	var classes []ClassDump
	if err := GetDB().Find(&classes).Error; err != nil {
		fmt.Printf("Error getting classes from database: %v\n", err)
		return result
	}

	fmt.Printf("Processing %d classes for full size calculation...\n", len(classes))

	for i, classDump := range classes {
		if i%100 == 0 {
			fmt.Printf("Processing class %d/%d\n", i+1, len(classes))
		}

		visited := make(map[ID]bool)
		var totalSize int64

		// Получаем имя класса
		className := getClassNameFromDB(classDump.ID)

		// 1. Добавляем размер самого класса (статические поля)
		classSize := calculateClassSizeFromDB(classDump.ID)
		totalSize += classSize

		// 2. Получаем все экземпляры данного класса
		instanceIds := getInstanceIdsForClassFromDB(classDump.ID)

		// 3. Для каждого экземпляра проходим граф ссылок в ширину
		queue := make([]ID, 0)

		for _, instanceId := range instanceIds {
			if !visited[instanceId] {
				visited[instanceId] = true
				queue = append(queue, instanceId)
				size := getObjectSizeFromDB(instanceId)
				totalSize += size
			}
		}

		staticRefs := getStaticFieldReferencesFromDB(classDump.ID)
		for _, refId := range staticRefs {
			if refId != 0 && !visited[refId] {
				visited[refId] = true
				queue = append(queue, refId)
				size := getObjectSizeFromDB(refId)
				totalSize += size
			}
		}

		for len(queue) > 0 {
			currentId := queue[0]
			queue = queue[1:]

			refs := getObjectReferencesFromDB(currentId)
			for _, refId := range refs {
				if refId != 0 && !visited[refId] {
					visited[refId] = true
					queue = append(queue, refId)
					size := getObjectSizeFromDB(refId)
					totalSize += size
				}
			}
		}

		result[classDump.ID] = ClassStats{
			ClassName: className,
			TotalSize: int32(totalSize),
		}
	}

	return result
}

func getClassNameFromDB(classID ID) string {
	// Получаем LoadClass запись для данного класса
	var loadClass LoadClass
	if err := GetDB().Where("\"ClassObjectID\" = ?", classID).First(&loadClass).Error; err != nil {
		fmt.Printf("Error getting LoadClass for class %d: %v\n", classID, err)
		return fmt.Sprintf("Unknown class %d", classID)
	}

	// Получаем строку с именем класса
	var stringData StringInUTF8
	if err := GetDB().Where("\"StringID\" = ?", loadClass.ClassNameStringID).First(&stringData).Error; err != nil {
		fmt.Printf("Error getting class name string for class %d: %v\n", classID, err)
		return fmt.Sprintf("Unknown class %d", classID)
	}

	className := string(stringData.Bytes)
	className = strings.ReplaceAll(className, "/", ".")

	return className
}

func calculateClassSizeFromDB(classID ID) int64 {
	var totalSize int64

	var staticFields []StaticFieldRecord
	if err := GetDB().Where("\"ClassDumpID\" = ?", classID).Find(&staticFields).Error; err != nil {
		fmt.Printf("Error getting static fields for class %d: %v\n", classID, err)
		return 0
	}

	for _, sf := range staticFields {
		totalSize += int64(sf.Type.GetSize())
	}

	return totalSize
}

func getInstanceIdsForClassFromDB(classID ID) []ID {
	var instanceIds []ID
	if err := GetDB().Table("InstanceDump").
		Where("\"ClassObjectID\" = ?", classID).
		Pluck("\"ID\"", &instanceIds).Error; err != nil {
		fmt.Printf("Error getting instance IDs for class %d: %v\n", classID, err)
		return nil
	}
	return instanceIds
}

func getObjectSizeFromDB(objectID ID) int64 {
	var instance InstanceDump
	if err := GetDB().Where("\"ID\" = ?", objectID).First(&instance).Error; err == nil {
		return int64(instance.NumberOfBytes)
	}

	var objectArray ObjectArrayDump
	if err := GetDB().Where("\"ID\" = ?", objectID).First(&objectArray).Error; err == nil {
		return int64(ArrayHeaderSize + objectArray.NumberOfElements*8)
	}

	var primitiveArray PrimitiveArrayDump
	if err := GetDB().Where("\"ID\" = ?", objectID).First(&primitiveArray).Error; err == nil {
		return int64(ArrayHeaderSize + primitiveArray.NumberOfElements*primitiveArray.Type.GetSize())
	}

	return 0
}

func getStaticFieldReferencesFromDB(classID ID) []ID {
	var refs []ID

	var staticFields []StaticFieldRecord
	if err := GetDB().Where("\"ClassDumpID\" = ? AND \"Type\" = ?", classID, Object).Find(&staticFields).Error; err != nil {
		fmt.Printf("Error getting static field references for class %d: %v\n", classID, err)
		return nil
	}

	for _, sf := range staticFields {
		if len(sf.Value) >= 8 {
			refId := ID(binary.BigEndian.Uint64(sf.Value))
			if refId != 0 {
				refs = append(refs, refId)
			}
		}
	}

	return refs
}

func getObjectReferencesFromDB(objectID ID) []ID {
	var refs []ID

	// Проверяем InstanceDump
	var instance InstanceDump
	if err := GetDB().Where("\"ID\" = ?", objectID).First(&instance).Error; err == nil {
		return parseInstanceReferencesFromDB(instance)
	}

	// Проверяем ObjectArrayDump - получаем элементы массива
	var elements []ObjectArrayElement
	if err := GetDB().Where("\"ObjectArrayDumpID\" = ?", objectID).Find(&elements).Error; err == nil {
		for _, element := range elements {
			if element.InstanceDumpID != 0 {
				refs = append(refs, element.InstanceDumpID)
			}
		}
		return refs
	}

	// PrimitiveArrayDump не содержит ссылок
	return nil
}

func parseInstanceReferencesFromDB(instance InstanceDump) []ID {
	var refs []ID

	// Получаем все поля экземпляра для данного класса и его суперклассов
	allFields := getAllInstanceFieldsFromDB(instance.ClassObjectID)

	// Парсим данные экземпляра
	offset := 0
	for _, field := range allFields {
		if field.Type == Object {
			start := offset
			end := offset + 8
			if end <= len(instance.Data) {
				refId := ID(binary.BigEndian.Uint64(instance.Data[start:end]))
				if refId != 0 {
					refs = append(refs, refId)
				}
			}
		}
		offset += int(field.Type.GetSize())
	}

	return refs
}

func getAllInstanceFieldsFromDB(classID ID) []InstanceFieldRecord {
	var allFields []InstanceFieldRecord
	currentClassID := classID

	for currentClassID != 0 {
		var fields []InstanceFieldRecord
		if err := GetDB().Where("\"ClassDumpID\" = ?", currentClassID).Find(&fields).Error; err != nil {
			fmt.Printf("Error getting instance fields for class %d: %v\n", currentClassID, err)
			break
		}

		// Добавляем поля в начало, так как нужно сохранить порядок наследования
		allFields = append(fields, allFields...)

		// Получаем суперкласс
		var classDump ClassDump
		if err := GetDB().Where("\"ID\" = ?", currentClassID).First(&classDump).Error; err != nil {
			break
		}
		currentClassID = classDump.SuperClassObjectID
	}

	return allFields
}
