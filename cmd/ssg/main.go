package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"blog-ssg/internal/app/sitegen"
)

func main() {
	var contentPath string
	var outputPath string

	flag.StringVar(&contentPath, "content", "", "path to content directory")
	flag.StringVar(&outputPath, "output", "", "path to output directory")
	flag.Parse()

	if err := run(contentPath, outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func run(contentPath, outputPath string) error {
	if contentPath == "" {
		return errors.New("missing required flag: --content")
	}
	if outputPath == "" {
		return errors.New("missing required flag: --output")
	}

	gen := sitegen.New()
	return gen.Generate(contentPath, outputPath)
}
