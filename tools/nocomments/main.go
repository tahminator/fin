package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var skippedDirs = map[string]bool{
	".git":         true,
	"js":           true,
	"node_modules": true,
	"vendor":       true,
}

func main() {
	root := "."

	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	violations, err := findComments(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	for _, violation := range violations {
		fmt.Println(violation)
	}

	if len(violations) > 0 {
		fmt.Fprintf(os.Stderr, "%d comment(s) found; comments are not allowed\n", len(violations))
		os.Exit(1)
	}
}

func findComments(root string) ([]string, error) {
	var violations []string

	fileSet := token.NewFileSet()

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			if path != root && skippedDirs[entry.Name()] {
				return filepath.SkipDir
			}

			return nil
		}

		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		found, err := commentsIn(fileSet, path)
		if err != nil {
			return err
		}

		violations = append(violations, found...)

		return nil
	})

	return violations, err
}

func commentsIn(fileSet *token.FileSet, path string) ([]string, error) {
	file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	var found []string

	for _, group := range file.Comments {
		for _, comment := range group.List {
			if strings.HasPrefix(comment.Text, "//go:") {
				continue
			}

			found = append(found, fmt.Sprintf("%s: %s", fileSet.Position(comment.Pos()), comment.Text))
		}
	}

	return found, nil
}
