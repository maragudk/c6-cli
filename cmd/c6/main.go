package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"

	"zombiezen.com/go/sqlite"
	"zombiezen.com/go/sqlite/sqlitex"
)

func main() {
	log := log.New(os.Stderr, "", 0)
	ctx := Context{Log: log}
	if err := start(ctx); err != nil {
		log.Println("Error:", err)
		os.Exit(1)
	}
}

type Context struct {
	Log  *log.Logger
	Args []string
}

func start(ctx Context) error {
	if len(os.Args) < 2 {
		printUsage(ctx)
		return nil
	}

	ctx.Args = os.Args[1:]

	switch os.Args[1] {
	case "ping":
		return ping(ctx)
	case "sql":
		return sql(ctx)
	case "update":
		return update(ctx)
	}
	return nil
}

func ping(ctx Context) error {
	dbPath, err := getDatabasePath()
	if err != nil {
		return err
	}

	conn, err := sqlite.OpenConn(dbPath, sqlite.OpenReadOnly)
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

func sql(ctx Context) error {
	dbPath, err := getDatabasePath()
	if err != nil {
		return err
	}
	cmd := exec.Command("sqlite3", "-readonly", dbPath)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

func update(ctx Context) error {
	ctx.Log.Println("Downloading database…")

	dbPath, err := getDatabasePath()
	if err != nil {
		return err
	}

	c6Dir := path.Dir(dbPath)
	if err := os.MkdirAll(c6Dir, 0755); err != nil {
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

	f, err := os.Create(dbPath + ".tmp")
	if err != nil {
		return fmt.Errorf("cannot create file: %w", err)
	}
	defer func() {
		_ = f.Close()
	}()

	if _, err := io.Copy(f, gzipReader); err != nil {
		return fmt.Errorf("cannot write to file: %w", err)
	}

	if err := os.Rename(dbPath+".tmp", dbPath); err != nil {
		return fmt.Errorf("cannot move database: %w", err)
	}

	ctx.Log.Println("Database downloaded to " + dbPath)

	return nil
}

func getDatabasePath() (string, error) {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot get user home directory: %w", err)
	}
	c6Dir := path.Join(userHomeDir, ".c6")
	dbPath := path.Join(c6Dir, "c6.db")
	return dbPath, nil
}

func printUsage(ctx Context) {
	ctx.Log.Println("Usage: c6 <command>")
	ctx.Log.Println()
	ctx.Log.Println("Commands:")
	ctx.Log.Println("  update    Update the local database")
}
