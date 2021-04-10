package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const usage = `
Usage: dzip <OUTPUT> <INPUTS>...

dzip creates zip file called <OUTPUT> from all provided <INPUTS>
while stripping down some meta information (like modification time
and permissions on files) in order to make deterministic output. 

This means that same output zip file should be created from same
input always, which is not the case with traditional zip tool. 
`

func main() {
	// first arg is name of the program, second is output path and then list of inputs,
	// so we need 2+ arguments at minimal in order for command to make sense
	if len(os.Args) <= 2 {
		fmt.Println("not enough arguments provided")
		fmt.Print(usage)
		os.Exit(1)
	}
	outputZipName := os.Args[1]
	inputFiles := os.Args[2:]
	// in order to process same inputs in same way, sort them, since changing order
	// changes the hash of output zip file
	sort.Strings(inputFiles)

	if err := createZip(outputZipName, inputFiles); err != nil {
		fmt.Printf("failed creating zip: %v\n", err)
		os.Exit(2)
	}
}

func createZip(output string, inputs []string) error {
	// first make sure that file output file does not already exist, we do not want to overwrite it
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		// only valid error should be IsNotExist, for anything else, we should fail
		return fmt.Errorf("file already exists: %v", output)
	}
	out, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("creating output: %w", err)
	}
	defer func(out *os.File) {
		err := out.Close()
		if err != nil {
			fmt.Printf("failed to close output file: %v", err)
		}
	}(out)

	zipWriter := zip.NewWriter(out)
	defer func(zipWriter *zip.Writer) {
		err := zipWriter.Close()
		if err != nil {
			fmt.Printf("failed to close zip file: %v", err)
		}
	}(zipWriter)


	return addFilesToZip(zipWriter, inputs)
}

func addFilesToZip(writer *zip.Writer, inputs []string) error {
	for _, inFile := range inputs {
		fileStat, err := os.Stat(inFile)
		if err != nil {
			return fmt.Errorf("failed to stat a file: %v", err)
		}
		// handle directories recursively
		if fileStat.IsDir() {
			if err := addDirToZip(writer, inFile); err != nil {
				return err
			}
		} else {
			if err := addSingleFile(writer, inFile); err != nil {
				return err
			}
		}
	}
	return nil
}

func addDirToZip(writer *zip.Writer, path string) error {
	// Note: WalkDir guarantees order
	return filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("failed walking dir: %w", err)
		}
		if !d.IsDir() { // WalkDir handles cases if d.IsDir() is true
			return addSingleFile(writer, path)
		}
		return nil
	})
}


func addSingleFile(zipWriter *zip.Writer, inputFile string) error {
	toAdd, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("faild opening file: %w", err)
	}
	defer func(toAdd *os.File) {
		err := toAdd.Close()
		if err != nil {
			fmt.Printf("failed to close the file: %v\n", err)
		}
	}(toAdd)

	info, err := toAdd.Stat()
	if err != nil {
		return fmt.Errorf("failed stating file: %w", err)
	}
	header, err := createZipHeader(info, inputFile)
	if err != nil {
		return err
	}
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed adding header to zip: %w", err)
	}
	_, err = io.Copy(writer, toAdd)
	if err != nil {
		return fmt.Errorf("failed writing content to zip file: %w", err)
	}

	return nil
}

func createZipHeader(info os.FileInfo, path string) (*zip.FileHeader, error) {
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return header, err
	}

	// set name to full provided path, to preserve dir structure inside the zip
	header.Name = path

	// this is deprecated and suggestion is to use Modified field instead
	// however, in addition to Modified field there are ModifiedTime and ModifiedDate
	// fields (which are also relevant for producing deterministic zip) and which are
	// set by SetModTime method.
	// nolint:staticcheck
	header.SetModTime(time.Time{})

	// TODO: be more intelligent with permissions, maybe allow modification in behavior via flags
	// first reset mode
	header.SetMode(0)
	// if file has exec perm, we want to preserve it
	if info.Mode() & 0111 != 0 {
		header.SetMode(0555)
	} else {
		header.SetMode(0444)
	}

	// actually compress the file
	header.Method = zip.Deflate

	return header, nil
}
