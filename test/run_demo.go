package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/scopweb/mcp-po-compiler-go/internal/po"
)

func main() {
	svc := po.NewService()
	ctx := context.Background()

	files, _ := filepath.Glob("*.po")
	if len(files) == 0 {
		fmt.Println("No .po files found in current directory")
		os.Exit(1)
	}

	for _, f := range files {
		fmt.Printf("\n=== Testing: %s ===\n", f)

		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Printf("Error reading file: %v\n", err)
			continue
		}

		// Test Summarize
		summary, err := svc.Summarize(ctx, string(content))
		if err != nil {
			fmt.Printf("Summarize error: %v\n", err)
		} else {
			fmt.Printf("Summary: Language=%s, Total=%d, Translated=%d, Untranslated=%d\n",
				summary.Language, summary.Total, summary.Translated, summary.Untranslated)
		}

		// Test Validate
		warnings, _, err := svc.Validate(ctx, string(content))
		if err != nil {
			fmt.Printf("Validate error: %v\n", err)
		} else {
			fmt.Printf("Validation: %d warnings\n", len(warnings))
			for _, w := range warnings {
				fmt.Printf("  - %s\n", w)
			}
		}

		// Test Compile
		result, err := svc.Compile(ctx, string(content), "base64")
		if err != nil {
			fmt.Printf("Compile error: %v\n", err)
		} else {
			fmt.Printf("Compiled: %d bytes (base64)\n", len(result.Base64))

			// Also save to .mo file
			moResult, err := svc.Compile(ctx, string(content), "path")
			if err != nil {
				fmt.Printf("Compile to path error: %v\n", err)
			} else {
				outName := f[:len(f)-3] + ".mo"
				os.Rename(moResult.Path, outName)
				fmt.Printf("Saved: %s\n", outName)
			}
		}
	}

	fmt.Println("\n=== All tests completed ===")
}
