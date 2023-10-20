package main

import (
	"compress/gzip"
	_ "embed"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/maragudk/env"
	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"

	"github.com/c6dk/c6-cli"
)

//go:embed ggml-metal.metal
var metal []byte

func main() {
	log := log.New(os.Stderr, "", 0)

	if err := start(log); err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}
}

func start(log *log.Logger) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot get user home directory: %w", err)
	}

	c6Dir := env.GetStringOrDefault("C6_DIR", path.Join(homeDir, ".c6"))
	ctx := c6.Context{
		Database: env.GetStringOrDefault("C6_DATABASE", path.Join(c6Dir, "c6.db")),
		Dir:      c6Dir,
		Log:      log,
		Model:    env.GetStringOrDefault("C6_LLM", path.Join(c6Dir, "codellama-7b-instruct.Q4_K_M.gguf")),
	}

	if len(os.Args) < 2 {
		printUsage(ctx)
		return nil
	}

	ctx.Args = os.Args[1:]

	switch os.Args[1] {
	case "ask":
		return ask(ctx)
	case "ping":
		return ping(ctx)
	case "sql":
		return sql(ctx)
	case "update":
		return update(ctx)
	}
	return nil
}

func ask(ctx c6.Context) error {
	if len(ctx.Args) < 2 {
		printUsage(ctx)
		return nil
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot get executable path: %w", err)
	}

	// Write the metal file to disk.
	// This is needed for llama2 to work.
	metalPath := path.Join(path.Dir(executable), "ggml-metal.metal")
	metalF, err := os.Create(metalPath)
	if err != nil {
		return fmt.Errorf("cannot create metal file: %w", err)
	}
	defer func() {
		if err := metalF.Close(); err != nil {
			ctx.Log.Println("Error closing metal file:", err)
		}
		if err := os.Remove(metalPath); err != nil {
			ctx.Log.Println("Error removing metal file:", err)
		}
	}()

	if _, err := metalF.Write(metal); err != nil {
		return fmt.Errorf("cannot write metal file: %w", err)
	}

	question := ctx.Args[1]

	return c6.Ask(ctx, question)
}

func ping(ctx c6.Context) error {
	conn, err := sqlite.OpenConn(ctx.Database, sqlite.OpenReadOnly)
	if err != nil {
		return fmt.Errorf("cannot open database: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	if err := sqlitex.ExecuteTransient(conn, "select 1", nil); err != nil {
		return fmt.Errorf("cannot ping database: %w", err)
	}

	ctx.Log.Println("Pong!")

	return nil
}

func sql(ctx c6.Context) error {
	cmd := exec.Command("sqlite3", "-readonly", "-column", ctx.Database)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

func update(ctx c6.Context) error {
	ctx.Log.Println("Downloading database…")

	if err := os.MkdirAll(ctx.Dir, 0755); err != nil {
		return fmt.Errorf("cannot create directory: %w", err)
	}

	res, err := http.Get("https://assets.c6.dk/c6.db.gz")
	if err != nil {
		return fmt.Errorf("cannot download database: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("cannot download database, got HTTP status code %v", res.Status)
	}

	gzipReader, err := gzip.NewReader(res.Body)
	if err != nil {
		return fmt.Errorf("cannot decompress database: %w", err)
	}
	defer func() {
		_ = gzipReader.Close()
	}()

	f, err := os.Create(ctx.Database + ".tmp")
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	if _, err := io.Copy(f, gzipReader); err != nil {
		return fmt.Errorf("cannot write to file: %w", err)
	}

	if err := os.Rename(ctx.Database+".tmp", ctx.Database); err != nil {
		return fmt.Errorf("cannot move database: %w", err)
	}

	ctx.Log.Println("Database downloaded to " + ctx.Database)

	ctx.Log.Println("Downloading LLM…")

	res, err = http.Get("https://assets.c6.dk/" + ctx.Model)
	if err != nil {
		return fmt.Errorf("cannot download llm: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("cannot download database, got HTTP status code %v", res.Status)
	}

	f, err = os.Create(ctx.Model + ".tmp")
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	if _, err := io.Copy(f, res.Body); err != nil {
		return fmt.Errorf("cannot write to file: %w", err)
	}

	if err := os.Rename(ctx.Model+".tmp", ctx.Model); err != nil {
		return fmt.Errorf("cannot move llm: %w", err)
	}

	ctx.Log.Println("LLM downloaded to " + ctx.Model)

	return nil
}

func printUsage(ctx c6.Context) {
	if len(ctx.Args) == 0 {
		ctx.Log.Println("Usage: c6 <command>")
		ctx.Log.Println()
		ctx.Log.Println("Commands:")
		ctx.Log.Println("  ask       Ask the local database with natural language")
		ctx.Log.Println("  ping      Ping the local database")
		ctx.Log.Println("  sql       Open an SQLite shell")
		ctx.Log.Println("  update    Update the local database and LLM")
		return
	}

	switch ctx.Args[0] {
	case "ask":
		ctx.Log.Println("Usage: c6 ask <question>")
	}
}
