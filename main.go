package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode"
)

func main() {
	s := Settings{}
	// Определяем флаги
	s.ignoreLetters = flag.Bool("a", false, "ignore letters")
	s.ignoreDigits = flag.Bool("n", false, "ignore numbers")
	s.ignoreSymbols = flag.Bool("s", false, "ignore symbols")
	s.caseSensetive = flag.Bool("cs", false, "case sensetive")
	s.countSpace = flag.Bool("sp", false, "count spaces")
	s.limitTop = flag.Int("top", 50, "show top N chars")
	// Парсим флаги
	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s %s\n", os.Args[0], " [flags] <GitRepoURL> <FileMask>")
		flag.PrintDefaults()
	}

	args := flag.Args()

	if len(args) < 2 {
		fmt.Println("Need <GitRepoURL> and <FileMask>")
		flag.Usage()
		os.Exit(1)
	}
	repoURL := args[0]
	fileMask := args[1]

	// init local db
	db, err := initDB("charcounter.db")
	if err != nil {
		fmt.Println("Error init local db: ", err)
		os.Exit(1)
	}
	defer db.Close()

	// read stats from local db
	var readStat = func() map[rune]int {
		stats, err := getStatistics(db, repoURL, fileMask)
		if err != nil {
			fmt.Println("Error read statistics from local db: ", err)
			os.Exit(1)
		}
		return stats
	}
	stats := readStat()

	// empty stat from local db - clone repo and save stat to db
	if len(stats) <= 0 {

		//make temp path to cloned repo
		localRepoPath := filepath.Join(os.TempDir(), "repo")
		fmt.Printf("clone repo %s to: %s\n", repoURL, localRepoPath)

		// clean temp path before clone
		os.RemoveAll(localRepoPath)
		// clean temp path with cloned repo
		defer os.RemoveAll(localRepoPath)
		//clone repo to temp path
		if err := cloneRepo(repoURL, localRepoPath); err != nil {
			fmt.Println("Error clone repo:", err)
			os.Exit(1)
		}

		// read all files from repo
		files, err := getRepoFiles(localRepoPath, "*.*")
		if err != nil {
			fmt.Println("Error reading files from cloned repo:", err)
			os.Exit(1)
		}
		fmt.Printf("\n%v files selected by mask %s\n", len(files), fileMask)

		//make list files by ext
		filesByExt := make(map[string][]string)
		for _, file := range files {
			//ignore non text files by ext
			if isTextFile(file) {
				ext := filepath.Ext(file)
				filesByExt[ext] = append(filesByExt[ext], file)
			}
		}
		fmt.Printf("Found %d file with %d different extensions.\n", len(files), len(filesByExt))

		// count stat in files for every ext
		for ext, files := range filesByExt {
			//count stat for all files by one ext
			frequency := make(map[rune]int)
			for _, file := range files {
				countChars(file, frequency, s)
			}
			fmt.Printf("%d files *%s contents %d different chars.\n", len(files), ext, len(frequency))
			//save stat by ext to local db
			saveStatistics(db, repoURL, "*"+ext, frequency)
		}

		stats = readStat()

	}

	// sort and print results
	printFrequency(stats, s)
}

// check is file has text content by it ext
func isTextFile(filename string) bool {
	// get file ext
	ext := strings.ToLower(filepath.Ext(filename))

	// list text ext
	textExtensions := []string{
		".txt", ".md", ".csv", ".log", ".json", ".xml", ".html", ".css", ".yaml", ".yml", // common text
		".c", ".cpp", ".cxx", ".cc", ".h", ".hpp", // C и C++
		".cs",             // C#
		".java", ".class", // Java
		".py", ".pyc", ".pyo", ".ipynb", // Python
		".js", ".jsx", ".ts", ".tsx", // JavaScript и TypeScript
		".php", ".phtml", // PHP
		".rb", ".erb", // Ruby
		".go",         // Go
		".swift",      // Swift
		".kt", ".kts", // Kotlin
		".r", ".R", ".rmd", // R
		".sh", ".bash", ".zsh", // Shell
		".pas", ".inc", //Pascal
		".pl", ".pm", // Perl
		".hs",         // Haskell
		".rs",         // Rust
		".scala",      // Scala
		".ex", ".exs", // Elixir
		".lua",       // Lua
		".m",         // Objective-C
		".asm", ".s", // Assembly
		".sql",          // SQL
		".clj", ".cljs", // Clojure
		".coffee",                  // CoffeeScript
		".dart",                    // Dart
		".fs", ".fsx", ".fsscript", // F#
	}

	// check if file ext it the ext list
	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}

	return false
}
func cloneRepo(repoURL, localRepoPath string) error {
	cmd := exec.Command("git", "clone", "--depth", "1", repoURL, localRepoPath)
	return cmd.Run()
}

func getRepoFiles(repoPath, fileMask string) ([]string, error) {
	var files []string

	// go to local cloned repo folder
	err := os.Chdir(repoPath)
	if err != nil {
		return nil, err
	}

	// read repo files by mask
	cmd := exec.Command("git", "ls-files", fileMask)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// split text list files to array
	for _, file := range strings.Split(string(output), "\n") {
		if file != "" {
			files = append(files, file)
		}
	}

	return files, nil
}

func countChars(file string, frequency map[rune]int, s Settings) error {
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return err
	}
	text := string(data)
	if !*s.caseSensetive {
		text = strings.ToLower(text)
	}
	for _, char := range text {
		frequency[char]++
	}
	return nil
}

func isIgnoredChar(char rune, s Settings) bool {
	if unicode.IsLetter(char) && *s.ignoreLetters {
		return true
	} else if unicode.IsDigit(char) && *s.ignoreDigits {
		return true
	} else if unicode.IsSymbol(char) && *s.ignoreSymbols {
		return true
	} else if unicode.IsSpace(char) && !*s.countSpace {
		return true
	} else {
		return false
	}
}

func printFrequency(frequency map[rune]int, s Settings) {
	type charFreq struct {
		Char rune
		Freq int
	}

	var sortedFreq []charFreq
	sum := 0.
	sumsel := 0.
	for char, freq := range frequency {
		sortedFreq = append(sortedFreq, charFreq{char, freq})
		if !isIgnoredChar(char, s) {
			sumsel += float64(freq)
		}
		sum += float64(freq)
	}

	sort.Slice(sortedFreq, func(i, j int) bool {
		return sortedFreq[i].Freq > sortedFreq[j].Freq
	})

	fmt.Printf("\n%-7s %-15s %-15s %-15s %s\n", "#", "Frequency", "% of all", "% of selected", "Char")
	fmt.Println(strings.Repeat("-", 60))
	i := 0
	for _, kv := range sortedFreq {
		if !isIgnoredChar(kv.Char, s) {
			i++
			fmt.Printf("%-7d %-15d %-15f %-15f %s\n",
				i,
				kv.Freq,
				float64(kv.Freq)*100/sum,
				float64(kv.Freq)*100/sumsel,
				RuneToStr(kv.Char),
			)
			if i > *s.limitTop {
				break
			}
		}
	}
}

func RuneToStr(char rune) string {
	if unicode.IsSpace(char) {
		code := fmt.Sprintf("\\u%04x ", char)
		switch char {
		case '\t':
			return "\\t " + code
		case '\n':
			return "\\n " + code
		case '\r':
			return "\\r " + code
		case '\v':
			return "\\v " + code
		case '\f':
			return "\\f " + code
		case ' ':
			return "\\s " + code // Для пробела можно использовать \s
		default:
			return "\\s " + code // Для других пробельных символов
		}
	} else {
		return string(char) // Выводим обычный символ
	}
}
