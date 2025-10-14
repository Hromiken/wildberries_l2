package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unicode"

	"github.com/chzyer/readline"
)

// currentProcesses — список процессов, которые в текущий момент запущены (для обработки SIGINT)
var currentProcesses []*os.Process
var procMu sync.Mutex

type ConditionalCmd struct {
	Cmd      string
	Operator string // "" / "&&" / "||"
}

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT)

	go func() {
		for range sigChan {
			procMu.Lock()
			procs := append([]*os.Process(nil), currentProcesses...)
			procMu.Unlock()
			for _, p := range procs {
				if p != nil {
					_ = syscall.Kill(-p.Pid, syscall.SIGINT)
				}
			}
			fmt.Println("\n[Ctrl+C] interrupted")
		}
	}()

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		HistoryFile:     "/tmp/shell_history.tmp", // сохраняет историю между сессиями
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "readline error:", err)
		return
	}
	defer rl.Close()

	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			fmt.Println()
			continue
		} else if err == io.EOF {
			fmt.Println("exit")
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		runConditionals(line)
	}
}

// runConditionals разбирает строку на команды с && и ||
func runConditionals(line string) {
	var cmds []ConditionalCmd
	trimmed := strings.TrimSpace(line)

	i := 0
	for i < len(trimmed) {
		var end int
		var op string

		idxAnd := indexOutsideQuotes(trimmed[i:], "&&")
		idxOr := indexOutsideQuotes(trimmed[i:], "||")

		if idxAnd == -1 && idxOr == -1 {
			end = len(trimmed)
			op = ""
		} else if idxAnd != -1 && (idxOr == -1 || idxAnd < idxOr) {
			end = i + idxAnd
			op = "&&"
		} else {
			end = i + idxOr
			op = "||"
		}

		cmdStr := strings.TrimSpace(trimmed[i:end])
		if cmdStr != "" {
			cmds = append(cmds, ConditionalCmd{Cmd: cmdStr, Operator: op})
		}

		i = end + len(op)
	}

	prevSuccess := true
	for _, c := range cmds {
		if c.Operator == "&&" && !prevSuccess {
			prevSuccess = false
			continue
		}
		if c.Operator == "||" && prevSuccess {
			prevSuccess = true
			continue
		}

		var err error
		// Если есть пайплайн
		if strings.Contains(c.Cmd, "|") {
			err = pipeLine(c.Cmd)
		} else {
			fields := splitFieldsRespectingQuotes(c.Cmd)
			if len(fields) == 0 {
				continue
			}

			fields = expandEnvVars(fields)
			fields, stdinFile, stdoutFile := handleRedirection(fields)

			if isBuiltin(fields[0]) {
				err = runBuiltin(fields, stdinFile, stdoutFile)
			} else {
				err = runExternal(fields, stdinFile, stdoutFile)
			}

		}

		prevSuccess = (err == nil)
	}
}

// isBuiltin проверяет, является ли команда встроенной (builtin),
func isBuiltin(cmd string) bool {
	switch cmd {
	case "cd", "pwd", "exit", "help", "echo", "kill", "ps":
		return true
	default:
		return false
	}
}

