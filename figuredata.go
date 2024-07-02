package main

type FigureData struct {
	M, F map[string]FigurePartSets
}

type FigurePartSets []FigurePartSet

type FigurePartSet struct {
	Type   string            `json:"-"`
	Id     int               `json:"s"`
	Parts  map[string]string `json:"p"`
	Colors []string          `json:"c"`
}

// Reverse color map from Figure Part Type -> Color (lowercase hex) -> Modern Color ID
type ColorMap = map[string]map[string]int

// Fixes the old figure data to valid JSON
func fixOriginsFigureData(b []byte) {
	sp := -1
	stack := [8]struct {
		i      int
		object bool
	}{}

	for i := range b {
		// assuming these characters don't appear inside any strings
		switch b[i] {
		case '[':
			sp++
			stack[sp].i = i
			stack[sp].object = false
		case ':':
			stack[sp].object = true
		case ']':
			if stack[sp].object {
				b[stack[sp].i] = '{'
				b[i] = '}'
			}
			sp--
		}
	}
}
