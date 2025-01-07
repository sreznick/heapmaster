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
	stackTraces, stackFrames, rootJavaFrames, rootJNILocals, err := hprof.ExtractCallStackRecords(f)
	if err != nil {
		return fmt.Errorf("error extracting call stack records: %v", err)
	}
	fmt.Println("\n--- Stack Traces ---")
	for _, stackTrace := range stackTraces {
		fmt.Printf("StackTrace Serial: %08X, Thread: %08X, Number of Frames: %d\n", stackTrace.StackTraceSerialNumber, stackTrace.ThreadSerialNumber, stackTrace.NumberOfFrames)
		for _, frameID := range stackTrace.FramesID {
			fmt.Printf("  Frame ID: %08X\n", frameID)
		}
	}
	fmt.Println("\n--- Stack Frames ---")
	for _, frame := range stackFrames {
		fmt.Printf("Frame MethodID: %08X, MethodSignatureID: %08X\n", frame.MethodId, frame.MethodSignatureStringId)
	}
	fmt.Println("\n--- Root Java Frames ---")
	for _, rootJavaFrame := range rootJavaFrames {
		fmt.Printf("Root Java Frame ID: %08X\n", rootJavaFrame.FrameNumber)
	}
	fmt.Println("\n--- Root JNI Locals ---")
	for _, rootJNILocal := range rootJNILocals {
		fmt.Printf("Root JNI Local ID: %08X\n", rootJNILocal.FrameNumber)
	}

	return nil
}
