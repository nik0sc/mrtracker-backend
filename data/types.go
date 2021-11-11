package data

import (
	"fmt"
	"strconv"
	"strings"
)

type Station struct {
	Code     string // alphanumeric station code
	Code3    string // three letter alphabetical station code
	Platform string // platform letter
	Name     string
}

func (s Station) CodeNum() int {
	if s.Code[2:] == "" {
		return 0
	}

	n, err := strconv.Atoi(s.Code[2:])
	if err != nil {
		panic(err)
	}
	return n
}

func (s Station) PlatformID() string {
	return fmt.Sprintf("%s_%s", s.Code3, s.Platform)
}

type Line []Station

func (l Line) Len() int {
	return len(l)
}

func (l Line) Less(i, j int) bool {
	return l[i].CodeNum() < l[j].CodeNum()
}

func (l Line) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

func (l Line) Repr() string {
	var sb strings.Builder
	sb.WriteString("Line{\n")
	for _, station := range l {
		sb.WriteString("\tStation{\n")
		sb.WriteString(fmt.Sprintf("\t\tCode:     %q,\n", station.Code))
		sb.WriteString(fmt.Sprintf("\t\tCode3:    %q,\n", station.Code3))
		sb.WriteString(fmt.Sprintf("\t\tPlatform: %q,\n", station.Platform))
		sb.WriteString(fmt.Sprintf("\t\tName:     %q,\n", station.Name))
		sb.WriteString("\t},\n")
	}
	sb.WriteString("}\n")

	return sb.String()
}
