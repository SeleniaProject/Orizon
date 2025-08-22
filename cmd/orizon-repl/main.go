package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/orizon-lang/orizon/internal/cli"
	"github.com/orizon-lang/orizon/internal/lexer"
	p "github.com/orizon-lang/orizon/internal/parser"
)

func main() {
	var (
		showVersion = flag.Bool("version", false, "show version information")
		showHelp    = flag.Bool("help", false, "show help information")
		jsonOutput  = flag.Bool("json", false, "output version in JSON format")
		debugMode   = flag.Bool("debug", false, "enable debug mode")
		noPrompt    = flag.Bool("no-prompt", false, "disable interactive prompt")
		evalStr     = flag.String("eval", "", "evaluate expression and exit")
		loadFile    = flag.String("load", "", "load and execute file before starting REPL")
		historyFile = flag.String("history", ".orizon_history", "history file path")
		maxHistory  = flag.Int("max-history", 1000, "maximum history entries")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Orizon interactive REPL (Read-Eval-Print Loop).\n\n")
		fmt.Fprintf(os.Stderr, "OPTIONS:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nREPL COMMANDS:\n")
		fmt.Fprintf(os.Stderr, "  :help, :h          Show help\n")
		fmt.Fprintf(os.Stderr, "  :quit, :q, :exit   Exit REPL\n")
		fmt.Fprintf(os.Stderr, "  :clear, :c         Clear screen\n")
		fmt.Fprintf(os.Stderr, "  :reset             Reset environment\n")
		fmt.Fprintf(os.Stderr, "  :load <file>       Load and execute file\n")
		fmt.Fprintf(os.Stderr, "  :save <file>       Save current session\n")
		fmt.Fprintf(os.Stderr, "  :history           Show command history\n")
		fmt.Fprintf(os.Stderr, "  :vars              Show current variables\n")
		fmt.Fprintf(os.Stderr, "  :debug on|off      Toggle debug mode\n")
		fmt.Fprintf(os.Stderr, "\nEXAMPLES:\n")
		fmt.Fprintf(os.Stderr, "  %s                     # Start interactive REPL\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --eval \"2 + 3\"      # Evaluate expression\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s --load init.oriz    # Load file and start REPL\n", os.Args[0])
	}

	flag.Parse()

	if *showHelp {
		flag.Usage()
		os.Exit(0)
	}

	if *showVersion {
		cli.PrintVersion("Orizon REPL", *jsonOutput)
		os.Exit(0)
	}

	repl := NewREPL(*debugMode, *historyFile, *maxHistory)

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nGoodbye!")
		repl.SaveHistory()
		os.Exit(0)
	}()

	// Load history
	repl.LoadHistory()

	// Load file if specified
	if *loadFile != "" {
		if err := repl.LoadFile(*loadFile); err != nil {
			cli.ExitWithError("failed to load file %s: %v", *loadFile, err)
		}
	}

	// Evaluate expression if specified
	if *evalStr != "" {
		result, err := repl.Evaluate(*evalStr)
		if err != nil {
			cli.ExitWithError("evaluation failed: %v", err)
		}
		fmt.Println(result)
		os.Exit(0)
	}

	// Start interactive mode
	if !*noPrompt {
		repl.PrintWelcome()
	}

	repl.Run(*noPrompt)
}

type REPL struct {
	debug       bool
	historyFile string
	maxHistory  int
	history     []string
	variables   map[string]interface{}
	scanner     *bufio.Scanner
}

func NewREPL(debug bool, historyFile string, maxHistory int) *REPL {
	return &REPL{
		debug:       debug,
		historyFile: historyFile,
		maxHistory:  maxHistory,
		history:     make([]string, 0),
		variables:   make(map[string]interface{}),
		scanner:     bufio.NewScanner(os.Stdin),
	}
}

func (r *REPL) PrintWelcome() {
	info := cli.GetVersionInfo()
	fmt.Printf("Orizon REPL v%s\n", info.Version)
	fmt.Printf("Type :help for help, :quit to exit\n")
	fmt.Println()
}

func (r *REPL) Run(noPrompt bool) {
	for {
		if !noPrompt {
			fmt.Print("orizon> ")
		}

		if !r.scanner.Scan() {
			break
		}

		line := strings.TrimSpace(r.scanner.Text())
		if line == "" {
			continue
		}

		// Add to history
		r.AddToHistory(line)

		// Handle commands
		if strings.HasPrefix(line, ":") {
			if r.HandleCommand(line) {
				break // Exit requested
			}
			continue
		}

		// Evaluate expression
		result, err := r.Evaluate(line)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		fmt.Printf("=> %s\n", result)
	}

	r.SaveHistory()
}

