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
	DATA_PATH     = "data"
	OUT_PATH      = "out"
)

// sanitizeFileName attempts to produce a filename valid on Windows, macOS, and Linux.
// It removes or replaces invalid characters for Windows (which covers most cases)
// and also removes '/' and '\0', which are invalid on Unix-like systems.
func sanitizeFileName(name string) string {
	// 1. Characters invalid on Windows:
	//    < > : " / \ | ? *
	// 2. On Unix-like (macOS, Linux), '/' is the path separator, and '\0' is never allowed.
	//    We'll remove or replace them for cross-platform safety.

	// We’ll unify them all into a single set of runes to replace.
	// Also note: Some filesystems won't allow leading/trailing spaces or periods (Windows).
	invalidChars := []rune{'<', '>', ':', '"', '/', '\\', '|', '?', '*', '\x00'}

	// Replace these with underscores
	for _, c := range invalidChars {
		name = strings.ReplaceAll(name, string(c), "_")
	}

	// Trim trailing spaces and periods (Windows)
	name = strings.TrimRight(name, " .")

	// (Optional) On Windows, certain names are reserved:
	//   CON, PRN, NUL, AUX, COM1..COM9, LPT1..LPT9, etc.
	// We can rename them to avoid conflicts.
	// This step won't harm on Linux/macOS.
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}
	upperName := strings.ToUpper(name)
	for _, reserved := range reservedNames {
		// If it's exactly reserved or reserved + extension, rename it
		if upperName == reserved || strings.HasPrefix(upperName, reserved+".") {
			name = "_" + name
			break
		}
	}

	// If the name becomes empty, use underscore as fallback
	if name == "" {
		name = "_"
	}

	return name
}

func saveAtt(from string, att *enmime.Part) error {
	ex, err := os.Executable()
	if err != nil {
		return err
	}
	outDir := filepath.Join(filepath.Dir(ex), OUT_PATH, sanitizeFileName(from))
	outPath := filepath.Join(outDir, sanitizeFileName(att.FileName))

	if _, err := os.Stat(outDir); os.IsNotExist(err) {
		err := os.MkdirAll(outDir, 0755)
		if err != nil {
			return err
		}
	}

	log.Println("-- Saving Attachment --------------------------------")
	log.Println(outPath)

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
		log.Printf("Attachment: %s (%d bytes)\n", att.FileName, len(att.Content))

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
			log.Println(fullPath)
			readFile(fullPath)
		}
	}

	return nil
}

func main() {
	err := walkDir(DATA_PATH)
	if err != nil {
		log.Fatalf("Error %s\n", err)
	}
}
