package c6

import (
	"fmt"

	"github.com/go-skynet/go-llama.cpp"
)

// promptTemplate for llama2.
// See https://huggingface.co/blog/llama2#how-to-prompt-llama-2
const promptTemplate = `<s>[INST] <<SYS>>
%v
<</SYS>>

%v [/INST]
`

func Ask(model, question string) error {
	systemPrompt := "You are an SQL query generator for SQLite. Answer only with SQLite queries, no text or newlines before or after."
	prompt := fmt.Sprintf(promptTemplate, systemPrompt, question)

	l, err := llama.New(model, llama.EnableF16Memory, llama.SetContext(4096), llama.SetGPULayers(1))
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
