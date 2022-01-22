package term_conn

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

func Replay(fname string, wait uint) {
	fp, err := os.Open(fname)

	if err != nil {
		log.Fatalln("Failed to open record file", err)
	}

	screen := struct {
		io.Reader
		io.Writer
	}{os.Stdin, os.Stdout}

	t := term.NewTerminal(screen, "$")

	if t == nil {
		log.Fatalln("Failed to create terminal")
	}

	w, h, _ := term.GetSize(int(os.Stdout.Fd()))

	if (w != 120) || (h != 36) {
		log.Println("Set terminal window to 120x36 before continue")
	}

	decoder := json.NewDecoder(fp)

	if decoder == nil {
		log.Fatalln("Failed to create JSON decoder")
	}

	// To work with javascript decoder, we organize the file as
	// an array of writeRecord. golang decode instead decode
	// as individual record. Call decoder.Token to skip opening [
	t.Write([]byte("\n\n---beginning of replay---\n\n"))

	decoder.Token()
	for decoder.More() {
		var record writeRecord

		if err := decoder.Decode(&record); err != nil {
			log.Println("Failed to decode record", err)
			continue
		}

		if record.Dur > time.Duration(wait)*time.Millisecond {
			record.Dur = time.Duration(wait) * time.Millisecond
		}

		time.Sleep(record.Dur)
		t.Write(record.Data)
	}

	t.Write([]byte("\n\n---end of replay---\n\n"))
}
