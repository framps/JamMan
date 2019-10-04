package tools

import (
	"fmt"
	"os"
)

func HandleError(err error) {
	if err != nil {
		fmt.Printf("error %s\n", err.Error())
		os.Exit(42)
	}
}
