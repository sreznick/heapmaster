package hprof

import (
	"fmt"
	"sort"
	"strings"
)

// ArrayInfo представляет информацию о массиве для анализа
type ArrayInfo struct {
	Kind        string
	ObjectID    ID
	NumElements int32
	TotalSize   int32
}

// AnalyzeLongArrays
// выводит информацию о массивов (объектных и примитивных), длина которых >= minElements.
func AnalyzeLongArrays(minElements int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: fmt.Sprintf("Анализ длинных массивов (minElements = %d)\n", minElements),
		Body:   make([]string, 0),
	}

	if !IsDBInitialized() {
		result.Body = append(result.Body, "Error: Database is not initialized\n")
		return result
	}

	var arrays []ArrayInfo

	// Анализ объектных массивов
	var objectArrays []ObjectArrayDump
	if err := GetDB().Where("\"NumberOfElements\" >= ?", minElements).Find(&objectArrays).Error; err != nil {
		result.Body = append(result.Body, fmt.Sprintf("Error retrieving object arrays: %v\n", err))
		return result
	}

	for _, arr := range objectArrays {
		size := ArrayHeaderSize + arr.NumberOfElements*8
		className := getClassNameFromDB(arr.ArrayClassObjectID)
		arrays = append(arrays, ArrayInfo{
			Kind:        "ObjectArray: " + className,
			ObjectID:    arr.ID,
			NumElements: arr.NumberOfElements,
			TotalSize:   size,
		})
	}

	// Анализ примитивных массивов
	var primitiveArrays []PrimitiveArrayDump
	if err := GetDB().Where("\"NumberOfElements\" >= ?", minElements).Find(&primitiveArrays).Error; err != nil {
		result.Body = append(result.Body, fmt.Sprintf("Error retrieving primitive arrays: %v\n", err))
		return result
	}

	for _, arr := range primitiveArrays {
		size := ArrayHeaderSize + arr.NumberOfElements*arr.Type.GetSize()
		arrays = append(arrays, ArrayInfo{
			Kind:        "PrimitiveArray: " + arr.Type.GetName(),
			ObjectID:    arr.ID,
			NumElements: arr.NumberOfElements,
			TotalSize:   size,
		})
	}

	// Сортируем массивы по размеру (убывание)
	sort.Slice(arrays, func(i, j int) bool {
		return arrays[i].TotalSize > arrays[j].TotalSize
	})

	// Формируем результат
	if len(arrays) == 0 {
		result.Body = append(result.Body, fmt.Sprintf("Массивы с количеством элементов >= %d не найдены\n", minElements))
	} else {
		for i, info := range arrays {
			result.Body = append(result.Body, fmt.Sprintf("%d. ID: %d, Вид: %s, Элементов: %d, Размер: %d байт\n",
				i+1, info.ObjectID, info.Kind, info.NumElements, info.TotalSize))
		}
	}

	return result
}

// HashMapInfo представляет информацию о HashMap для анализа оверхеда
type HashMapInfo struct {
	ObjectID  ID
	ClassName string
	Size      int32
}

// AnalyzeHashMapOverheads:
// ищет экземпляры, у которых имя класса содержит "HashMap"
// и выводит их размер, что может служить индикатором высокого оверхеда.
func AnalyzeHashMapOverheads(maxSize int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: fmt.Sprintf("Анализ оверхеда HashMap (maxSize = %d)", maxSize),
		Body:   make([]string, 0),
	}

	if !IsDBInitialized() {
		result.Body = append(result.Body, "Error: Database is not initialized\n")
		return result
	}

	var hashMaps []HashMapInfo

	// Получаем все экземпляры из базы данных
	var instances []InstanceDump
	if err := GetDB().Find(&instances).Error; err != nil {
		result.Body = append(result.Body, fmt.Sprintf("Error retrieving instances: %v\n", err))
		return result
	}

	// Фильтруем экземпляры HashMap
	for _, instance := range instances {
		className := getClassNameFromDB(instance.ClassObjectID)
		if strings.Contains(className, "HashMap") {
			hashMaps = append(hashMaps, HashMapInfo{
				ObjectID:  instance.ID,
				ClassName: className,
				Size:      instance.NumberOfBytes,
			})
		}
	}

	// Сортируем по размеру (убывание)
	sort.Slice(hashMaps, func(i, j int) bool {
		return hashMaps[i].Size > hashMaps[j].Size
	})

	// Формируем результат
	if len(hashMaps) == 0 {
		result.Body = append(result.Body, "HashMap экземпляры не найдены\n")
	} else {
		for i, info := range hashMaps {
			if i >= maxSize {
				break
			}
			result.Body = append(result.Body, fmt.Sprintf("%d. ID: %d, Класс: %s, Размер экземпляра: %d байт\n",
				i+1, info.ObjectID, info.ClassName, info.Size))
		}
	}

	return result
}

