package web

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/sreznick/heapmaster/internal/hprof"
)

func Execute() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Получаем путь к файлу дампа
		fileName := r.URL.Query().Get("file")
		if fileName == "" {
			// Если параметр file не передан, выводим HTML‑форму для ввода параметров
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<html><body>
                <h2>Heapdump Web Interface</h2>
                <form method="GET">
                    <label>Heapdump file:</label><br>
                    <input type="text" name="file" placeholder="/path/to/dump"><br><br>
                    <label>Command (1-8, опционально):</label><br>
                    <input type="text" name="cmd"><br><br>
                    <label>Option (если требуется):</label><br>
                    <input type="text" name="option"><br><br>
                    <input type="submit" value="Submit">
                </form>
            </body></html>`)
			return
		}

		// Открываем файл дампа
		f, err := os.Open(fileName)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error opening file: %v", err), http.StatusInternalServerError)
			return
		}
		defer f.Close()

		// Парсим дамп
		hprof.ParseHeapDump(f)

		// Если передана команда, пробуем выполнить её
		cmdStr := r.URL.Query().Get("cmd")
		if cmdStr != "" {
			cmdNum, err := strconv.Atoi(cmdStr)
			if err != nil || cmdNum < 1 || cmdNum > 8 {
				http.Error(w, "Invalid command number", http.StatusBadRequest)
				return
			}

			var result hprof.AnalyzeResult
			// Если команда требует параметра (команды 1-7), ожидаем значение option
			if cmdNum != 8 {
				optStr := r.URL.Query().Get("option")
				if optStr == "" {
					http.Error(w, "Option parameter required for this command", http.StatusBadRequest)
					return
				}
				opt, err := strconv.Atoi(optStr)
				if err != nil {
					http.Error(w, "Invalid option value", http.StatusBadRequest)
					return
				}
				// Выполнение команды с параметром
				
				switch cmdNum {
				case 1:
					result = hprof.PrintSizeClasses(opt)
				case 2:
					result = hprof.PrintCountInstances(opt)
				case 3:
					result = hprof.PrintObjectLoadersInfo(opt)
				case 4:
					result = hprof.PrintFullClassSize(opt)
				case 5:
					result = hprof.PrintArrayInfo(opt)
				case 6:
					result = hprof.AnalyzeLongArrays(opt)
				case 7:
					result = hprof.AnalyzeHashMapOverheads(opt)
				}
			} else {
				// Команда 8 не требует параметра
				result = hprof.AnalyzeDuplicateStrings()
			}
			fmt.Fprintf(w, `<html><body>
				%s
				</body></html>`, result.ToHTML())
			return
		}

		// Если команда не передана, выводим список доступных команд
		help := `Available commands:
1. Print size classes (requires option)
2. Print count instances (requires option)
3. Print object loaders info (requires option)
4. Print full class size (requires option)
5. Print array info (requires option)
6. Analyze long arrays (requires option)
7. Analyze HashMap overheads (requires option)
8. Analyze duplicate strings (no option required)

Передавайте параметры через GET-запрос, например:
http://localhost:8080/?file=/path/to/dump&cmd=1&option=10`
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body>
            <pre>%s</pre>
            <br>
            <a href="/">Try another action</a>
        </body></html>`, help)
	})

	fmt.Println("Starting web server on http://localhost:8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
