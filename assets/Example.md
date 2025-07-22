# Пример использования функций для анализа HeapDump

### Пример кода программы, которая сгенерировала HeapDump

```kotlin
fun main() {
    setup()

    val symbols = ('a'..'z') + ('A'..'Z') + ('0'..'9')
    val trash = mutableListOf<String>()
    while (true) {
        trash += (0..1024 * 1024).map { symbols.random() }.joinToString("")
    }
}
```

### Результат выполнения `class.ParseHeapDump(heapDumpFile *os.File)`

#### printSizeClasses(15)
```
Top 15 classes by instance size
1. Class ID: 4292873424, Size: 126602, Name: java/lang/String
2. Class ID: 4292947504, Size: 75572, Name: java/util/concurrent/ConcurrentHashMap$Node
3. Class ID: 4292961392, Size: 54012, Name: java/util/HashMap$Node
4. Class ID: 4292911920, Size: 13152, Name: java/util/HashMap
5. Class ID: 4292921608, Size: 8712, Name: java/lang/module/ModuleDescriptor$Exports
6. Class ID: 4292930016, Size: 8064, Name: java/net/URI
7. Class ID: 4292894360, Size: 8008, Name: java/lang/invoke/MemberName
8. Class ID: 4292894104, Size: 7200, Name: java/lang/invoke/MethodType
9. Class ID: 4292940928, Size: 6912, Name: java/lang/invoke/MethodType$ConcurrentWeakInternSet$WeakEntry
10. Class ID: 4292935464, Size: 6052, Name: java/lang/invoke/LambdaForm$Name
11. Class ID: 4292910128, Size: 5922, Name: java/lang/module/ModuleDescriptor
12. Class ID: 4293080472, Size: 5814, Name: jdk/internal/math/FDBigInteger
13. Class ID: 4292981736, Size: 5292, Name: jdk/internal/module/ModuleReferenceImpl
14. Class ID: 4292899352, Size: 4620, Name: java/util/concurrent/ConcurrentHashMap
15. Class ID: 4292873120, Size: 4485, Name: java/lang/Module
```
#### printCountInstances(15)
```
Top 15 classes by instance count
1. Class ID: 4292873424, Count: 9042, Name: java/lang/String
2. Class ID: 4292947504, Count: 2698, Name: java/util/concurrent/ConcurrentHashMap$Node
3. Class ID: 4292961392, Count: 1928, Name: java/util/HashMap$Node
4. Class ID: 4292921608, Count: 362, Name: java/lang/module/ModuleDescriptor$Exports
5. Class ID: 4293080472, Count: 341, Name: jdk/internal/math/FDBigInteger
6. Class ID: 4292911920, Count: 273, Name: java/util/HashMap
7. Class ID: 4292876960, Count: 268, Name: java/lang/Integer
8. Class ID: 4292877360, Count: 256, Name: java/lang/Long
9. Class ID: 4292876352, Count: 256, Name: java/lang/Byte
10. Class ID: 4292872392, Count: 256, Name: java/lang/Object
11. Class ID: 4292876656, Count: 256, Name: java/lang/Short
12. Class ID: 4292954904, Count: 249, Name: java/util/ImmutableCollections$Set12
13. Class ID: 4292910624, Count: 203, Name: java/util/HashSet
14. Class ID: 4292940928, Count: 191, Name: java/lang/invoke/MethodType$ConcurrentWeakInternSet$WeakEntry
15. Class ID: 4292999176, Count: 187, Name: jdk/internal/module/ServicesCatalog$ServiceProvider
```

#### printObjectLoadersInfo(15)

