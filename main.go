package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/jhillyerd/enmime"
)

const (
	EML_EXTENSION = ".eml"
	DATA_PATH     = "./data"
	OUT_PATH      = "./out"
)

func saveAtt(from string, att *enmime.Part) error {
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	outDir := filepath.Join(filepath.Dir(ex), OUT_PATH, from)
	outPath := filepath.Join(outDir, att.FileName)

	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		err := os.MkdirAll(outDir, 0755)
		if err != nil {
			return err
		}
	}

	err = os.WriteFile(outPath, att.Content, 0644)
	if err != nil {
		return err
	}

	return nil
}

func readFile(path string) {
	// Read the raw bytes from the EML file
	raw, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("Error reading file: %v", err)
	}

	// Use enmime to parse the message
	env, err := enmime.ReadEnvelope(strings.NewReader(string(raw)))
	if err != nil {
		log.Fatalf("Error parsing EML: %v", err)
	}

	// Print out some basic info
	fmt.Println("Subject:", env.GetHeader("Subject"))
	fmt.Println("From:   ", env.GetHeader("From"))
	fmt.Println("To:     ", env.GetHeader("To"))

	fmt.Println("\n-- HEADERS --------------------------------------------------------")
	for _, headerKey := range env.GetHeaderKeys() {
		headerValue := env.GetHeaderValues(headerKey)
		fmt.Println(headerKey, headerValue)
	}

	// Print out plain text (if present). `Text` is the concatenation of all plain text parts.
	fmt.Println("\n-- TEXT BODY -----------------------------------------------------")
	fmt.Println(env.Text)

	// Print out HTML (if present). `HTML` is the concatenation of all HTML parts.
	fmt.Println("\n-- HTML BODY -----------------------------------------------------")
	fmt.Println(env.HTML)

	// Process attachments (if any)
	fmt.Println("\n-- ATTACHMENTS ---------------------------------------------------")
	for _, att := range env.Attachments {
		fmt.Printf("Attachment: %s (%d bytes)\n", att.FileName, len(att.Content))

		err = saveAtt(env.GetHeader("From"), att)

		if err != nil {
			log.Printf("Error saving attachment %s: %v\n", att.FileName, err)
		}
	}
}

func walkDir(dir string) error {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	path := filepath.Join(filepath.Dir(ex), dir)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Fatalf("No '%s' directory found.\n", path)
		os.Exit(1)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())
		ext := filepath.Ext(fullPath)

		if entry.IsDir() {
			// Recursively process subdirectory
			if err := walkDir(fullPath); err != nil {
				return err
			}
		} else if ext == EML_EXTENSION {
			fmt.Println(fullPath)
			readFile(fullPath)
		}
	}

	return nil
}

func main() {
	err := walkDir(DATA_PATH)
	if err != nil {
		fmt.Printf("Error %s\n", err)
	}
}