// ArrayOwnerInfo представляет информацию о владельце массива
type ArrayOwnerInfo struct {
	ArrayID       ID
	ArrayType     string
	ArrayElements int32
	OwnerType     string // "InstanceField", "StaticField", "ArrayElement", "RootReference"
	OwnerID       ID
	OwnerClass    string
	FieldName     string
}

// AnalyzeArrayOwners
// выводит информацию о владельцах массивов, которые имеют более maxElements элементов.
func AnalyzeArrayOwners(maxElements int) (result AnalyzeResult) {
	result = AnalyzeResult{
		Header: fmt.Sprintf("Анализ владельцев больших массивов (maxElements = %d)\n", maxElements),
		Body:   make([]string, 0),
	}

	if !IsDBInitialized() {
		result.Body = append(result.Body, "Ошибка: База данных не инициализирована\n")
		return result
	}

	var owners []ArrayOwnerInfo

	// 1. Поиск объектных массивов как полей экземпляров
	objectArrayFieldQuery := `
		SELECT DISTINCT
			oad."ID" as array_id,
			COALESCE(REPLACE(convert_from(s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad."ArrayClassObjectID"::text) || '[]' as array_type,
			oad."NumberOfElements" as array_elements,
			'InstanceField' as owner_type,
			ifv."InstanceDumpID" as owner_id,
			COALESCE(REPLACE(convert_from(owner_s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || id."ClassObjectID"::text) as owner_class,
			COALESCE(convert_from(field_s."Bytes", 'UTF8'), 'Unknown field') as field_name
		FROM "ObjectArrayDump" oad
		JOIN "InstanceFieldValues" ifv ON decode(lpad(to_hex(oad."ID"), 16, '0'), 'hex') = ifv."Value" AND ifv."Type" = 2
		JOIN "InstanceDump" id ON ifv."InstanceDumpID" = id."ID"
		JOIN "InstanceFieldRecord" ifr ON ifr."ClassDumpID" = id."ClassObjectID" AND ifr."ID" = ifv."Index" + 1
		LEFT JOIN "LoadClass" lc ON oad."ArrayClassObjectID" = lc."ClassObjectID"
		LEFT JOIN "StringInUTF8" s ON lc."ClassNameStringID" = s."StringID"
		LEFT JOIN "LoadClass" owner_lc ON id."ClassObjectID" = owner_lc."ClassObjectID"
		LEFT JOIN "StringInUTF8" owner_s ON owner_lc."ClassNameStringID" = owner_s."StringID"
		LEFT JOIN "StringInUTF8" field_s ON ifr."FieldNameStringID" = field_s."StringID"
		WHERE oad."NumberOfElements" >= ?
	`

	var objectArrayFieldResults []ArrayOwnerInfo
	if err := GetDB().Raw(objectArrayFieldQuery, maxElements).Scan(&objectArrayFieldResults).Error; err != nil {
		result.Body = append(result.Body, fmt.Sprintf("Ошибка при поиске объектных массивов в полях экземпляров: %v\n", err))
	} else {
		owners = append(owners, objectArrayFieldResults...)
	}

	// 2. Поиск примитивных массивов как полей экземпляров
	primitiveArrayFieldQuery := `
		SELECT DISTINCT
			pad."ID" as array_id,
			CASE pad."Type"
				WHEN 4 THEN 'boolean[]'
				WHEN 5 THEN 'char[]'
				WHEN 6 THEN 'float[]'
				WHEN 7 THEN 'double[]'
				WHEN 8 THEN 'byte[]'
				WHEN 9 THEN 'short[]'
				WHEN 10 THEN 'int[]'
				WHEN 11 THEN 'long[]'
				ELSE 'unknown[]'
			END as array_type,
			pad."NumberOfElements" as array_elements,
			'InstanceField' as owner_type,
			ifv."InstanceDumpID" as owner_id,
			COALESCE(REPLACE(convert_from(owner_s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || id."ClassObjectID"::text) as owner_class,
			COALESCE(convert_from(field_s."Bytes", 'UTF8'), 'Unknown field') as field_name
		FROM "PrimitiveArrayDump" pad
		JOIN "InstanceFieldValues" ifv ON decode(lpad(to_hex(pad."ID"), 16, '0'), 'hex') = ifv."Value" AND ifv."Type" = 2
		JOIN "InstanceDump" id ON ifv."InstanceDumpID" = id."ID"
		JOIN "InstanceFieldRecord" ifr ON ifr."ClassDumpID" = id."ClassObjectID" AND ifr."ID" = ifv."Index" + 1
		LEFT JOIN "LoadClass" owner_lc ON id."ClassObjectID" = owner_lc."ClassObjectID"
		LEFT JOIN "StringInUTF8" owner_s ON owner_lc."ClassNameStringID" = owner_s."StringID"
		LEFT JOIN "StringInUTF8" field_s ON ifr."FieldNameStringID" = field_s."StringID"
		WHERE pad."NumberOfElements" >= ?
	`

	var primitiveArrayFieldResults []ArrayOwnerInfo
	if err := GetDB().Raw(primitiveArrayFieldQuery, maxElements).Scan(&primitiveArrayFieldResults).Error; err != nil {
		result.Body = append(result.Body, fmt.Sprintf("Ошибка при поиске примитивных массивов в полях экземпляров: %v\n", err))
	} else {
		owners = append(owners, primitiveArrayFieldResults...)
	}

	// 3. Поиск объектных массивов как статических полей
	objectArrayStaticQuery := `
		SELECT DISTINCT
			oad."ID" as array_id,
			COALESCE(REPLACE(convert_from(s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad."ArrayClassObjectID"::text) || '[]' as array_type,
			oad."NumberOfElements" as array_elements,
			'StaticField' as owner_type,
			sfr."ClassDumpID" as owner_id,
			COALESCE(REPLACE(convert_from(owner_s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || sfr."ClassDumpID"::text) as owner_class,
			COALESCE(convert_from(field_s."Bytes", 'UTF8'), 'Unknown static field') as field_name
		FROM "ObjectArrayDump" oad
		JOIN "StaticFieldRecord" sfr ON decode(lpad(to_hex(oad."ID"), 16, '0'), 'hex') = sfr."Value" AND sfr."Type" = 2
		LEFT JOIN "LoadClass" lc ON oad."ArrayClassObjectID" = lc."ClassObjectID"
		LEFT JOIN "StringInUTF8" s ON lc."ClassNameStringID" = s."StringID"
		LEFT JOIN "LoadClass" owner_lc ON sfr."ClassDumpID" = owner_lc."ClassObjectID"
		LEFT JOIN "StringInUTF8" owner_s ON owner_lc."ClassNameStringID" = owner_s."StringID"
		LEFT JOIN "StringInUTF8" field_s ON sfr."StaticFieldNameStringID" = field_s."StringID"
		WHERE oad."NumberOfElements" >= ?
	`

	var objectArrayStaticResults []ArrayOwnerInfo
	if err := GetDB().Raw(objectArrayStaticQuery, maxElements).Scan(&objectArrayStaticResults).Error; err != nil {
		result.Body = append(result.Body, fmt.Sprintf("Ошибка при поиске объектных массивов в статических полях: %v\n", err))
	} else {
		owners = append(owners, objectArrayStaticResults...)
	}

	// 4. Поиск примитивных массивов как статических полей
	primitiveArrayStaticQuery := `
		SELECT DISTINCT
			pad."ID" as array_id,
			CASE pad."Type"
				WHEN 4 THEN 'boolean[]'
				WHEN 5 THEN 'char[]'
				WHEN 6 THEN 'float[]'
				WHEN 7 THEN 'double[]'
				WHEN 8 THEN 'byte[]'
				WHEN 9 THEN 'short[]'
				WHEN 10 THEN 'int[]'
				WHEN 11 THEN 'long[]'
				ELSE 'unknown[]'
			END as array_type,
			pad."NumberOfElements" as array_elements,
			'StaticField' as owner_type,
			sfr."ClassDumpID" as owner_id,
			COALESCE(REPLACE(convert_from(owner_s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || sfr."ClassDumpID"::text) as owner_class,
			COALESCE(convert_from(field_s."Bytes", 'UTF8'), 'Unknown static field') as field_name
		FROM "PrimitiveArrayDump" pad
		JOIN "StaticFieldRecord" sfr ON decode(lpad(to_hex(pad."ID"), 16, '0'), 'hex') = sfr."Value" AND sfr."Type" = 2
		LEFT JOIN "LoadClass" owner_lc ON sfr."ClassDumpID" = owner_lc."ClassObjectID"
		LEFT JOIN "StringInUTF8" owner_s ON owner_lc."ClassNameStringID" = owner_s."StringID"
		LEFT JOIN "StringInUTF8" field_s ON sfr."StaticFieldNameStringID" = field_s."StringID"
		WHERE pad."NumberOfElements" >= ?
	`

	var primitiveArrayStaticResults []ArrayOwnerInfo
	if err := GetDB().Raw(primitiveArrayStaticQuery, maxElements).Scan(&primitiveArrayStaticResults).Error; err != nil {
		result.Body = append(result.Body, fmt.Sprintf("Ошибка при поиске примитивных массивов в статических полях: %v\n", err))
	} else {
		owners = append(owners, primitiveArrayStaticResults...)
	}

	// 5. Поиск массивов как элементов других объектных массивов
	arrayInArrayQuery := `
		SELECT DISTINCT
			CASE 
				WHEN oad_inner."ID" IS NOT NULL THEN oad_inner."ID"
				WHEN pad_inner."ID" IS NOT NULL THEN pad_inner."ID"
			END as array_id,
			CASE 
				WHEN oad_inner."ID" IS NOT NULL THEN 
					COALESCE(REPLACE(convert_from(s_inner."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad_inner."ArrayClassObjectID"::text) || '[]'
				WHEN pad_inner."ID" IS NOT NULL THEN 
					CASE pad_inner."Type"
						WHEN 4 THEN 'boolean[]'
						WHEN 5 THEN 'char[]'
						WHEN 6 THEN 'float[]'
						WHEN 7 THEN 'double[]'
						WHEN 8 THEN 'byte[]'
						WHEN 9 THEN 'short[]'
						WHEN 10 THEN 'int[]'
						WHEN 11 THEN 'long[]'
						ELSE 'unknown[]'
					END
			END as array_type,
			CASE 
				WHEN oad_inner."ID" IS NOT NULL THEN oad_inner."NumberOfElements"
				WHEN pad_inner."ID" IS NOT NULL THEN pad_inner."NumberOfElements"
			END as array_elements,
			'ArrayElement' as owner_type,
			oad_outer."ID" as owner_id,
			COALESCE(REPLACE(convert_from(s_outer."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad_outer."ArrayClassObjectID"::text) || '[]' as owner_class,
			'[' || oae."Index"::text || ']' as field_name
		FROM "ObjectArrayElement" oae
		JOIN "ObjectArrayDump" oad_outer ON oae."ObjectArrayDumpID" = oad_outer."ID"
		LEFT JOIN "ObjectArrayDump" oad_inner ON oae."InstanceDumpID" = oad_inner."ID"
		LEFT JOIN "PrimitiveArrayDump" pad_inner ON oae."InstanceDumpID" = pad_inner."ID"
		LEFT JOIN "LoadClass" lc_inner ON oad_inner."ArrayClassObjectID" = lc_inner."ClassObjectID"
		LEFT JOIN "StringInUTF8" s_inner ON lc_inner."ClassNameStringID" = s_inner."StringID"
		LEFT JOIN "LoadClass" lc_outer ON oad_outer."ArrayClassObjectID" = lc_outer."ClassObjectID"
		LEFT JOIN "StringInUTF8" s_outer ON lc_outer."ClassNameStringID" = s_outer."StringID"
		WHERE (oad_inner."NumberOfElements" >= ? OR pad_inner."NumberOfElements" >= ?)
	`

	var arrayInArrayResults []ArrayOwnerInfo
	if err := GetDB().Raw(arrayInArrayQuery, maxElements, maxElements).Scan(&arrayInArrayResults).Error; err != nil {
		result.Body = append(result.Body, fmt.Sprintf("Ошибка при поиске массивов в других массивах: %v\n", err))
	} else {
		owners = append(owners, arrayInArrayResults...)
	}

	sort.Slice(owners, func(i, j int) bool {
		return owners[i].ArrayElements > owners[j].ArrayElements
	})


	if len(owners) == 0 {
		result.Body = append(result.Body, fmt.Sprintf("Массивы с количеством элементов >= %d и их владельцы не найдены\n", maxElements))
	} else {
		result.Body = append(result.Body, fmt.Sprintf("Найдено %d массивов с владельцами:\n\n", len(owners)))

		for i, owner := range owners {
			ownerDescription := ""
			switch owner.OwnerType {
			case "InstanceField":
				ownerDescription = fmt.Sprintf("поле '%s' экземпляра класса '%s' (ID: %d)",
					owner.FieldName, owner.OwnerClass, owner.OwnerID)
			case "StaticField":
				ownerDescription = fmt.Sprintf("статическое поле '%s' класса '%s' (ID: %d)",
					owner.FieldName, owner.OwnerClass, owner.OwnerID)
			case "ArrayElement":
				ownerDescription = fmt.Sprintf("элемент %s массива '%s' (ID: %d)",
					owner.FieldName, owner.OwnerClass, owner.OwnerID)
			default:
				ownerDescription = fmt.Sprintf("неизвестный тип владельца: %s (ID: %d)",
					owner.OwnerType, owner.OwnerID)
			}

			result.Body = append(result.Body, fmt.Sprintf("%d. Массив ID: %d, Тип: %s, Элементов: %d\n   Владелец: %s\n\n",
				i+1, owner.ArrayID, owner.ArrayType, owner.ArrayElements, ownerDescription))
		}
	}

	return result
}

