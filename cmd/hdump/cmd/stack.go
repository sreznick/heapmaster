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
			err := dumpStack(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Problem: %s\n", err)
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

func dumpStack(name string) error {
	fmt.Println("Processing stack dump file:", name)
	f, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("Can't open file: %v", err)
	}
	defer f.Close()

	header, err := hprof.ReadHeader(f)
	if err != nil {
		return fmt.Errorf("error reading header: %v", err)
	}
	fmt.Println("Started at:", header.TimeStamp)

	stackTraces, stackFrames, threadStatus, err := hprof.ExtractCallStackRecords(f)
	if err != nil {
		return fmt.Errorf("error extracting call stack records: %v", err)
	}

	threadStacks, err := hprof.BuildThreadStacks(stackTraces, stackFrames, threadStatus)

	fmt.Println("\n--- Stack Traces ---")
	for _, stackTrace := range stackTraces {
		fmt.Printf("StackTrace Serial: %d, Thread: %d, Number of Frames: %d\n", stackTrace.StackTraceSerialNumber, stackTrace.ThreadSerialNumber, stackTrace.NumberOfFrames)
		for _, frameID := range stackTrace.FramesID {
			fmt.Printf("  Frame ID: %d\n", frameID)
		}
	}
	fmt.Println("\n--- Stack Frames ---")
	for _, frame := range stackFrames {
		fmt.Printf("Frame MethodID: %d, MethodSignatureID: %d, SourceFileNameStringID : %d\n", frame.MethodId, frame.MethodSignatureStringId, frame.SourceFileNameStringId)
	}
	threadStacks, err = hprof.BuildThreadStacks(stackTraces, stackFrames, threadStatus)
	if err != nil {
		return fmt.Errorf("error building thread stacks: %v", err)
	}
	fmt.Println("\n--- Thread Stacks ---")
	for _, threadStack := range threadStacks {
		fmt.Printf("Thread Serial Number: %d, Alive: %t\n", threadStack.ThreadID, threadStack.IsAlive) // добавлен вывод статуса
		if len(threadStack.StackFrames) == 0 {
			fmt.Println("  No frames in this thread stack.")
		} else {
			for _, frame := range threadStack.StackFrames {
				fmt.Printf("  Frame MethodID: %d, MethodSignatureID: %d\n", frame.MethodId, frame.MethodSignatureStringId)
			}
		}
	}
	// fmt.Println("\n--- Root Java Frames ---")
	// for _, rootJavaFrame := range rootJavaFrames {
	// 	fmt.Printf("Root Java Frame ID: %d\n", rootJavaFrame.FrameNumber)
	// }
	// fmt.Println("\n--- Root JNI Locals ---")
	// for _, rootJNILocal := range rootJNILocals {
	// 	fmt.Printf("Root JNI Local ID: %d\n", rootJNILocal.FrameNumber)
	// }

	return nil
}
