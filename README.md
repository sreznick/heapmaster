# heapmaster

## Запуск программы

Главный файл с функцией main находится в cmd/hdump/hdump.go.

### Запуск из исходников

``` bash
go run ./cmd/hdump/hdump.go <имя_файла_.hprof>
```

### Сборка и запуск

``` bash
go build -o hdump ./cmd/hdump
./hdump <имя_файла>
```
