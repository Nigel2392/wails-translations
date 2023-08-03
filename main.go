package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/franciscoescher/goopenai"

	_ "embed"
)

type file struct {
	path    string
	newText string
}

//go:embed completions.json
var completionsJSON []byte
var globalCompletions = make([]goopenai.CreateCompletionsResponse, 0)

func init() {
	var err = json.Unmarshal(completionsJSON, &globalCompletions)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	go func() {
		var sig = make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("Saving completions...")
		var data, err = json.MarshalIndent(globalCompletions, "", "  ")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = os.WriteFile("completions.json", data, 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Done!")
		os.Exit(0)
	}()
}

func main() {
	var err error
	apiKey := "-"
	organization := "-"

	client := goopenai.NewClient(apiKey, organization)

	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <language to translate> <dir to translate>")
		os.Exit(1)
	} else if len(os.Args) > 3 {
		var reset = os.Args[3]
		if reset == "reset" {
			fmt.Println("Resetting completions...")
			globalCompletions = make([]goopenai.CreateCompletionsResponse, 0)
			os.Exit(0)
		}
	}

	lang := os.Args[1]
	dir := os.Args[2]

	fmt.Printf("Translating to %s...\n", lang)
	time.Sleep(2 * time.Second)

	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer func() {
		r := recover()
		if r != nil {
			fmt.Println(r)
			os.Exit(1)
		}
	}()

	var translatedFiles = make([]*file, 0)
	err = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Println("Error walking directory")
			fmt.Println(err)
			os.Exit(1)
		}

		var f *file = new(file)
		var name = info.Name()
		if !strings.HasSuffix(name, ".mdx") {
			fmt.Printf("Skipping file %s...\n", info.Name())
			return nil
		}

		rel, err := (filepath.Rel(abs, path))
		if err != nil {
			fmt.Println("Error getting relative path")
			fmt.Println(err)
			os.Exit(1)
		}

		f.path = filepath.Join(fmt.Sprintf("docs-%s", lang), rel)

		if fInfo, err := os.Stat(f.path); err == nil && fInfo.Size() > 0 {
			fmt.Printf("Skipping file %s because it might already be present (optionally reset)...\n", info.Name())
			return nil
		}

		translatedFiles = append(translatedFiles, f)

		if info.IsDir() {
			fmt.Println("Skipping directory...")
			return nil
		}

		var data = make([]byte, info.Size())
		file, err := os.Open(path)
		if err != nil {
			fmt.Println("Error opening file")
			fmt.Println(err)
			os.Exit(1)
		}
		defer file.Close()
		_, err = file.Read(data)
		if err != nil {
			fmt.Println("Error reading file")
			fmt.Println(err)
			os.Exit(1)
		}

		var completions goopenai.CreateCompletionsResponse

		var b = new(strings.Builder)
		var split = bytes.Split(data, []byte("{/* GIPPITY SPLITTER */}"))
		for i, data := range split {
			r := goopenai.CreateCompletionsRequest{
				Model: "gpt-3.5-turbo-16k-0613",
				Messages: []goopenai.Message{
					{
						Role:    "user",
						Content: fmt.Sprintf("Translate the following text to %s:\n\n%s", lang, string(data)),
					},
				},
				Temperature: 0.7,
			}

			fmt.Printf("Creating completions with content length (%s): %d\n", f.path, len(data))
			completions, err = client.CreateCompletions(context.Background(), r)
			if err != nil {
				f.newText = string(data)
				fmt.Println("Error creating completions")
				fmt.Println(err)
				os.Exit(1)
			}

			fmt.Println(completions)

			fmt.Println(completions)
			if len(completions.Choices) == 0 {
				fmt.Println("Skipping file because it has no completions...")
				b.WriteString(string(data))
				return nil
			}

			globalCompletions = append(globalCompletions, completions)
			b.WriteString(completions.Choices[0].Message.Content)
			if i < len(split)-1 {
				b.WriteString("\n\n")
			}
		}

		var dir = filepath.Dir(f.path)
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Printf("Writing to file %s...\n", f.path)
		file, err = os.Create(f.path)
		if err != nil {
			f.newText = string(data)
			fmt.Println(err)
			os.Exit(1)
		}
		defer file.Close()
		io.Copy(file, strings.NewReader(b.String()))
		return nil
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