// runBuiltin — обработка встроенных команд вроде cd, pwd, echo и т.д.
func runBuiltin(fields []string, stdinFile, stdoutFile string) error {
	var output string
	var err error

	switch fields[0] {
	case "cd":
		if len(fields) < 2 {
			home := os.Getenv("HOME")
			if home == "" {
				return fmt.Errorf("cd: missing argument")
			}
			fields = append(fields, home)
		}
		err = os.Chdir(fields[1])

	case "pwd":
		dir, e := os.Getwd()
		if e != nil {
			return e
		}
		output = dir

	case "echo":
		if len(fields) > 1 {
			output = strings.Join(fields[1:], " ")
		}

	case "help":
		output = "Builtins: cd <path>, pwd, echo <args>, kill <pid>, ps, exit, help"

	case "ps":
		cmd := exec.Command("ps", "aux")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()

	case "kill":
		if len(fields) < 2 {
			return fmt.Errorf("kill: missing argument")
		}
		pid, e := strconv.Atoi(fields[1])
		if e != nil {
			return fmt.Errorf("kill: invalid pid: %s", fields[1])
		}
		return syscall.Kill(pid, syscall.SIGTERM)

	case "exit":
		procMu.Lock()
		for _, p := range currentProcesses {
			if p != nil {
				_ = syscall.Kill(-p.Pid, syscall.SIGTERM)
			}
		}
		procMu.Unlock()
		os.Exit(0)
	}

	// Если есть редирект — записываем в файл
	if stdoutFile != "" {
		var f *os.File
		if strings.HasPrefix(stdoutFile, ">>") {
			f, err = os.OpenFile(stdoutFile[2:], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			f, err = os.Create(stdoutFile)
		}
		if err != nil {
			return err
		}
		defer f.Close()
		if output != "" {
			_, err = fmt.Fprintln(f, output)
		}
		return err
	}

	// иначе просто выводим на экран
	if output != "" {
		fmt.Println(output)
	}

	return err
}

// runExternal выполняет внешнюю команду (не builtin).
// Поддерживает перенаправление ввода (<) и вывода (> и >>),
// добавляет процесс в список currentProcesses для обработки Ctrl+C (SIGINT)
func runExternal(fields []string, stdinFile, stdoutFile string) error {
	if len(fields) == 0 {
		return nil
	}

	cmd := exec.Command(fields[0], fields[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// stdin
	if stdinFile != "" {
		inFile, err := os.Open(stdinFile)
		if err != nil {
			return fmt.Errorf("input file error: %v", err)
		}
		defer inFile.Close()
		cmd.Stdin = inFile
	} else {
		cmd.Stdin = os.Stdin
	}

	// stdout
	if stdoutFile != "" {
		var outFile *os.File
		var err error
		if strings.HasPrefix(stdoutFile, ">>") {
			outFile, err = os.OpenFile(stdoutFile[2:], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			outFile, err = os.Create(stdoutFile)
		}
		if err != nil {
			return fmt.Errorf("output file error: %v", err)
		}
		defer outFile.Close()
		cmd.Stdout = outFile
	} else {
		cmd.Stdout = os.Stdout
	}

	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	addCurrentProcess(cmd.Process)
	defer removeCurrentProcess(cmd.Process)

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				return fmt.Errorf("process exited with code %d", status.ExitStatus())
			}
		}
		return err
	}

	return nil
}

// pipeLine принимает строку вида "ps | grep foo | wc -l".
func pipeLine(line string) error {
	parts := strings.Split(line, "|")
	numCmds := len(parts)
	if numCmds == 0 {
		return nil
	}

	cmds := make([]*exec.Cmd, numCmds)
	pipes := make([][2]*os.File, numCmds-1)
	var closers []*os.File

	// Создаём пайпы
	for i := 0; i < numCmds-1; i++ {
		r, w, err := os.Pipe()
		if err != nil {
			return fmt.Errorf("pipe error: %v", err)
		}
		pipes[i] = [2]*os.File{r, w}
	}

	// Настраиваем команды
	for i, part := range parts {
		fields := splitFieldsRespectingQuotes(strings.TrimSpace(part))
		if len(fields) == 0 {
			return fmt.Errorf("empty command in pipeline")
		}

		fields = expandEnvVars(fields)
		fields, stdinFile, stdoutFile := handleRedirection(fields)

		cmd := exec.Command(fields[0], fields[1:]...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Stderr = os.Stderr

		// 🔹 stdin для первой команды
		if i == 0 {
			if stdinFile != "" {
				in, err := os.Open(stdinFile)
				if err != nil {
					return fmt.Errorf("input file error: %v", err)
				}
				cmd.Stdin = in
				closers = append(closers, in)
			} else {
				cmd.Stdin = os.Stdin
			}
		} else {
			cmd.Stdin = pipes[i-1][0]
		}

		// 🔹 stdout для последней команды
		if i == numCmds-1 {
			if stdoutFile != "" {
				var out *os.File
				var err error
				if strings.HasPrefix(stdoutFile, ">>") {
					out, err = os.OpenFile(stdoutFile[2:], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				} else {
					out, err = os.Create(stdoutFile)
				}
				if err != nil {
					return fmt.Errorf("output file error: %v", err)
				}
				cmd.Stdout = out
				closers = append(closers, out)
			} else {
				cmd.Stdout = os.Stdout
			}
		} else {
			cmd.Stdout = pipes[i][1]
		}

		cmds[i] = cmd
	}

	// 🔹 Запускаем команды с откатом при ошибке
	started := []*os.Process{}

	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			// Останавливаем уже запущенные процессы
			for _, p := range started {
				if p != nil {
					_ = syscall.Kill(-p.Pid, syscall.SIGTERM)
					removeCurrentProcess(p)
				}
			}
			return err
		}
		addCurrentProcess(cmd.Process)
		started = append(started, cmd.Process)
	}

	// 🔹 Закрываем копии пайпов в родителе (чтобы дочерние получили EOF)
	for i := 0; i < len(pipes); i++ {
		_ = pipes[i][0].Close()
		_ = pipes[i][1].Close()
	}

	// 🔹 Закрываем все лишние файловые дескрипторы (файлы редиректа)
	for _, f := range closers {
		if f != nil {
			_ = f.Close()
		}
	}

	// 🔹 Ожидаем завершения всех команд пайплайна
	var lastErr error
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			lastErr = err
		}
		removeCurrentProcess(cmd.Process)
	}

	return lastErr

}

