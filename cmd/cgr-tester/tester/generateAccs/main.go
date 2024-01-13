package main

import (
	"fmt"
	"os"
)

func main() {
	file, err := os.Create("output.txt")
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	for i := 1000; i <= 9999; i++ {
		line := fmt.Sprintf("cgrates.org,%d,PACKAGE_1001,,,\n", i)
		_, err := file.WriteString(line)
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
	}
}
