package main

import (
	"context"
	"fmt"

	"go.lepak.sg/mrtracker-backend/data"
	"go.lepak.sg/mrtracker-backend/smrt"
)

func main() {
	names := make([]string, len(data.NS_1))
	for i := range data.NS_1 {
		names[i] = data.NS_1[i].PlatformID()
	}

	results, err := smrt.GetNPlatform(context.Background(), len(names), 100, names...)
	if err != nil {
		panic(err)
	}

	modelLine := smrt.PlatformResultToModel(results, data.NS_1)
	if len(modelLine) != len(data.NS_1) {
		panic(fmt.Sprintf("dim mismatch %d %d", len(modelLine), len(data.NS_1)))
	}

	for i := range modelLine {
		fmt.Printf("%s: %+v\n", data.NS_1[i].Name, modelLine[i])
		fmt.Printf(">> [%v]\n\n", results[data.NS_1[i].Name])
	}
}