func (r *REPL) HandleCommand(cmd string) bool {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return false
	}

	switch parts[0] {
	case ":help", ":h":
		r.PrintHelp()
	case ":quit", ":q", ":exit":
		fmt.Println("Goodbye!")
		return true
	case ":clear", ":c":
		fmt.Print("\033[2J\033[H") // Clear screen
	case ":reset":
		r.variables = make(map[string]interface{})
		fmt.Println("Environment reset")
	case ":load":
		if len(parts) < 2 {
			fmt.Println("Usage: :load <file>")
		} else {
			if err := r.LoadFile(parts[1]); err != nil {
				fmt.Printf("Error loading file: %v\n", err)
			}
		}
	case ":save":
		if len(parts) < 2 {
			fmt.Println("Usage: :save <file>")
		} else {
			if err := r.SaveSession(parts[1]); err != nil {
				fmt.Printf("Error saving session: %v\n", err)
			}
		}
	case ":history":
		r.ShowHistory()
	case ":vars":
		r.ShowVariables()
	case ":debug":
		if len(parts) < 2 {
			fmt.Printf("Debug mode: %v\n", r.debug)
		} else {
			switch parts[1] {
			case "on", "true", "1":
				r.debug = true
				fmt.Println("Debug mode enabled")
			case "off", "false", "0":
				r.debug = false
				fmt.Println("Debug mode disabled")
			default:
				fmt.Println("Usage: :debug on|off")
			}
		}
	default:
		fmt.Printf("Unknown command: %s\n", parts[0])
		fmt.Println("Type :help for available commands")
	}

	return false
}

func (r *REPL) PrintHelp() {
	fmt.Println("REPL Commands:")
	fmt.Println("  :help, :h          Show this help")
	fmt.Println("  :quit, :q, :exit   Exit REPL")
	fmt.Println("  :clear, :c         Clear screen")
	fmt.Println("  :reset             Reset environment")
	fmt.Println("  :load <file>       Load and execute file")
	fmt.Println("  :save <file>       Save current session")
	fmt.Println("  :history           Show command history")
	fmt.Println("  :vars              Show current variables")
	fmt.Println("  :debug on|off      Toggle debug mode")
	fmt.Println()
	fmt.Println("Enter Orizon expressions to evaluate them.")
}

func (r *REPL) Evaluate(input string) (string, error) {
	if r.debug {
		fmt.Printf("Debug: Evaluating '%s'\n", input)
	}

	// Lexical analysis
	l := lexer.NewWithFilename(input, "<repl>")
	tokens := []lexer.Token{}

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == lexer.TokenEOF {
			break
		}
	}

	if r.debug {
		fmt.Printf("Debug: Tokens: %+v\n", tokens)
	}

	// Parse the input
	parser := p.NewParser(l, "<repl>")

	program, errors := parser.Parse()
	if len(errors) > 0 {
		return "", fmt.Errorf("parse errors: %v", errors)
	}

	if r.debug {
		fmt.Printf("Debug: AST: %+v\n", program)
	}

	// For now, just return a simple evaluation
	// In a full implementation, this would use an interpreter
	return fmt.Sprintf("Parsed expression: %s", input), nil
}

func (r *REPL) LoadFile(filename string) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	_, err = r.Evaluate(string(content))
	if err != nil {
		return err
	}

	fmt.Printf("Loaded file: %s\n", filename)
	return nil
}

func (r *REPL) SaveSession(filename string) error {
	content := strings.Join(r.history, "\n")
	err := os.WriteFile(filename, []byte(content), 0644)
	if err != nil {
		return err
	}

	fmt.Printf("Session saved to: %s\n", filename)
	return nil
}

func (r *REPL) AddToHistory(line string) {
	r.history = append(r.history, line)
	if len(r.history) > r.maxHistory {
		r.history = r.history[1:]
	}
}

func (r *REPL) ShowHistory() {
	if len(r.history) == 0 {
		fmt.Println("No history")
		return
	}

	fmt.Println("Command history:")
	for i, cmd := range r.history {
		fmt.Printf("%3d: %s\n", i+1, cmd)
	}
}

func (r *REPL) ShowVariables() {
	if len(r.variables) == 0 {
		fmt.Println("No variables defined")
		return
	}

	fmt.Println("Current variables:")
	for name, value := range r.variables {
		fmt.Printf("  %s = %v\n", name, value)
	}
}

func (r *REPL) LoadHistory() {
	content, err := os.ReadFile(r.historyFile)
	if err != nil {
		return // History file doesn't exist yet
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			r.history = append(r.history, line)
		}
	}

	// Trim to max history size
	if len(r.history) > r.maxHistory {
		r.history = r.history[len(r.history)-r.maxHistory:]
	}
}

func (r *REPL) SaveHistory() {
	if len(r.history) == 0 {
		return
	}

	content := strings.Join(r.history, "\n")
	os.WriteFile(r.historyFile, []byte(content), 0644)
}
