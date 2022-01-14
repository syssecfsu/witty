package main

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"time"

	"golang.org/x/term"
)

type writeRecord struct {
	Dur  time.Duration `json:"Duration"`
	Data []byte        `json:"Data"`
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalln("Usage: replay <recordfile>")
	}

	fp, err := os.Open(os.Args[1])

	if err != nil {
		log.Fatalln("Failed to open record file", err)
	}

	screen := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}

	t := term.NewTerminal(screen, ">")

	if t == nil {
		log.Fatalln("Failed to create terminal")
	}

	w, h, _ := term.GetSize(int(os.Stdout.Fd()))

	if (w != 120) || (h != 36) {
		log.Fatalln("Set terminal window to 120x36 before continue")
	}

	decoder := json.NewDecoder(fp)

	if decoder == nil {
		log.Fatalln("Failed to create JSON decoder")
	}

	for decoder.More() {
		var record writeRecord

		if err := decoder.Decode(&record); err != nil {
			log.Println("Failed to decode record", err)
			continue
		}

		time.Sleep(record.Dur)
		t.Write(record.Data)
	}
}
