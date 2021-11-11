package data

func GetNames() []string {
	// TODO: This can probably be generated once
	nameSet := make(map[string]struct{})
	lineSource := []Line{EW_1, NS_1, CG_1}

	for _, l := range lineSource {
		for i := range l {
			nameSet[l[i].Name] = struct{}{}
		}
	}

	var names []string
	for k := range nameSet {
		names = append(names, k)
	}

	return names
}

type LineNameDataPair struct {
	Name string
	Line Line
}

func GetLines() []LineNameDataPair {
	return []LineNameDataPair{
		{"ns1", NS_1},
		{"ns2", NS_2},
		{"ew1", EW_1},
		{"ew2", EW_2},
		{"cg1", CG_1},
		{"cg2", CG_2},
	}
}
