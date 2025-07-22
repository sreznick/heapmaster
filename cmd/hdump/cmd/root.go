package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	//	"github.com/spf13/viper"

	"github.com/sreznick/heapmaster/internal/hprof"
)

var rootCmd = &cobra.Command{
	Use:   "hdump",
	Short: "Output hprof dump",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		for _, name := range args {
			err := dumpFile(name)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Problem : %s\n", err)
			}
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dumpFile(name string) error {
	fmt.Println("dump", name)
	f, err := os.Open(os.Args[1])
	if err != nil {
		return err
	}
	defer f.Close()
	header, err := hprof.ReadHeader(f)
	if err != nil {
		return err
	}
	fmt.Println("started at", header.TimeStamp)

	for {
		record, blob, err := hprof.ReadRecord(f, header)
		if err != nil {
			return err
		}
		switch record.Tag {
		case hprof.Utf8:
			utfRecord := &hprof.RecordUtf8{Record: record}
			utfRecord.Init(blob)
			fmt.Printf("utf8: %08X %s\n", utfRecord.Id, utfRecord.Value)
		case hprof.TagLoadClass:
			lcRecord := &hprof.RecordLoadClass{Record: record}
			err := lcRecord.Init(blob)
			if err != nil {
				fmt.Printf("Error initializing LoadClass record: %v\n", err)
			}
			fmt.Printf("Load Class Record:\n")
			fmt.Printf("  ClassSerial: %08X\n", lcRecord.ClassSerial)
			fmt.Printf("  ObjectId: %016X\n", lcRecord.ObjectId)
			fmt.Printf("  StackTraceSerial: %08X\n", lcRecord.StackTraceSerial)
			fmt.Printf("  NameId: %016X\n", lcRecord.NameId)
		}
	}
}
