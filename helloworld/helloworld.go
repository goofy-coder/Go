package main

import (
	"fmt"
	"calendar"
	"encoding/json"
	"math"
)

func main() {
	c := calendar.Calendar {"Goofy Code", false}
	if s, err := json.Marshal(c); err == nil {	
		fmt.Printf("Hello World! My calendar is %s\n", s)
	}

	f  := -1.122003
	fmt.Println(math.Abs(f))
}
