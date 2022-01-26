package web

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/syssecfsu/witty/term_conn"
)

type RecordedSession struct {
	Fname    string
	Fsize    string
	Duration string
	Time     string
}

// how many seconds of the session
func getDuration(fname string) int64 {
	fp, err := os.Open("./records/" + fname)

	if err != nil {
		log.Println("Failed to open record file", err)
		return 0
	}

	decoder := json.NewDecoder(fp)

	if decoder == nil {
		log.Println("Failed to create JSON decoder")
		return 0
	}

	// To work with javascript decoder, we organize the file as
	// an array of writeRecord. golang decode instead decode
	// as individual record. Call decoder.Token to skip opening [
	decoder.Token()

	var dur int64 = 0

	for decoder.More() {
		var record term_conn.WriteRecord

		if err := decoder.Decode(&record); err != nil {
			log.Println("Failed to decode record", err)
			continue
		}

		dur += record.Dur.Milliseconds()
	}

	return dur/1000 + 1
}

func collectRecords(c *gin.Context, cmd string) (records []RecordedSession) {
	files, err := ioutil.ReadDir("./records/")

	if err == nil {
		for _, finfo := range files {
			fname := finfo.Name()
			if !strings.HasSuffix(fname, ".scr") {
				continue
			}
			fsize := finfo.Size() / 1024
			duration := getDuration(fname)

			records = append(records,
				RecordedSession{
					Fname:    fname,
					Fsize:    strconv.FormatInt(fsize, 10),
					Duration: strconv.FormatInt(duration, 10),
					Time:     finfo.ModTime().Format("Jan/2/2006, 15:04:05"),
				})
		}
	}

	return
}

func startRecord(c *gin.Context) {
	id := c.Param("id")
	term_conn.StartRecord(id)
}

func stopRecord(c *gin.Context) {
	id := c.Param("id")
	term_conn.StopRecord(id)
}

func replayPage(c *gin.Context) {
	id := c.Param("id")
	log.Println("replay/ called with", id)
	c.HTML(http.StatusOK, "replay.html", gin.H{
		"fname": id,
	})
}

func delRec(c *gin.Context) {
	fname := c.Param("fname")
	if err := os.Remove("./records/" + fname); err != nil {
		log.Println("Failed to delete file,", err)
	}
}

func renameRec(c *gin.Context) {
	oldName := "./records/" + c.Param("oldname")
	newName := "./records/" + c.Param("newname")

	if !strings.HasSuffix(newName, ".scr") {
		newName += ".scr"
	}

	if _, err := os.Stat(newName); err == nil {
		log.Println(newName, "already exist, ignore the request")
		return
	}

	if err := os.Rename(oldName, newName); err != nil {
		log.Println("Failed to rename file,", err)
	}
}
