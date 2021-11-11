package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"go.lepak.sg/mrtracker-backend/data"
	"go.lepak.sg/mrtracker-backend/smrt"
)

/*
Generate line data

Source data: github.com/cheeaun/sgraildata licensed under the ISC License (presumed in package.json)
Notice of source data follows:

Copyright 2021 Lim Chee Aun

Permission to use, copy, modify, and/or distribute this software for any purpose with or without fee is hereby granted,
provided that the above copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT,
INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF
THIS SOFTWARE.
*/

const (
	path = "https://raw.githubusercontent.com/cheeaun/sgraildata/master/data/raw/wikipedia-mrt.json"
)

func main() {
	var rawData []struct {
		Codes []string
		Name  string
	}

	resp, err := http.Get(path)
	if err != nil {
		panic(err)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	err = resp.Body.Close()
	if err != nil {
		panic(err)
	}

	err = json.Unmarshal(buf, &rawData)
	if err != nil {
		panic(err)
	}

	lines := map[string]data.Line{
		"NS": {},
		"EW": {},
		"CG": {},
	}

	for _, el := range rawData {
		for _, code := range el.Codes {
			if _, ok := lines[code[:2]]; ok {
				lines[code[:2]] = append(lines[code[:2]], data.Station{Code: code, Name: el.Name})
			}
		}
	}

	var linenames []string
	for k := range lines {
		linenames = append(linenames, k)
	}

	for _, k := range linenames {
		sort.Sort(lines[k])

		lines[k+"_1"] = lines[k]

		reverse := make(data.Line, len(lines[k]))
		copy(reverse, lines[k])
		sort.Sort(sort.Reverse(reverse))

		lines[k+"_2"] = reverse

		delete(lines, k)
	}

	for k, line := range lines {
		end := line[len(line)-1].Name
	outerLoop:
		for i, station := range line {
			res, err := smrt.GetOne(context.Background(), 5, station.Name)
			if err != nil {
				log.Printf("error [%s][%s]: %v", k, station.Name, err)
				continue
			}
			for _, resPlatform := range res {
				if resPlatform.NextTrainDestination == end {
					// copy this platform data
					substrings := strings.Split(resPlatform.PlatformID, "_")
					line[i].Code3 = substrings[0]
					line[i].Platform = substrings[1]
					continue outerLoop
				}
			}
			log.Printf("no hit for [%s][%s]", k, station.Name)
		}
	}

	f, err := os.Create("lines.go")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	_, err = f.WriteString("package data\n\n// Generated by data/gen/main.go, do not edit\n\n")
	if err != nil {
		panic(err)
	}

	linenames = nil
	for k := range lines {
		linenames = append(linenames, k)
	}
	sort.Strings(linenames)

	for _, k := range linenames {
		_, err = f.WriteString(fmt.Sprintf("var %s = %s\n", k, lines[k].Repr()))
		if err != nil {
			panic(err)
		}
	}
}