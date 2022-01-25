package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/syssecfsu/witty/term_conn"
	"github.com/syssecfsu/witty/web"
)

const (
	subcmds = "witty (adduser|deluser|listusers|replay|merge|run)"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println(subcmds)
		return
	}

	switch os.Args[1] {
	case "adduser":
		if len(os.Args) != 3 {
			fmt.Println("witty adduser <username>")
			return
		}
		web.AddUser(os.Args[2])

	case "deluser":
		if len(os.Args) != 3 {
			fmt.Println("witty deluser <username>")
			return
		}
		web.DelUser(os.Args[2])

	case "listusers":
		web.ListUsers()

	case "replay":
		var wait uint
		replayCmd := flag.NewFlagSet("replay", flag.ExitOnError)
		replayCmd.UintVar(&wait, "w", 2000, "Max wait time between outputs")
		replayCmd.UintVar(&wait, "wait", 2000, "Max wait time between outputs")

		replayCmd.Parse(os.Args[2:])

		if len(replayCmd.Args()) != 1 {
			fmt.Println("witty replay <recorded>")
			return
		}

		term_conn.Replay(replayCmd.Arg(0), wait)

	case "merge":
		var output string

		defName := "merged_" + strconv.FormatInt(time.Now().Unix(), 16) + ".scr"

		mergeCmd := flag.NewFlagSet("merge", flag.ExitOnError)
		mergeCmd.StringVar(&output, "o", defName, "Set the name of merged files")
		mergeCmd.StringVar(&output, "output", defName, "Set the name of merged files")

		mergeCmd.Parse(os.Args[2:])

		if len(mergeCmd.Args()) < 2 {
			fmt.Println("witty merge -o output_file file1 file2 ... (at least two files)")
			return
		}

		term_conn.Merge(mergeCmd.Args(), output)

	case "run":
		var naked bool
		var port uint
		runCmd := flag.NewFlagSet("run", flag.ExitOnError)
		runCmd.BoolVar(&naked, "n", false, "Run WiTTY without user authentication")
		runCmd.BoolVar(&naked, "naked", false, "Run WiTTY without user authentication")
		runCmd.UintVar(&port, "p", 8080, "Port number to listen on")
		runCmd.UintVar(&port, "port", 8080, "Port number to listen on")

		fp, err := os.OpenFile("witty.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)

		if err == nil {
			defer fp.Close()
			log.SetOutput(fp)
		}

		runCmd.Parse(os.Args[2:])

		var cmdToExec []string
		args := runCmd.Args()
		if len(args) > 0 {
			cmdToExec = args
		} else {
			cmdToExec = []string{"bash"}
		}

		web.StartWeb(fp, cmdToExec, naked, uint16(port))

	default:
		fmt.Println(subcmds)
		return
	}

}
