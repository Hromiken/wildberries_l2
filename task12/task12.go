package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
)

type Config struct {
	After  int  // -A
	Before int  // -B
	Close  int  // -C
	Count  bool // -c
	Ignore bool // -i
	Invert bool // -v
	Fixed  bool // -F
	Number bool // -n
	Flags  uint32
}

const (
	FLAG_IGNORE uint32 = 1 << iota // -i: игнорировать регистр
	FLAG_INVERT                    // -v: инвертировать фильтр
	FLAG_FIXED                     // -F: фиксированная строка
	FLAG_NUMBER                    // -n: выводить номер строки
	FLAG_AFTER                     // -A N: N строк после
	FLAG_BEFORE                    // -B N: N строк до
	FLAG_CLOSE                     // -C N: контекст вокруг (эквивалент A+B)
	FLAG_COUNT                     // -c: выводить только количество
)

type Line struct {
	Number int
	Text   string
}

type Result struct {
	Index int
	Text  string
}

func main() {
	cfg, filename, err, pattern := parseAll()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	lines, err := readLines(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading lines:", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	jobs := make(chan Line)
	results := make(chan Result, len(lines))
	numWorkers := min(runtime.NumCPU(), len(lines))

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range jobs {
				match(pattern, line, cfg, line.Number-1, results)
			}

		}()
	}
	for _, line := range lines {
		jobs <- Line{Number: line.Number, Text: line.Text}
	}

	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	var output []Result
	for r := range results {
		output = append(output, r)
	}

	unique := make(map[int]bool)
	var filtered []Result
	for _, r := range output {
		if !unique[r.Index] {
			unique[r.Index] = true
			filtered = append(filtered, r)
		}
	}
	output = filtered

	if len(output) == 0 {
		return
	}
	if cfg.Flags&FLAG_COUNT != 0 {
		fmt.Println(len(output))
		return
	}

	context := make(map[int]bool)
	for _, r := range output {
		i := r.Index

		after := cfg.After
		before := cfg.Before
		if cfg.Close > 0 {
			after = cfg.Close
			before = cfg.Close
		}

		start := i - before
		if start < 0 {
			start = 0
		}
		end := i + after
		if end >= len(lines) {
			end = len(lines) - 1
		}
		for j := start; j <= end; j++ {
			context[j] = true
		}
	}

	// собираем итог
	var result []Result
	for idx := range context {
		line := lines[idx]
		text := line.Text
		if cfg.Flags&FLAG_NUMBER != 0 {
			text = fmt.Sprintf("%d:%s", line.Number, line.Text)
		}
		result = append(result, Result{Index: idx, Text: text})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Index < result[j].Index
	})

	for _, r := range result {
		fmt.Println(r.Text)
	}

}

// parseAll считывает флаги и паттерн
func parseAll() (cfg Config, filename string, err error, pattern string) {
	flag.IntVar(&cfg.After, "A", 0, "Print N lines after match")
	flag.IntVar(&cfg.Before, "B", 0, "Print N lines before match")
	flag.IntVar(&cfg.Close, "C", 0, "Print N lines around match (before and after)")
	flag.BoolVar(&cfg.Count, "c", false, "Print only count of matching lines")
	flag.BoolVar(&cfg.Ignore, "i", false, "Ignore case distinctions")
	flag.BoolVar(&cfg.Invert, "v", false, "Invert match: select non-matching lines")
	flag.BoolVar(&cfg.Fixed, "F", false, "Interpret pattern as a fixed string")
	flag.BoolVar(&cfg.Number, "n", false, "Prefix each line with its line number")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		return cfg, "", errors.New("pattern not provided"), ""
	}

	pattern = args[0]
	if len(args) > 1 {
		filename = args[1]
	}

	setFlags(&cfg)
	return cfg, filename, nil, pattern
}

// readLines читает файл
func readLines(fileName string) (line []Line, err error) {
	var scanner *bufio.Scanner
	if fileName != "" {
		file, errOpenFile := os.Open(fileName)
		if errOpenFile != nil {
			return line, errOpenFile
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	var lines []Line
	lineNumber := 1
	for scanner.Scan() {
		lines = append(lines, Line{
			Number: lineNumber,
			Text:   scanner.Text(),
		})
		lineNumber++
	}

	return lines, scanner.Err()
}

// match ядро программы.
func match(pattern string, l Line, cfg Config, index int, ch chan Result) {
	text := l.Text
	pat := pattern

	if cfg.Flags&FLAG_IGNORE != 0 {
		text = strings.ToLower(text)
		pat = strings.ToLower(pat)
	}

	var matched bool
	if cfg.Flags&FLAG_FIXED != 0 {
		matched = strings.Contains(text, pat)
	} else {
		re, err := regexp.Compile(pat)
		if err != nil {
			return
		}
		matched = re.MatchString(text)
	}

	if cfg.Flags&FLAG_INVERT != 0 {
		matched = !matched
	}

	if matched {
		if cfg.Flags&FLAG_NUMBER != 0 {
			ch <- Result{Index: index, Text: fmt.Sprintf("%d:%s", l.Number, l.Text)}
		} else {
			ch <- Result{Index: index, Text: l.Text}
		}
	}

}

// setFlags делает битмаску для определения работы match
func setFlags(cfg *Config) {
	cfg.Flags = 0

	if cfg.Ignore {
		cfg.Flags |= FLAG_IGNORE
	}
	if cfg.Invert {
		cfg.Flags |= FLAG_INVERT
	}
	if cfg.Fixed {
		cfg.Flags |= FLAG_FIXED
	}
	if cfg.Number {
		cfg.Flags |= FLAG_NUMBER
	}
	if cfg.After > 0 {
		cfg.Flags |= FLAG_AFTER
	}
	if cfg.Before > 0 {
		cfg.Flags |= FLAG_BEFORE
	}
	if cfg.Close > 0 {
		cfg.Flags |= FLAG_CLOSE
	}
	if cfg.Count {
		cfg.Flags |= FLAG_COUNT
	}
}
