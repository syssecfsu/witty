package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/syssecfsu/witty/term_conn"
	"github.com/syssecfsu/witty/web"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("witty (adduser|deluser|replay|run)")
		return
	}

	var naked bool
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runCmd.BoolVar(&naked, "n", false, "Run WiTTY without user authentication")
	runCmd.BoolVar(&naked, "naked", false, "Run WiTTY without user authentication")

	var wait uint
	replayCmd := flag.NewFlagSet("replay", flag.ExitOnError)
	replayCmd.UintVar(&wait, "w", 2000, "Max wait time between outputs")
	replayCmd.UintVar(&wait, "wait", 2000, "Max wait time between outputs")

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
		replayCmd.Parse(os.Args[2:])

		if len(replayCmd.Args()) != 1 {
			fmt.Println("witty replay <recorded>")
			return
		}

		term_conn.Replay(replayCmd.Arg(0), wait)

	case "run":
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

		web.StartWeb(fp, cmdToExec, naked)

	default:
		fmt.Println("witty (adduser|deluser|replay|run)")
		return
	}

}
