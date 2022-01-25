package term_conn

import (
	"encoding/json"
	"log"
	"os"
)

func Merge(fnames []string, output string) {
	var all_recrods []writeRecord
	var records []writeRecord

	for _, fname := range fnames {
		file, err := os.ReadFile(fname)
		if err != nil {
			log.Println("Failed to read users file", err)
			return
		}

		err = json.Unmarshal(file, &records)

		if err != nil {
			log.Println("Failed to parse json format", err, "for", fname)
			return
		}

		all_recrods = append(all_recrods, records...)
	}

	data, err := json.Marshal(all_recrods)

	if err != nil {
		log.Println("Failed to merge into JSON format", err)
		return
	}

	os.WriteFile(output, data, 0664)
}
