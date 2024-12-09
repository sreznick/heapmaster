package main

import (
	"github.com/sreznick/heapmaster/cmd/hdump/cmd"
)

func main() {
	cmd.Execute()
/*
	for {
		var tag uint8
		err = binary.Read(f, binary.BigEndian, &tag)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("tag: %d\n", tag)

		var tsd uint32
		err = binary.Read(f, binary.BigEndian, &tsd)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("tsd: %d\n", tsd)

		var rSize uint32
		err = binary.Read(f, binary.BigEndian, &rSize)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("record size: %d\n", rSize)

		b1 = make([]byte, rSize)
		_, err = io.ReadFull(f, b1)
		if err != nil {
			log.Fatal(err)
		}
	}
*/
}
