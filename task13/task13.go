package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	FieldsRaw string
	Fields    []int
	Delimiter string
	Separated bool
	Flags     uint32
}

type Line struct {
	number int
	text   string
}

const (
	FLAG_FIELDS uint32 = 1 << iota
	FLAG_DELIMITER
	FLAG_SEPARATOR
)

func main() {
	cfg, filename, err := parsingAll()
	if err != nil {
		log.Fatal(err, "Ошибка при передачи флагов")
	}
	lines, err := readLines(filename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading lines:", err)
		os.Exit(1)
	}
	if cfg.Fields == nil {
		log.Fatal("flag -f (fields) is required")
	}
	fmt.Println(cutMachine(lines, cfg))
}

// parsingAll считывает флаги
func parsingAll() (cfg Config, filename string, err error) {
	flag.StringVar(&cfg.FieldsRaw, "f", "", "fields list")
	flag.StringVar(&cfg.Delimiter, "d", "\t", "delimiter")
	flag.BoolVar(&cfg.Separated, "s", false, "only print lines with delimiter")
	flag.Parse()

	arg := flag.Args()
	if len(arg) < 1 {
		return cfg, filename, errors.New("missing filename")
	}
	filename = arg[0]
	cfg.Fields, err = parseFields(cfg.FieldsRaw)
	setFlags(&cfg)
	return cfg, filename, err
}

// readLines читает файл
func readLines(filename string) (lines []Line, err error) {
	var scanner *bufio.Scanner
	if filename != "" {
		file, errOpenFile := os.Open(filename)
		if errOpenFile != nil {
			return lines, errOpenFile
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	lineNumber := 1
	for scanner.Scan() {
		lines = append(lines, Line{
			number: lineNumber,
			text:   scanner.Text(),
		})
		lineNumber++
	}
	return lines, scanner.Err()
}

// cutMachine
func cutMachine(lines []Line, cfg Config) string {
	var result []string

	for _, line := range lines {
		if cfg.Flags&FLAG_SEPARATOR != 0 {
			if !strings.Contains(line.text, cfg.Delimiter) {
				continue
			}
		}

		parts := strings.Split(line.text, cfg.Delimiter)
		var selected []string
		for _, f := range cfg.Fields {
			if f > 0 && f <= len(parts) {
				selected = append(selected, parts[f-1])
			}
		}
		result = append(result, strings.Join(selected, cfg.Delimiter))
	}

	return strings.Join(result, "\n")
}

// setFlags создает битмаску
func setFlags(cfg *Config) {
	cfg.Flags = 0
	if len(cfg.Fields) > 0 {
		cfg.Flags |= FLAG_FIELDS
	}
	if cfg.Delimiter != "" {
		cfg.Flags |= FLAG_DELIMITER
	}
	if cfg.Separated {
		cfg.Flags |= FLAG_SEPARATOR
	}
}

// parseFields
func parseFields(raw string) ([]int, error) {
	if raw == "" {
		return nil, errors.New("no fields provided")
	}

	var result []int
	parts := strings.Split(raw, ",")

	for _, p := range parts {
		if strings.Contains(p, "-") {
			bounds := strings.SplitN(p, "-", 2)
			if len(bounds) != 2 {
				return nil, fmt.Errorf("invalid range: %s", p)
			}
			start, err1 := strconv.Atoi(bounds[0])
			end, err2 := strconv.Atoi(bounds[1])
			if err1 != nil || err2 != nil || start > end {
				return nil, fmt.Errorf("invalid range: %s", p)
			}
			for i := start; i <= end; i++ {
				result = append(result, i)
			}
		} else {
			num, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("invalid field number: %s", p)
			}
			result = append(result, num)
		}
	}
	return result, nil
}
