package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"go.lepak.sg/mrtracker-backend/data"
	"go.lepak.sg/mrtracker-backend/smrt"
)

var (
	replay = flag.Bool("replay", false, "")

	l = data.EW_1
)

func main() {
	flag.Parse()
	var results map[string]smrt.Result

	if *replay {
		jsondata, err := os.ReadFile("last.json")
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(jsondata, &results)
		if err != nil {
			panic(err)
		}
		fmt.Println("replaying...")
	} else {
		var err error
		names := make([]string, len(l))
		for i := range l {
			names[i] = l[i].Name
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		results, err = smrt.GetN(ctx, len(names), 100, names...)
		if err != nil {
			panic(err)
		}

		jsondata, err := json.Marshal(results)
		if err != nil {
			panic(err)
		}
		err = os.WriteFile("last.json", jsondata, 0666)
		if err != nil {
			panic(err)
		}
	}

	modelLine := smrt.ToModel(results, l)
	if len(modelLine) != len(l) {
		panic(fmt.Sprintf("dim mismatch %d %d", len(modelLine), len(l)))
	}

	pos := modelLine.ToPosition()

	for i := range pos {
		fmt.Printf("%t\t\t", pos[i])
		if i%2 == 0 {
			fmt.Printf("%s: %+v", l[i/2].Name, modelLine[i/2])
		}
		//fmt.Printf(">> [%v]\n\n", results[data.NS_1[i].Name])
		fmt.Println()
	}
}
