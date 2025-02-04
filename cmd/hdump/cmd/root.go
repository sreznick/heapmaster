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
		case hprof.StringUtf8Tag: 
			utfRecord := &hprof.RecordUtf8{Record: record}
			utfRecord.Init(blob)
			fmt.Printf("utf8: %08X %s\n", utfRecord.Id, utfRecord.Value)
		case hprof.LoadClassTag:
			lcRecord := &hprof.RecordLoadClass{Record: record}
			//utfRecord.Init(blob)
			_ = lcRecord
			fmt.Printf("load class record\n")
        }
	}

	return nil
}
