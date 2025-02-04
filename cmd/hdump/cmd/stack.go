package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sreznick/heapmaster/internal/hprof"
)

var stackCmd = &cobra.Command{
	Use:   "stack",
	Short: "Extract call stack from heap dump",
	Long:  `Extract and display call stack records from a given HPROF file.`,
	Run: func(cmd *cobra.Command, args []string) {
		for _, name := range args {
			err := processStackDump(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error processing %s: %v\n", name, err)
			}
		}
	},
}

func ExecuteStack() {
	if err := stackCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func processStackDump(name string) error {
	fmt.Println("Processing stack dump file:", name)

	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("can't open file: %v", err)
	}
	defer f.Close()

	header, err := hprof.ReadHeader(f)
	if err != nil {
		return fmt.Errorf("error reading header: %v", err)
	}
	fmt.Printf("Started at: %s\n", header.TimeStamp)

	idMap := make(map[int64]string)

	stackTraces, stackFrames, _, _, startThreads, endThreads, err := hprof.ProcessRecords(f, idMap)
	if err != nil {
		return fmt.Errorf("error processing records: %v", err)
	}

	threadStatus := make(map[int32]bool)
	for _, startThread := range startThreads {
		threadStatus[startThread.ThreadSerialNumber] = true
	}
	for _, endThread := range endThreads {
		threadStatus[endThread.ThreadSerialNumber] = false
	}

	threadStacks, err := hprof.BuildThreadStacks(stackTraces, stackFrames, threadStatus)
	if err != nil {
		return fmt.Errorf("error building thread stacks: %v", err)
	}

	hprof.PrintStackInfo(stackTraces, stackFrames, threadStacks, idMap)
	return nil
}
