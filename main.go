package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var (
	fJunk      bool
	fOverwrite bool
)

func init() {
	// initialize flags
	flag.BoolVar(&fJunk, "j", false, "junk (don't record) directory names")
	flag.BoolVar(&fOverwrite, "O", false, "overwrite (if exists) output file")

}

const usage = `
Usage: dzip <OUTPUT> <INPUTS>...

dzip creates zip file called <OUTPUT> from all provided <INPUTS>
while stripping down some meta information (like modification time
and permissions on files) in order to make deterministic output. 

This means that same output zip file should be created from same
input always, which is not the case with traditional zip tool. 
`

func main() {
	os.Exit(Main())
}

func Main() int {
	flag.Parse()
	args := flag.Args()

	// first arg is output path and then list of inputs,
	// so we need 2+ arguments at minimal in order for command to make sense
	if len(args) < 2 {
		fmt.Println("not enough arguments provided")
		fmt.Print(usage)
		return 1
	}

	outputZipName := args[0]
	inputFiles := args[1:]
	// in order to process same inputs in same way, sort them, since changing order
	// changes the hash of output zip file
	sort.Strings(inputFiles)

	if err := createZip(outputZipName, inputFiles); err != nil {
		fmt.Printf("failed creating zip: %v\n", err)
		return 2
	}
	return 0
}

func createZip(output string, inputs []string) error {
	// first make sure that file output file does not already exist, we do not want to overwrite it unless flag is given
	outStat, err := os.Stat(output)
	if err != nil {
		// only error that is ok is notExist, for all other fail, since we could not perform stat
		if !os.IsNotExist(err) {
			return fmt.Errorf("failed checking file: %v: %v", output, err)
		}
	} else {
		// file exists, allow continuing only if overwrite flag is provided
		if !fOverwrite {
			return fmt.Errorf("file already exists: %v, provide -O for overwrite", output)
		}
	}

	// even if overwrite is given, we if it is a folder, it is probably an error, so check it here
	if outStat != nil && outStat.IsDir() {
		return fmt.Errorf("got existing directory as output, it should be a file")
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
		if d.IsDir() {
			if fJunk {
				return nil // if we don't record directory names, there is no need to add them either
			}
			// this is just because zip tool has slash at the end for directories
			if !strings.HasSuffix(path, string(filepath.Separator)) {
				path = path + string(filepath.Separator)
			}
			fmt.Println("  adding:", path)
			_, err = writer.Create(path)
			return err
		} else {
			return addSingleFile(writer, path)
		}
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

	// set name to full path on only path name, depending on junk flag
	// if we use full name, structure is preserved within zip file, otherwise only
	// filename is used, which means that flat zip is created.
	if fJunk {
		header.Name = filepath.Base(path)
	} else {
		header.Name = path
	}
	fmt.Println("  adding:", header.Name)

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
	if info.Mode()&0111 != 0 {
		header.SetMode(0555)
	} else {
		header.SetMode(0444)
	}

	// actually compress the file
	header.Method = zip.Deflate

	return header, nil
}
