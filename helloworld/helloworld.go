package main

import (
	"fmt"
	"loadtest"
	"time"
)

var theResourceDetails = loadtest.ComputeInstanceDetails{
	AvailabilityDomain: "JtNn:PHX-AD-1",
	CompartmentID:      "ocid1.tenancy.oc1..aaaaaaaahpvug5mdalnf7ixt7sahn7jlhksofqn55uayxjkjj5u6wnpy7mya",
	Shape:              "VM.Standard.E2.1.Micro",
	SubnetID:           "ocid1.subnet.oc1.phx.aaaaaaaalbfufeq57ju67vxt3oc3bk2itenzkw75j3pay4sbkzu6ttojm4tq",
	ImageID:            "ocid1.image.oc1.phx.aaaaaaaactxf4lnfjj6itfnblee3g3uckamdyhqkwfid6wslesdxmlukqvpa"}

func worker(load chan int) {
	w := <-load
	fmt.Println(w)
	//time.Sleep(time.Second)
}

func throttledLoad() {
	load := make(chan int, 10)

	go func() {
		for i := 0; i < 100; i++ {
			for j := 0; j < 10; j++ {
				load <- (i*10 + j)
			}
			time.Sleep(time.Second)
		}
	}()

	for {
		go worker(load)
	}
}
func main() {
	loadtest.AvailabilityDomain = "JtNn:PHX-AD-1"
	loadtest.OCIServiceEndpoint = "iaas.us-phoenix-1.oraclecloud.com"
	loadtest.ClientHost = "lizhuangmac"
	loadtest.CompartmentID = "ocid1.tenancy.oc1..aaaaaaaahpvug5mdalnf7ixt7sahn7jlhksofqn55uayxjkjj5u6wnpy7mya"

	throttledLoad()

	/*
		results, err := loadtest.ListInstance()
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(results)
		}
	*/
	/*
		workrequestID, err := loadtest.CreateInstance(&theResourceDetails)
		if err != nil {
			fmt.Printf("create compute instance failed %v\n", err)
		} else {
			fmt.Printf("workrequest id %s\n", workrequestID)
			for {
				timing, err := loadtest.GetWorkflowRequest(workrequestID)
				if err != nil {
					fmt.Printf("Get work request failed: %v\n", err)
				} else {
					fmt.Printf("%s %s %s\n", timing.TimeAccepted, timing.TimeStarted, timing.TimeFinished)
				}
				if timing.TimeFinished != "" {
					break
				}
				time.Sleep(20 * time.Second)
			}
		}
	*/
	/*
		c := calendar.Calendar{"Goofy Code", false}
		if s, err := json.Marshal(c); err == nil {
			fmt.Printf("Hello World! My calendar is %s\n", s)
		}

		f := -1.122003
		fmt.Println(math.Abs(f))
	*/
}
