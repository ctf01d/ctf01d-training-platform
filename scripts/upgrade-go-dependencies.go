//go:build ignore
// +build ignore

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

func main() {
	log.Println("Запускаем обновление зависимостей...")

	// 1 - выполняем go get + go mod tidy
	cmd := exec.Command("bash", "-c", "go get -u ./... && go mod tidy")
	combined, err := cmd.CombinedOutput()
	if err != nil {
		// отдаём вывод, чтобы было понятно, где упало
		log.Fatalf("Ошибка выполнения go get / go mod tidy:\n%s\n%v", string(combined), err)
	}
	output := string(combined)

	// 2 - отбираем строки с «go: upgraded»
	upgradedLines := filterUpgradedLines(output)
	if len(upgradedLines) == 0 {
		// ничего не обновилось - просто выходим
		log.Println("Нет новых версий модулей.")
		return
	}

	// 3 - строим краткую сводку для темы коммита
	summaryParts := make([]string, 0, len(upgradedLines))
	re := regexp.MustCompile(`go: upgraded (\S+) (\S+) => (\S+)`)
	for _, line := range upgradedLines {
		m := re.FindStringSubmatch(line)
		if len(m) != 4 {
			continue
		}
		modulePath := m[1]
		newVer := m[3]

		short := shortName(modulePath)
		summaryParts = append(summaryParts, fmt.Sprintf("%s %s", short, newVer))
	}
	shortInfo := strings.Join(summaryParts, ", ")

	// 4 - формируем сообщение коммита
	var buf bytes.Buffer
	fmt.Fprintf(&buf,
		"build(core): ⬆️ upgrade deps: %s\n\n", shortInfo)
	for _, l := range upgradedLines {
		buf.WriteString(l)
		if !strings.HasSuffix(l, "\n") {
			buf.WriteByte('\n')
		}
	}
	commitMsg := buf.String()

	fmt.Print(commitMsg)

	// 5 - git add go.mod go.sum
	run("git", "add", "go.mod", "go.sum")

	// 6 - git commit (через stdin, чтобы сохранить newlines)
	gitCommit(commitMsg)

	log.Println("Коммит успешно создан.")
}

// filterUpgradedLines выбирает строки, начинающиеся на «go: upgraded».
func filterUpgradedLines(out string) []string {
	lines := strings.Split(out, "\n")
	var res []string
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "go: upgraded") {
			res = append(res, l)
		}
	}
	return res
}

// shortName берёт «protobuf» из «google.golang.org/protobuf»
// (если <4 символов - берёт два последних сегмента).
func shortName(path string) string {
	parts := strings.Split(path, "/")
	last := parts[len(parts)-1]
	if len(last) < 4 && len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}
	return last
}

// gitCommit создаёт коммит, передавая текст через stdin.
func gitCommit(message string) {
	cmd := exec.Command("git", "commit", "-F", "-")
	cmd.Stdin = strings.NewReader(message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("Не удалось сделать коммит: %v", err)
	}
}

// run запускает внешнюю команду и завершает программу при ошибке.
func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Fatalf("%s %v: %v", name, args, err)
	}
}
