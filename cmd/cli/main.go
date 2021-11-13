package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"go.lepak.sg/mrtracker-backend/data"
	"go.lepak.sg/mrtracker-backend/model"
	"go.lepak.sg/mrtracker-backend/smrt"
)

var (
	verbose = flag.Bool("v", false, "verbose mode")
	timeout = flag.String("t", "10s", "timeout")
	refresh = flag.String("r", "30s", "refresh interval")
)

func main() {
	flag.Parse()

	if !*verbose {
		// suppress noisy log output
		log.Default().SetOutput(io.Discard)
	}

	ctxTimeout, err := time.ParseDuration(*timeout)
	if err != nil {
		fmt.Printf("invalid timeout: %v\n", err)
		os.Exit(1)
	}

	refreshDur, err := time.ParseDuration(*refresh)
	if err != nil {
		fmt.Printf("invalid refresh: %v\n", err)
		os.Exit(1)
	}

	if ctxTimeout > refreshDur {
		ctxTimeout = refreshDur
		fmt.Printf("clamping timeout to refresh %s\n", refreshDur.String())
	}

	nameSet := make(map[string]struct{})
	lineSource := []data.Line{data.EW_1, data.NS_1, data.CG_1}

	for _, l := range lineSource {
		for i := range l {
			nameSet[l[i].Name] = struct{}{}
		}
	}

	var names []string
	for k := range nameSet {
		names = append(names, k)
	}

	timer := time.Tick(refreshDur)

	for {
		func(names []string, ctxTimeout time.Duration) {
			ctx, cancel := context.WithTimeout(context.Background(), ctxTimeout)
			defer cancel()

			fmt.Println("loading...")

			results, _, err := smrt.GetN(ctx, 0, 10, names...)
			if err != nil {
				fmt.Println("error:", err)
				return
			}

			positions := map[string]model.Position{
				"ns1": smrt.ToModel(results, data.NS_1).ToPosition(),
				"ns2": smrt.ToModel(results, data.NS_2).ToPosition(),
				"ew1": smrt.ToModel(results, data.EW_1).ToPosition(),
				"ew2": smrt.ToModel(results, data.EW_2).ToPosition(),
				"cg1": smrt.ToModel(results, data.CG_1).ToPosition(),
				"cg2": smrt.ToModel(results, data.CG_2).ToPosition(),
			}

			fmt.Print("\033[H\033[2J") // clear terminal

			fmt.Println(formatPair(data.NS_1, positions["ns1"].ToString(), positions["ns2"].Reverse().ToString()))
			fmt.Println(formatPair(data.EW_1, positions["ew1"].ToString(), positions["ew2"].Reverse().ToString()))
			fmt.Println(formatPair(data.CG_1, positions["cg1"].ToString(), positions["cg2"].Reverse().ToString()))

			// pack all
			for k, v := range positions {
				positions[k] = v[:len(v)-1]
			}

			packed, err := model.PackBoardV1(positions)
			if err != nil {
				fmt.Println("error packing:", err)
				return
			}

			for _, v := range packed {
				fmt.Printf("%x :: ", v)
				for _, b := range v {
					fmt.Printf("%08b ", b)
				}
				fmt.Println()
			}

		}(names, ctxTimeout)
		<-timer
	}
}

func formatPair(dl data.Line, forward string, reverse string) string {
	var sb strings.Builder

	sb.WriteString(dl[0].Name)
	sb.WriteRune(' ')
	sb.WriteString(forward)
	sb.WriteString(" >>>\n")
	for i := 0; i < len(dl[0].Name)-3; i++ {
		sb.WriteRune(' ')
	}
	sb.WriteString("<<< ")
	sb.WriteString(reverse)
	sb.WriteRune(' ')
	sb.WriteString(dl[len(dl)-1].Name)

	return sb.String()
}