type OwnerArraysInfo struct {
	OwnerType     string // "InstanceField", "StaticField", "ArrayElement"
	OwnerID       ID
	OwnerClass    string
	OwnerField    string // Для случая, когда владелец - поле
	Arrays        []ArrayDetail
	TotalArrays   int
	TotalElements int64
	TotalSize     int64
}

type ArrayDetail struct {
	ArrayID   ID
	ArrayType string
	Elements  int32
	Size      int64
}


// AnalyzeTopArrayOwners
// выводит информацию о владельцах с самыми большими массивами (по суммарному размеру).
// Для каждого владельца показывает все его ограниченное количество (maxArraysPerOwner).
func AnalyzeTopArrayOwners(maxOwners int) (result AnalyzeResult) {
    maxArraysPerOwner := 10
    result = AnalyzeResult{
        Header: fmt.Sprintf("Топ %d владельцев больших массивов (до %d массивов на владельца)\n", maxOwners, maxArraysPerOwner),
        Body:   make([]string, 0),
    }

    if !IsDBInitialized() {
        result.Body = append(result.Body, "Ошибка: База данных не инициализирована\n")
        return result
    }

    type OwnerArrayResult struct {
        OwnerType     string `gorm:"column:owner_type"`
        OwnerID       ID     `gorm:"column:owner_id"`
        OwnerClass    string `gorm:"column:owner_class"`
        OwnerField    string `gorm:"column:owner_field"`
        ArrayID       ID     `gorm:"column:array_id"`
        ArrayType     string `gorm:"column:array_type"`
        ArrayElements int32  `gorm:"column:array_elements"`
        ArraySize     int64  `gorm:"column:array_size"`
    }

    var allResults []OwnerArrayResult

    // 1. Объектные массивы как поля экземпляров
    objectArrayFieldQuery := `
        SELECT DISTINCT
            'InstanceField' as owner_type,
            ifv."InstanceDumpID" as owner_id,
            COALESCE(REPLACE(convert_from(owner_s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || id."ClassObjectID"::text) as owner_class,
            COALESCE(convert_from(field_s."Bytes", 'UTF8'), 'Unknown field') as owner_field,
            oad."ID" as array_id,
            COALESCE(REPLACE(convert_from(s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad."ArrayClassObjectID"::text) || '[]' as array_type,
            oad."NumberOfElements" as array_elements,
            (? + oad."NumberOfElements" * 8) as array_size
        FROM "ObjectArrayDump" oad
        JOIN "InstanceFieldValues" ifv ON decode(lpad(to_hex(oad."ID"), 16, '0'), 'hex') = ifv."Value" AND ifv."Type" = 2
        JOIN "InstanceDump" id ON ifv."InstanceDumpID" = id."ID"
        JOIN "InstanceFieldRecord" ifr ON ifr."ClassDumpID" = id."ClassObjectID" AND ifr."ID" = ifv."Index" + 1
        LEFT JOIN "LoadClass" lc ON oad."ArrayClassObjectID" = lc."ClassObjectID"
        LEFT JOIN "StringInUTF8" s ON lc."ClassNameStringID" = s."StringID"
        LEFT JOIN "LoadClass" owner_lc ON id."ClassObjectID" = owner_lc."ClassObjectID"
        LEFT JOIN "StringInUTF8" owner_s ON owner_lc."ClassNameStringID" = owner_s."StringID"
        LEFT JOIN "StringInUTF8" field_s ON ifr."FieldNameStringID" = field_s."StringID"
        ORDER BY array_size DESC
    `

    var objectArrayFieldResults []OwnerArrayResult
    if err := GetDB().Raw(objectArrayFieldQuery, ArrayHeaderSize).Scan(&objectArrayFieldResults).Error; err != nil {
        result.Body = append(result.Body, fmt.Sprintf("Ошибка при получении объектных массивов в полях экземпляров: %v\n", err))
    } else {
        allResults = append(allResults, objectArrayFieldResults...)
    }

    // 2. Примитивные массивы как поля экземпляров
    primitiveArrayFieldQuery := `
        SELECT DISTINCT
            'InstanceField' as owner_type,
            ifv."InstanceDumpID" as owner_id,
            COALESCE(REPLACE(convert_from(owner_s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || id."ClassObjectID"::text) as owner_class,
            COALESCE(convert_from(field_s."Bytes", 'UTF8'), 'Unknown field') as owner_field,
            pad."ID" as array_id,
            CASE pad."Type"
                WHEN 4 THEN 'boolean[]'
                WHEN 5 THEN 'char[]'
                WHEN 6 THEN 'float[]'
                WHEN 7 THEN 'double[]'
                WHEN 8 THEN 'byte[]'
                WHEN 9 THEN 'short[]'
                WHEN 10 THEN 'int[]'
                WHEN 11 THEN 'long[]'
                ELSE 'unknown[]'
            END as array_type,
            pad."NumberOfElements" as array_elements,
            (? + pad."NumberOfElements" * 
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
            ) as array_size
        FROM "PrimitiveArrayDump" pad
        JOIN "InstanceFieldValues" ifv ON decode(lpad(to_hex(pad."ID"), 16, '0'), 'hex') = ifv."Value" AND ifv."Type" = 2
        JOIN "InstanceDump" id ON ifv."InstanceDumpID" = id."ID"
        JOIN "InstanceFieldRecord" ifr ON ifr."ClassDumpID" = id."ClassObjectID" AND ifr."ID" = ifv."Index" + 1
        LEFT JOIN "LoadClass" owner_lc ON id."ClassObjectID" = owner_lc."ClassObjectID"
        LEFT JOIN "StringInUTF8" owner_s ON owner_lc."ClassNameStringID" = owner_s."StringID"
        LEFT JOIN "StringInUTF8" field_s ON ifr."FieldNameStringID" = field_s."StringID"
        ORDER BY array_size DESC
    `

    var primitiveArrayFieldResults []OwnerArrayResult
    if err := GetDB().Raw(primitiveArrayFieldQuery, ArrayHeaderSize).Scan(&primitiveArrayFieldResults).Error; err != nil {
        result.Body = append(result.Body, fmt.Sprintf("Ошибка при получении примитивных массивов в полях экземпляров: %v\n", err))
    } else {
        allResults = append(allResults, primitiveArrayFieldResults...)
    }

    // 3. Объектные массивы как статические поля
    objectArrayStaticQuery := `
        SELECT DISTINCT
            'StaticField' as owner_type,
            sfr."ClassDumpID" as owner_id,
            COALESCE(REPLACE(convert_from(owner_s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || sfr."ClassDumpID"::text) as owner_class,
            COALESCE(convert_from(field_s."Bytes", 'UTF8'), 'Unknown static field') as owner_field,
            oad."ID" as array_id,
            COALESCE(REPLACE(convert_from(s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad."ArrayClassObjectID"::text) || '[]' as array_type,
            oad."NumberOfElements" as array_elements,
            (? + oad."NumberOfElements" * 8) as array_size
        FROM "ObjectArrayDump" oad
        JOIN "StaticFieldRecord" sfr ON decode(lpad(to_hex(oad."ID"), 16, '0'), 'hex') = sfr."Value" AND sfr."Type" = 2
        LEFT JOIN "LoadClass" lc ON oad."ArrayClassObjectID" = lc."ClassObjectID"
        LEFT JOIN "StringInUTF8" s ON lc."ClassNameStringID" = s."StringID"
        LEFT JOIN "LoadClass" owner_lc ON sfr."ClassDumpID" = owner_lc."ClassObjectID"
        LEFT JOIN "StringInUTF8" owner_s ON owner_lc."ClassNameStringID" = owner_s."StringID"
        LEFT JOIN "StringInUTF8" field_s ON sfr."StaticFieldNameStringID" = field_s."StringID"
        ORDER BY array_size DESC
    `

    var objectArrayStaticResults []OwnerArrayResult
    if err := GetDB().Raw(objectArrayStaticQuery, ArrayHeaderSize).Scan(&objectArrayStaticResults).Error; err != nil {
        result.Body = append(result.Body, fmt.Sprintf("Ошибка при получении объектных массивов в статических полях: %v\n", err))
    } else {
        allResults = append(allResults, objectArrayStaticResults...)
    }

    // 4. Примитивные массивы как статические поля
    primitiveArrayStaticQuery := `
        SELECT DISTINCT
            'StaticField' as owner_type,
            sfr."ClassDumpID" as owner_id,
            COALESCE(REPLACE(convert_from(owner_s."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || sfr."ClassDumpID"::text) as owner_class,
            COALESCE(convert_from(field_s."Bytes", 'UTF8'), 'Unknown static field') as owner_field,
            pad."ID" as array_id,
            CASE pad."Type"
                WHEN 4 THEN 'boolean[]'
                WHEN 5 THEN 'char[]'
                WHEN 6 THEN 'float[]'
                WHEN 7 THEN 'double[]'
                WHEN 8 THEN 'byte[]'
                WHEN 9 THEN 'short[]'
                WHEN 10 THEN 'int[]'
                WHEN 11 THEN 'long[]'
                ELSE 'unknown[]'
            END as array_type,
            pad."NumberOfElements" as array_elements,
            (? + pad."NumberOfElements" * 
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
            ) as array_size
        FROM "PrimitiveArrayDump" pad
        JOIN "StaticFieldRecord" sfr ON decode(lpad(to_hex(pad."ID"), 16, '0'), 'hex') = sfr."Value" AND sfr."Type" = 2
        LEFT JOIN "LoadClass" owner_lc ON sfr."ClassDumpID" = owner_lc."ClassObjectID"
        LEFT JOIN "StringInUTF8" owner_s ON owner_lc."ClassNameStringID" = owner_s."StringID"
        LEFT JOIN "StringInUTF8" field_s ON sfr."StaticFieldNameStringID" = field_s."StringID"
        ORDER BY array_size DESC
    `

    var primitiveArrayStaticResults []OwnerArrayResult
    if err := GetDB().Raw(primitiveArrayStaticQuery, ArrayHeaderSize).Scan(&primitiveArrayStaticResults).Error; err != nil {
        result.Body = append(result.Body, fmt.Sprintf("Ошибка при получении примитивных массивов в статических полях: %v\n", err))
    } else {
        allResults = append(allResults, primitiveArrayStaticResults...)
    }

    // 5. Объектные массивы как элементы других массивов
    objectArrayInArrayQuery := `
        SELECT DISTINCT
            'ArrayElement' as owner_type,
            oad_outer."ID" as owner_id,
            COALESCE(REPLACE(convert_from(s_outer."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad_outer."ArrayClassObjectID"::text) || '[]' as owner_class,
            '[' || oae."Index"::text || ']' as owner_field,
            oad_inner."ID" as array_id,
            COALESCE(REPLACE(convert_from(s_inner."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad_inner."ArrayClassObjectID"::text) || '[]' as array_type,
            oad_inner."NumberOfElements" as array_elements,
            (? + oad_inner."NumberOfElements" * 8) as array_size
        FROM "ObjectArrayElement" oae
        JOIN "ObjectArrayDump" oad_outer ON oae."ObjectArrayDumpID" = oad_outer."ID"
        JOIN "ObjectArrayDump" oad_inner ON oae."InstanceDumpID" = oad_inner."ID"
        LEFT JOIN "LoadClass" lc_inner ON oad_inner."ArrayClassObjectID" = lc_inner."ClassObjectID"
        LEFT JOIN "StringInUTF8" s_inner ON lc_inner."ClassNameStringID" = s_inner."StringID"
        LEFT JOIN "LoadClass" lc_outer ON oad_outer."ArrayClassObjectID" = lc_outer."ClassObjectID"
        LEFT JOIN "StringInUTF8" s_outer ON lc_outer."ClassNameStringID" = s_outer."StringID"
        ORDER BY array_size DESC
    `

    var objectArrayInArrayResults []OwnerArrayResult
    if err := GetDB().Raw(objectArrayInArrayQuery, ArrayHeaderSize).Scan(&objectArrayInArrayResults).Error; err != nil {
        result.Body = append(result.Body, fmt.Sprintf("Ошибка при получении объектных массивов в других массивах: %v\n", err))
    } else {
        allResults = append(allResults, objectArrayInArrayResults...)
    }

    // 6. Примитивные массивы как элементы объектных массивов
    primitiveArrayInArrayQuery := `
        SELECT DISTINCT
            'ArrayElement' as owner_type,
            oad_outer."ID" as owner_id,
            COALESCE(REPLACE(convert_from(s_outer."Bytes", 'UTF8'), '/', '.'), 'Unknown class ' || oad_outer."ArrayClassObjectID"::text) || '[]' as owner_class,
            '[' || oae."Index"::text || ']' as owner_field,
            pad_inner."ID" as array_id,
            CASE pad_inner."Type"
                WHEN 4 THEN 'boolean[]'
                WHEN 5 THEN 'char[]'
                WHEN 6 THEN 'float[]'
                WHEN 7 THEN 'double[]'
                WHEN 8 THEN 'byte[]'
                WHEN 9 THEN 'short[]'
                WHEN 10 THEN 'int[]'
                WHEN 11 THEN 'long[]'
                ELSE 'unknown[]'
            END as array_type,
            pad_inner."NumberOfElements" as array_elements,
            (? + pad_inner."NumberOfElements" * 
                CASE pad_inner."Type"
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
            ) as array_size
        FROM "ObjectArrayElement" oae
        JOIN "ObjectArrayDump" oad_outer ON oae."ObjectArrayDumpID" = oad_outer."ID"
        JOIN "PrimitiveArrayDump" pad_inner ON oae."InstanceDumpID" = pad_inner."ID"
        LEFT JOIN "LoadClass" lc_outer ON oad_outer."ArrayClassObjectID" = lc_outer."ClassObjectID"
        LEFT JOIN "StringInUTF8" s_outer ON lc_outer."ClassNameStringID" = s_outer."StringID"
        ORDER BY array_size DESC
    `

    var primitiveArrayInArrayResults []OwnerArrayResult
    if err := GetDB().Raw(primitiveArrayInArrayQuery, ArrayHeaderSize).Scan(&primitiveArrayInArrayResults).Error; err != nil {
        result.Body = append(result.Body, fmt.Sprintf("Ошибка при получении примитивных массивов в других массивах: %v\n", err))
    } else {
        allResults = append(allResults, primitiveArrayInArrayResults...)
    }

    ownerMap := make(map[string]*OwnerArraysInfo)
    ownerFields := make(map[string]map[string]bool)

    for _, row := range allResults {
        ownerKey := fmt.Sprintf("%s_%d", row.OwnerType, row.OwnerID)

        if _, exists := ownerFields[ownerKey]; !exists {
            ownerFields[ownerKey] = make(map[string]bool)
        }
        ownerFields[ownerKey][row.OwnerField] = true

        if owner, exists := ownerMap[ownerKey]; exists {
            owner.Arrays = append(owner.Arrays, ArrayDetail{
                ArrayID:   row.ArrayID,
                ArrayType: row.ArrayType,
                Elements:  row.ArrayElements,
                Size:      row.ArraySize,
            })
            owner.TotalArrays++
            owner.TotalElements += int64(row.ArrayElements)
            owner.TotalSize += row.ArraySize
        } else {
            ownerMap[ownerKey] = &OwnerArraysInfo{
                OwnerType:  row.OwnerType,
                OwnerID:    row.OwnerID,
                OwnerClass: row.OwnerClass,
                OwnerField: row.OwnerField,
                Arrays: []ArrayDetail{{
                    ArrayID:   row.ArrayID,
                    ArrayType: row.ArrayType,
                    Elements:  row.ArrayElements,
                    Size:      row.ArraySize,
                }},
                TotalArrays:   1,
                TotalElements: int64(row.ArrayElements),
                TotalSize:     row.ArraySize,
            }
        }
    }

    var owners []OwnerArraysInfo
    for ownerKey, owner := range ownerMap {
        fields := make([]string, 0, len(ownerFields[ownerKey]))
        for field := range ownerFields[ownerKey] {
            fields = append(fields, field)
        }
        sort.Strings(fields)
        
        if len(fields) > 1 {
            owner.OwnerField = fmt.Sprintf("множественные поля: %s", strings.Join(fields, ", "))
        } else if len(fields) == 1 {
            owner.OwnerField = fields[0]
        }

        sort.Slice(owner.Arrays, func(i, j int) bool {
            return owner.Arrays[i].Size > owner.Arrays[j].Size
        })
        owners = append(owners, *owner)
    }

    sort.Slice(owners, func(i, j int) bool {
        return owners[i].TotalSize > owners[j].TotalSize
    })

    if len(owners) == 0 {
        result.Body = append(result.Body, "Владельцы массивов не найдены\n")
    } else {
        displayCount := maxOwners
        if len(owners) < displayCount {
            displayCount = len(owners)
        }

        result.Body = append(result.Body, fmt.Sprintf("Найдено %d владельцев массивов, показано топ %d:\n\n", len(owners), displayCount))

        for i := 0; i < displayCount; i++ {
            owner := owners[i]

            ownerDescription := ""
            switch owner.OwnerType {
            case "InstanceField":
                ownerDescription = fmt.Sprintf("Экземпляр '%s' (ID: %d), поля: %s",
                    owner.OwnerClass, owner.OwnerID, owner.OwnerField)
            case "StaticField":
                ownerDescription = fmt.Sprintf("Класс '%s' (ID: %d), статические поля: %s",
                    owner.OwnerClass, owner.OwnerID, owner.OwnerField)
            case "ArrayElement":
                ownerDescription = fmt.Sprintf("Массив '%s' (ID: %d), элементы: %s",
                    owner.OwnerClass, owner.OwnerID, owner.OwnerField)
            }

            result.Body = append(result.Body, fmt.Sprintf("%d. %s\n", i+1, ownerDescription))
            result.Body = append(result.Body, fmt.Sprintf("   Массивов: %d, Всего элементов: %d, Общий размер: %d байт\n",
                owner.TotalArrays, owner.TotalElements, owner.TotalSize))

            arrayCount := maxArraysPerOwner
            if len(owner.Arrays) < arrayCount {
                arrayCount = len(owner.Arrays)
            }

            for j := 0; j < arrayCount; j++ {
                array := owner.Arrays[j]
                result.Body = append(result.Body, fmt.Sprintf("     - ID: %d, Тип: %s, Элементов: %d, Размер: %d байт\n",
                    array.ArrayID, array.ArrayType, array.Elements, array.Size))
            }

            if len(owner.Arrays) > maxArraysPerOwner {
                result.Body = append(result.Body, fmt.Sprintf("     ... и еще %d массивов\n",
                    len(owner.Arrays)-maxArraysPerOwner))
            }

            result.Body = append(result.Body, "\n")
        }
    }

    return result
}