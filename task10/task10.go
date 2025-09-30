package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// MonthOrder сопоставляет названия месяцев с их порядковым номером
var MonthOrder = map[string]int{
	"Jan": 1, "Feb": 2, "Mar": 3, "Apr": 4, "May": 5, "Jun": 6,
	"Jul": 7, "Aug": 8, "Sep": 9, "Oct": 10, "Nov": 11, "Dec": 12,
}

// HumanReadableSuffixes сопоставляет суффиксы с множителями
var HumanReadableSuffixes = map[string]int64{
	"K": 1024, "k": 1024,
	"M": 1024 * 1024, "m": 1024 * 1024,
	"G": 1024 * 1024 * 1024, "g": 1024 * 1024 * 1024,
}

// Config хранит параметры сортировки
type Config struct {
	Column       int
	Numeric      bool
	Reverse      bool
	Unique       bool
	Month        bool
	IgnoreBlanks bool
	Check        bool
	Human        bool
}

// parseArgs парсит флаги командной строки
func parseArgs() (Config, string, error) {
	var cfg Config
	flag.IntVar(&cfg.Column, "k", 0, "column number to sort by (1-based)")
	flag.BoolVar(&cfg.Numeric, "n", false, "sort numerically")
	flag.BoolVar(&cfg.Reverse, "r", false, "reverse order")
	flag.BoolVar(&cfg.Unique, "u", false, "unique lines only")
	flag.BoolVar(&cfg.Month, "M", false, "sort by month name")
	flag.BoolVar(&cfg.IgnoreBlanks, "b", false, "ignore leading blanks")
	flag.BoolVar(&cfg.Check, "c", false, "check if sorted")
	flag.BoolVar(&cfg.Human, "h", false, "human-readable numeric sort (K, M, G)")
	flag.Parse()

	var fileName string
	if flag.NArg() > 0 {
		fileName = flag.Arg(0)
	}

	// Проверка на конфликт флагов
	if cfg.Month && cfg.Numeric {
		return cfg, "", errors.New("cannot combine -M and -n")
	}
	return cfg, fileName, nil
}

// readLines читает строки из файла или STDIN
func readLines(fileName string) ([]string, error) {
	var scanner *bufio.Scanner
	if fileName != "" {
		file, err := os.Open(fileName)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		scanner = bufio.NewScanner(file)
	} else {
		scanner = bufio.NewScanner(os.Stdin)
	}

	// Увеличиваем буфер, чтобы поддерживать длинные строки
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// getKey возвращает ключ для сортировки по конфигурации
func getKey(line string, cfg Config) string {
	if cfg.Column > 0 {
		cols := strings.Split(line, "\t")
		if cfg.Column <= len(cols) {
			line = cols[cfg.Column-1]
		} else {
			line = ""
		}
	}
	if cfg.IgnoreBlanks {
		line = strings.TrimLeft(line, " ")
	}
	return line
}

// parseMonth парсит название месяца
func parseMonth(s string) (int, bool) {
	if len(s) < 3 {
		return 0, false
	}
	c := cases.Title(language.English)
	key := c.String(strings.ToLower(s[:3]))
	val, ok := MonthOrder[key]
	return val, ok
}

// humanToInt конвертирует строку с суффиксом в число
func humanToInt(s string) (int64, error) {
	n := len(s)
	if n == 0 {
		return 0, errors.New("empty string")
	}
	numPart := s[:n-1]
	suffix := s[n-1:]
	if mult, ok := HumanReadableSuffixes[suffix]; ok {
		val, err := strconv.ParseFloat(numPart, 64)
		if err != nil {
			return 0, err
		}
		return int64(val * float64(mult)), nil
	}
	return strconv.ParseInt(s, 10, 64)
}

// compare сравнивает две строки по конфигурации
// возвращает -1 если a<b, 0 если a==b, 1 если a>b
func compare(a, b string, cfg Config) int {
	ka := getKey(a, cfg)
	kb := getKey(b, cfg)

	if cfg.Month {
		ma, oka := parseMonth(ka)
		mb, okb := parseMonth(kb)
		if oka && okb {
			if ma < mb {
				return -1
			} else if ma > mb {
				return 1
			}
			return 0
		}
	}

	if cfg.Numeric {
		af, err1 := strconv.ParseFloat(ka, 64)
		bf, err2 := strconv.ParseFloat(kb, 64)
		if err1 == nil && err2 == nil {
			if af < bf {
				return -1
			} else if af > bf {
				return 1
			}
			return 0
		}
	}

	if cfg.Human {
		ai, err1 := humanToInt(ka)
		bi, err2 := humanToInt(kb)
		if err1 == nil && err2 == nil {
			if ai < bi {
				return -1
			} else if ai > bi {
				return 1
			}
			return 0
		}
	}

	if ka < kb {
		return -1
	} else if ka > kb {
		return 1
	}
	return 0
}

// sortLines выполняет сортировку с учётом флагов
func sortLines(lines []string, cfg Config) []string {
	sort.SliceStable(lines, func(i, j int) bool {
		res := compare(lines[i], lines[j], cfg)
		if cfg.Reverse {
			return res > 0
		}
		return res < 0
	})

	if cfg.Unique {
		lines = uniqueLines(lines)
	}
	return lines
}

// uniqueLines возвращает уникальные строки (после сортировки)
func uniqueLines(lines []string) []string {
	if len(lines) == 0 {
		return lines
	}
	result := []string{lines[0]}
	for i := 1; i < len(lines); i++ {
		if lines[i] != lines[i-1] {
			result = append(result, lines[i])
		}
	}
	return result
}

// checkSorted проверяет, отсортированы ли строки
func checkSorted(lines []string, cfg Config) bool {
	for i := 1; i < len(lines); i++ {
		res := compare(lines[i-1], lines[i], cfg)
		if cfg.Reverse {
			if res < 0 {
				return false
			}
		} else {
			if res > 0 {
				return false
			}
		}
	}
	return true
}

func main() {
	cfg, fileName, err := parseArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	lines, err := readLines(fileName)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading lines:", err)
		os.Exit(1)
	}

	if cfg.Check {
		if checkSorted(lines, cfg) {
			fmt.Println("Sorted")
			os.Exit(0)
		} else {
			fmt.Println("Not sorted")
			os.Exit(1)
		}
	}

	lines = sortLines(lines, cfg)

	for _, line := range lines {
		fmt.Println(line)
	}
}