```
Object loaders info
Loader ID: 0, Name: Bootstrap ClassLoader (System), Number of objects: 1034
		Class ID: 4293029000, Name: java/time/LocalTime
		Class ID: 4292958168, Name: jdk/internal/access/JavaUtilZipFileAccess
		Class ID: 4292943688, Name: java/net/URLStreamHandler
		Class ID: 4292918160, Name: java/io/BufferedOutputStream
		Class ID: 4045605960, Name: [[Ljava/lang/Comparable;
		Class ID: 4045577312, Name: java/lang/management/DefaultPlatformMBeanProvider$4
		Class ID: 4292932976, Name: jdk/internal/reflect/DelegatingConstructorAccessorImpl
		Class ID: 4293077336, Name: java/util/zip/ZipCoder$UTF8ZipCoder
		Class ID: 4292916208, Name: jdk/internal/loader/NativeLibrary
		Class ID: 4293032264, Name: java/time/temporal/ValueRange
		Class ID: 4292981216, Name: java/io/DefaultFileSystem
		Class ID: 4293084096, Name: jdk/internal/org/objectweb/asm/SymbolTable$Entry
		Class ID: 4292976120, Name: java/util/Collections$EmptyIterator
		Class ID: 4292906992, Name: java/util/Spliterator$OfInt
		Class ID: 4045605400, Name: [Lsun/util/calendar/ZoneInfoFile$ZoneOffsetTransitionRule;
		...
Loader ID: 4293225160, Name: jdk/internal/loader/ClassLoaders$AppClassLoader, Number of objects: 56
		Class ID: 4045424888, Name: kotlin/text/StringsKt__IndentKt
		Class ID: 4045428176, Name: kotlin/collections/CollectionsKt__MutableCollectionsJVMKt
		Class ID: 4045442104, Name: kotlin/ranges/CharProgressionIterator
		Class ID: 4045415568, Name: kotlin/random/Random$Default
		Class ID: 4045427840, Name: kotlin/collections/CollectionsKt__IterablesKt
		Class ID: 4045416440, Name: kotlin/random/AbstractPlatformRandom
		Class ID: 4045487816, Name: kotlin/ranges/CharRange$Companion
		Class ID: 4045425336, Name: kotlin/text/StringsKt__StringBuilderKt
		Class ID: 4045427616, Name: kotlin/collections/CollectionsKt__CollectionsJVMKt
		Class ID: 4045412904, Name: kotlin/ranges/IntRange
		Class ID: 4045428288, Name: kotlin/collections/CollectionsKt__MutableCollectionsKt
		Class ID: 4045487400, Name: kotlin/ranges/CharProgression
		Class ID: 4045406704, Name: ru/hse/MainKt
		Class ID: 4045416080, Name: kotlin/internal/jdk8/JDK8PlatformImplementations
		Class ID: 4045487520, Name: kotlin/ranges/CharRange
		...
```
#### printFullClassSize(15)
```
Top 15 classes by full size (with all depends object)
1. Class ID: 4292873424, Size: 116863143, Name: java/lang/String
2. Class ID: 4292947504, Size: 423599, Name: java/util/concurrent/ConcurrentHashMap$Node
3. Class ID: 4292961392, Size: 382839, Name: java/util/HashMap$Node
4. Class ID: 4292872048, Size: 169407, Name: java/lang/System
5. Class ID: 4292998984, Size: 148003, Name: java/util/concurrent/CopyOnWriteArrayList
6. Class ID: 4292873120, Size: 147522, Name: java/lang/Module
7. Class ID: 4292999176, Size: 143827, Name: jdk/internal/module/ServicesCatalog$ServiceProvider
8. Class ID: 4293027832, Size: 137137, Name: sun/util/calendar/ZoneInfoFile
9. Class ID: 4292955040, Size: 136379, Name: java/util/ImmutableCollections$SetN
10. Class ID: 4292945128, Size: 135091, Name: jdk/internal/loader/BuiltinClassLoader$LoadedModule
11. Class ID: 4292954904, Size: 134327, Name: java/util/ImmutableCollections$Set12
12. Class ID: 4292978120, Size: 134088, Name: jdk/internal/module/ArchivedModuleGraph
13. Class ID: 4292921232, Size: 133768, Name: java/lang/ModuleLayer
14. Class ID: 4292978000, Size: 133194, Name: jdk/internal/module/ArchivedBootLayer
15. Class ID: 4292929440, Size: 133183, Name: java/lang/module/Configuration
```
#### printArrayInfo(15)
```
Top 15 arrays by size
1. Array: byte, Size: 117056313
2. Array: [Ljava/lang/Object;, Size: 6662808
3. Array: [Ljava/util/concurrent/ConcurrentHashMap$Node;, Size: 69472
4. Array: [Ljava/util/HashMap$Node;, Size: 57440
5. Array: int, Size: 51176
6. Array: char, Size: 33022
7. Array: [Ljava/lang/ref/SoftReference;, Size: 11880
8. Array: [Ljava/lang/String;, Size: 10992
9. Array: [Ljava/lang/invoke/MethodHandle;, Size: 8840
10. Array: [Ljava/lang/Class;, Size: 5720
11. Array: [Ljava/lang/invoke/LambdaForm$Name;, Size: 3512
12. Array: long, Size: 3016
13. Array: [[B, Size: 2832
14. Array: [Ljdk/internal/math/FDBigInteger;, Size: 2736
15. Array: [Ljava/lang/Integer;, Size: 2064
```
