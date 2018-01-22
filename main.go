package main

import (
	"fmt"

	"github.com/daticahealth/cli/datica"
)

func main() {
	fmt.Println("+-----------------------------------------------------------------------------------------------+")
	fmt.Printf("The CLI does not update automatically across major versions. Please update to the latest version\nto get new functionality and receive updates as they are released.\n\n          https://github.com/daticahealth/cli/releases\n\n")
	fmt.Println("+-----------------------------------------------------------------------------------------------+")
	datica.Run()
}
