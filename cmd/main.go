package main

import (
	"fmt"
	"prob_info_screen/config"
)

func main() {

	config := config.MustLoadConfig()

	fmt.Printf("Config: %+v\n", config)
}
