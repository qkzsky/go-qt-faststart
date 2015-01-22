// qt-faststart is a Go clone of the original qt-faststart utility written in C
// by Mike Melanson.
//
//		qt-faststart <inFile.mov> <outFile.mov>
//
package main

import (
	"fmt"
	"github.com/DejaMi/go-qt-faststart"
	"io"
	"os"
	"path/filepath"
)

func main() {

	// Make sure we have the correct number of arguments
	if len(os.Args) != 3 {
		fmt.Println("Usage: qt-faststart <inFile.mov> <outFile.mov>")
		return
	}

	// Open the input file
	inFile, err := os.Open(fullPath(os.Args[1]))
	defer inFile.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Open the output file
	outFile, err := os.Create(fullPath(os.Args[2]))
	defer outFile.Close()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Parse the input file
	qtFile, err := qtfaststart.Read(inFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Is conversion necessary?
	if qtFile.FastStartEnabled() {
		fmt.Println("No conversion necessary")

	} else {

		// Perform the conversion
		err = qtFile.Convert(true)
		if err != nil {
			fmt.Println(err)
			return
		}

		// Save the converted movie to the output file
		_, err := io.Copy(outFile, qtFile)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("Conversion complete")
		}
	}
}

func fullPath(input string) string {
	if input[:1] == "/" {
		return input
	}
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, input)
}
