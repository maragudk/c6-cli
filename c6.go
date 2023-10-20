package c6

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/go-skynet/go-llama.cpp"
)

type Context struct {
	Args     []string
	Database string
	Dir      string
	Log      *log.Logger
	Model    string
}

// promptTemplate for llama2.
// See https://huggingface.co/blog/llama2#how-to-prompt-llama-2
const promptTemplate = `<s>[INST] <<SYS>>
%v
<</SYS>>

%v [/INST]
`

func Ask(ctx Context, question string) error {
	schema, err := getSchema(ctx.Database)
	if err != nil {
		return err
	}

	systemPrompt := "You are an SQL query generator for SQLite. Answer only with SQLite queries, no text before or after."
	systemPrompt += "\n\nThis is the database schema:\n\n" + schema
	prompt := fmt.Sprintf(promptTemplate, systemPrompt, question)

	l, err := llama.New(ctx.Model, llama.EnableF16Memory, llama.SetContext(4096), llama.SetGPULayers(1))
	if err != nil {
		return err
	}

	opts := []llama.PredictOption{
		llama.SetTokenCallback(func(token string) bool {
			fmt.Print(token)
			return true
		}),
		llama.SetTokens(4000),
		llama.SetThreads(1),
		llama.SetTopK(40),
		llama.SetTopP(0.9),
		llama.SetSeed(-1),
	}

	_, err = l.Predict(prompt, opts...)
	if err != nil {
		return err
	}
	return nil
}

func getSchema(dbPath string) (string, error) {
	cmd := exec.Command("sqlite3", dbPath, ".schema")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("cannot get schema: %w", err)
	}
	return string(output), nil
}
