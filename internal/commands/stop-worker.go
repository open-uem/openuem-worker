package commands

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/urfave/cli/v2"
)

func StopWorker() *cli.Command {
	return &cli.Command{
		Name:   "stop",
		Usage:  "Stop worker",
		Action: stopWorker,
	}
}

func stopWorker(cCtx *cli.Context) error {
	pidByte, err := os.ReadFile("PIDFILE")
	if err != nil {
		return fmt.Errorf("could not find the PIDFILE")
	}

	pid, err := strconv.Atoi(string(pidByte))
	if err != nil {
		return fmt.Errorf("could not parse the pid from PIDFILE")
	}
	p, err := os.FindProcess(pid)

	if err != nil {
		return fmt.Errorf("could not find process associated with worker")
	}

	if err := p.Signal(os.Interrupt); err != nil {
		return fmt.Errorf("could not terminate the process associated with the worker, reason: %v", err)
	}

	log.Printf("👋 Done! Your worker has stopped listening\n\n")

	if err := os.Remove("PIDFILE"); err != nil {
		return err
	}
	return nil
}
