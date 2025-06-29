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

type command struct {
	id int
	description string
	prompt any
	action any
}

var commands = []command{
		{1, "Print size classes", "Enter max count of classes to print: " ,hprof.PrintSizeClasses},
		{2, "Print count instances", "Enter max count of instances to print: ", hprof.PrintCountInstances},
		{3, "Print object loaders info", "Enter max count of loaders to print: ", hprof.PrintObjectLoadersInfo},
		{4, "Print full class size", "Enter max count of classes to print: ", hprof.PrintFullClassSize},
		{5, "Print array info", "Enter max count of arrays to print: ", hprof.PrintArrayInfo},
		{6, "Analyze long arrays", "Enter min size of array: ", hprof.AnalyzeLongArrays},
		{7, "Analyze HashMap overheads", "Enter max count of HashMap: ", hprof.AnalyzeHashMapOverheads},
		{8, "Analyze array owners", "Enter min count of elements in array, witch owners need to print: ", hprof.AnalyzeArrayOwners},
		{9, "Analyze top array owners", "Enter max count of array owners to print: ", hprof.AnalyzeTopArrayOwners},
	}

func getDiscription() string {
	help := "Available commands:\n"
	for _, cmd := range commands {
		help += fmt.Sprintf("%d. %s\n", cmd.id, cmd.description)
	}
	help += "-1 for exit\n"
	help += "Enter the number of the command you want: "
	return help
}

func dumpFile(name string) error {
	fmt.Println("dump", name)
	f, err := os.Open(os.Args[1])
	if err != nil {
		return err
	}
	defer f.Close()
	
	hprof.ParseHeapDump(f);

	help := getDiscription();
	fmt.Print(help);
	var com int
	if _, err := fmt.Scanln(&com); err != nil {
		return err
	}

	for com != -1 {
		com -= 1
		var num int;
		if (commands[com].prompt != nil) {
			fmt.Print(commands[com].prompt)
			if _, err := fmt.Scanln(&num); err != nil {
				return err
			}
			if num < 0 {
				fmt.Println("Invalid number")
				continue
			}
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
		
		if com < 0 || com >= len(commands) {
			fmt.Println("Invalid command")
			continue
		}
		if commands[com].action != nil {
			var result hprof.AnalyzeResult
			switch f := commands[com].action.(type) {
			case func(int) (hprof.AnalyzeResult):
				result = f(num)
			case func() (hprof.AnalyzeResult):
				result = f()
			default:
				fmt.Println("Invalid command")
				continue
			}
			result.Print()
		} 

		fmt.Print(help)
		if _, err := fmt.Scanln(&com); err != nil {
			return err
		}
	}
}
