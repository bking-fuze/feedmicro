package main

import (
	"os"
	"fmt"
	"flag"
	"time"
)

func main() {
	startTimeFlag := flag.String("start-time", "", "the RFC3339 start timestamp")
	endTimeFlag := flag.String("end-time", "", "the RFC3339 end timestamp")
	outputFlag := flag.String("output", "", "output path")
	flag.Parse()

	if *startTimeFlag == "" || *endTimeFlag == "" || *outputFlag == "" {
		flag.Usage()
		os.Exit(1)
	}

	startTime, err := time.Parse(time.RFC3339, *startTimeFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid start-time: %s\n", err)
		os.Exit(1)
	}
	endTime, err := time.Parse(time.RFC3339, *endTimeFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: invalid end-time: %s\n", err)
		os.Exit(1)
	}
	if endTime.Sub(startTime) < 0 {
		fmt.Fprintf(os.Stderr, "ERROR: end-time (%s) before start-time (%s)\n", endTime, startTime)
		os.Exit(1)
	}
	output, err := os.Create(*outputFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: couldn't create %s: %s\n", *outputFlag, err)
		os.Exit(1)
	}
	defer output.Close()
	err = stream(startTime, endTime, output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	}
}