// addCurrentProcess сохраняет процесс, чтобы потом можно было его убить при Ctrl+C.
func addCurrentProcess(p *os.Process) {
	if p == nil {
		return
	}
	procMu.Lock()
	currentProcesses = append(currentProcesses, p)
	procMu.Unlock()
}

// removeCurrentProcess удаляет процесс из списка текущих процессов
func removeCurrentProcess(p *os.Process) {
	if p == nil {
		return
	}
	procMu.Lock()
	defer procMu.Unlock()
	newList := currentProcesses[:0]
	for _, pp := range currentProcesses {
		if pp != nil && pp.Pid != p.Pid {
			newList = append(newList, pp)
		}
	}
	currentProcesses = newList
}

// splitFieldsRespectingQuotes разбивает строку на аргументы, уважая кавычки (и экранирование не реализовано)
func splitFieldsRespectingQuotes(s string) []string {
	var res []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false

	for _, r := range s {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if r == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if r == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if unicode.IsSpace(r) && !inSingle && !inDouble {
			if cur.Len() > 0 {
				res = append(res, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		res = append(res, cur.String())
	}
	return res
}

// expandEnvVars подставляет значения переменных окружения в аргументы.
func expandEnvVars(fields []string) []string {
	for i, f := range fields {
		if strings.HasPrefix(f, "$") && len(f) > 1 && !strings.ContainsAny(f, "\"'") {
			val := os.Getenv(f[1:])
			fields[i] = val // если переменной нет, вернёт ""
			continue
		}
		if strings.Contains(f, "$") {
			var out strings.Builder
			j := 0
			for j < len(f) {
				if f[j] == '$' {
					k := j + 1
					for k < len(f) {
						r := rune(f[k])
						if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
							k++
						} else {
							break
						}
					}
					if k > j+1 {
						varName := f[j+1 : k]
						val := os.Getenv(varName)
						out.WriteString(val) // если нет — просто ""
						j = k
						continue
					}
					out.WriteByte('$')
					j++
					continue
				}
				out.WriteByte(f[j])
				j++
			}
			fields[i] = out.String()
		}
	}
	return fields
}

// handleRedirection — убирает символы ">", ">>", "<" и возвращает имя входного/выходного файла (если задано)
func handleRedirection(fields []string) (cmdFields []string, stdinFile, stdoutFile string) {
	cmdFields = []string{}
	for i := 0; i < len(fields); i++ {
		if fields[i] == ">" && i+1 < len(fields) {
			stdoutFile = fields[i+1]
			i++
		} else if fields[i] == ">>" && i+1 < len(fields) {
			stdoutFile = ">>" + fields[i+1] // 🔹 помечаем, что это append
			i++
		} else if fields[i] == "<" && i+1 < len(fields) {
			stdinFile = fields[i+1]
			i++
		} else {
			cmdFields = append(cmdFields, fields[i])
		}
	}
	return
}

// indexOutsideQuotes ищет подстроку sub в s, игнорируя вхождения внутри кавычек
func indexOutsideQuotes(s, sub string) int {
	inSingle := false
	inDouble := false
	for i := 0; i <= len(s)-len(sub); i++ {
		c := s[i]
		// управление состоянием кавычек
		if c == '\\' {
			i++ // пропускаем следующий символ
			continue
		}
		if c == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if c == '\'' && !inDouble {
			inSingle = !inSingle
			continue
		}
		if inSingle || inDouble {
			continue
		}
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
