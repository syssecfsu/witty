package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/syssecfsu/witty/web"
)

func main() {
	fp, err := os.OpenFile("witty.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)

	if err == nil {
		defer fp.Close()
		log.SetOutput(fp)
	}

	if len(os.Args) < 2 {
		fmt.Println("witty (adduser|deluser|run)")
		return
	}

	var naked bool
	runCmd := flag.NewFlagSet("run", flag.ExitOnError)
	runCmd.BoolVar(&naked, "n", false, "Run WiTTY without user authentication")
	runCmd.BoolVar(&naked, "naked", false, "Run WiTTY without user authentication")

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

	case "run":
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
		fmt.Println("witty (adduser|deluser|run)")
		return
	}

}
