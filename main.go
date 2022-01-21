package main

import (
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

	// parse the arguments. User can pass the command to execute
	// by default, we use bash, but macos users might want to use zsh
	// you can also run single program, such as pstree, htop...
	// but program might misbehave (htop seems to be fine)
	var cmdToExec = []string{"bash"}
	args := os.Args

	if len(args) > 1 {
		cmdToExec = args[1:]
		log.Println(cmdToExec)
	}

	web.StartWeb(fp, cmdToExec)
}
