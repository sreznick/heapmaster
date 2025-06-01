package hprof

import (
	"encoding/binary"
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"
	"unsafe"
)

// AnalyzeLongArrays
// выводит информацию о массивов (объектных и примитивных), длина которых >= minElements.
func AnalyzeLongArrays(minElements int) (result AnalyzeResult){
	result = AnalyzeResult{
		Header: fmt.Sprintf("Анализ длинных массивов (minElements = %d)", minElements),
		Body:  make([]string, 0),
	}

	type ArrayInfo struct {
		Kind        string
		ObjectID    ID
		NumElements int32
		TotalSize   int32
	}
	var arrays []ArrayInfo

	// Анализ объектных массивов
	for _, arr := range ObjectIdToObjectArrayDumpMap {
		size := ArrayHeaderSize + arr.NumberOfElements*8
		if arr.NumberOfElements >= int32(minElements) {
			arrays = append(arrays, ArrayInfo{
				Kind:        "ObjectArray: " + getNameByClassObjectId(arr.ArrayClassObjectId),
				ObjectID:    arr.ArrayObjectId,
				NumElements: arr.NumberOfElements,
				TotalSize:   size,
			})
		}
	}

	// Анализ примитивных массивов
	for _, arr := range ObjectIdToPrimitiveArrayDumpMap {
		size := ArrayHeaderSize + arr.NumberOfElements*arr.ElementType.GetSize()
		if arr.NumberOfElements >= int32(minElements) {
			arrays = append(arrays, ArrayInfo{
				Kind:        "PrimitiveArray: " + arr.ElementType.GetName(),
				ObjectID:    arr.ArrayObjectId,
				NumElements: arr.NumberOfElements,
				TotalSize:   size,
			})
		}
	}

	sort.Slice(arrays, func(i, j int) bool { return arrays[i].TotalSize > arrays[j].TotalSize })
	for i, info := range arrays {
		result.Body = append(result.Body, fmt.Sprintf("%d. ID: %d, Вид: %s, Элементов: %d, Размер: %d байт\n",
			i+1, info.ObjectID, info.Kind, info.NumElements, info.TotalSize))
	}
	
	return result
}

// AnalyzeHashMapOverheads:
// ищет экземпляры, у которых имя класса содержит "HashMap"
// и выводит их размер, что может служить индикатором высокого оверхеда.
func AnalyzeHashMapOverheads(maxSize int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: fmt.Sprintf("Анализ оверхеда HashMap (maxSize = %d)", maxSize),
		Body:  make([]string, 0),
	}
	type HashMapInfo struct {
		ObjectID  ID
		ClassName string
		Size      int32
	}
	var hashMaps []HashMapInfo
	for _, instance := range ObjectIdToInstanceDumpMap {
		className := getNameByClassObjectId(instance.ClassObjectId)
		if strings.Contains(className, "HashMap") {
			hashMaps = append(hashMaps, HashMapInfo{
				ObjectID:  instance.ObjectId,
				ClassName: className,
				Size:      instance.NumberOfBytes,
			})
		}
	}

	sort.Slice(hashMaps, func(i, j int) bool {
		return hashMaps[i].Size > hashMaps[j].Size
	})
	for i, info := range hashMaps {
		if i >= maxSize {
			break
		}
		result.Body = append(result.Body, fmt.Sprintf("%d. ID: %d, Класс: %s, Размер экземпляра: %d байт\n",
			i+1, info.ObjectID, info.ClassName, info.Size))
	}
	return result
}

// AnalyzeDuplicateStrings выводит информацию о повторяющихся строках.
// Группировка производится по содержимому и по одинаковому указателю на данные.
func AnalyzeDuplicateStrings() (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: "Анализ повторяющихся строк",
		Body:   make([]string, 0),
	}

	type DupInfo struct {
		Content string
		Count   int
		Ptr     uintptr
	}
	duplicates := make([]DupInfo, 0)
	// Changed: use string (content) as key instead of uintptr
	freq := make(map[string]DupInfo)

	// Helper: декодирование char-массива (big-endian)
	decodeCharArray := func(data []byte) string {
		if len(data)%2 != 0 {
			data = data[:len(data)-1]
		}
		u16s := make([]uint16, 0, len(data)/2)
		for i := 0; i < len(data); i += 2 {
			u16 := binary.BigEndian.Uint16(data[i : i+2])
			u16s = append(u16s, u16)
		}
		return string(utf16.Decode(u16s))
	}

	for _, inst := range ObjectIdToInstanceDumpMap {
		className := getNameByClassObjectId(inst.ClassObjectId)
		if className != "java.lang.String" {
			continue
		}
		_, ok := ClassObjectIdToClassDumpMap[inst.ClassObjectId]
		if !ok {
			continue
		}
		var allFields []InstanceFieldRecord
		currentClassId := inst.ClassObjectId
		for {
			cd, exists := ClassObjectIdToClassDumpMap[currentClassId]
			if !exists {
				break
			}
			allFields = append(cd.InstanceFields, allFields...)
			if cd.SuperClassObjectId == 0 {
				break
			}
			currentClassId = cd.SuperClassObjectId
		}
		var offset int
		found := false
		for _, field := range allFields {
			fieldName := IDtoStringInUTF8[field.FieldNameStringId]
			fieldSize := int(field.Type.GetSize())
			if field.Type == Object && fieldName == "value" {
				found = true
				break
			}
			offset += fieldSize
		}
		if !found || offset+8 > len(inst.InstanceFieldValues) {
			continue
		}
		refId := ID(binary.BigEndian.Uint64(inst.InstanceFieldValues[offset : offset+8]))
		arr, exists := ObjectIdToPrimitiveArrayDumpMap[refId]
		if !exists || len(arr.Elements) == 0 {
			continue
		}
		ptr := uintptr(unsafe.Pointer(&arr.Elements[0]))
		var content string
		if arr.ElementType.GetName() == "char" {
			content = decodeCharArray(arr.Elements)
		} else {
			content = string(arr.Elements)
		}
		// Group by value (content) instead of pointer:
		if di, exists := freq[content]; exists {
			di.Count++
			freq[content] = di
		} else {
			freq[content] = DupInfo{
				Content: content,
				Count:   1,
				Ptr:     ptr, // сохраняем первый указатель
			}
		}
	}

	for _, di := range freq {
		if di.Count > 1 {
			duplicates = append(duplicates, di)
		}
	}
	sort.Slice(duplicates, func(i, j int) bool { return duplicates[i].Count > duplicates[j].Count })

	for i, dup := range duplicates {
		result.Body = append(result.Body, fmt.Sprintf("%d. Количество: %d, Адрес: 0x%x, Строка: %s\n",
			i+1, dup.Count, dup.Ptr, dup.Content))
	}
	if len(duplicates) == 0 {
		result.Body = append(result.Body, "Нет повторяющихся строк.\n")
	}
	return result
}
