package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"xabbo.b7c.io/nx"
	gd "xabbo.b7c.io/nx/gamedata"
)

var errUsage = errors.New("usage")

const (
	colorMapFile          = "colormap.json"
	originsFigureDataFile = "origins-figuredata.json"
	originsFigureDataUrl  = "http://origins-gamedata.habbo.com/figuredata/1"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		if errors.Is(err, errUsage) {
			name := os.Args[0]
			if filepath.IsAbs(name) {
				name = filepath.Base(os.Args[0])
			}
			if runtime.GOOS == "windows" {
				name = strings.TrimSuffix(name, filepath.Ext(name))
			}
			fmt.Printf("usage: %s [figureString]\n", name)
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		}
		os.Exit(1)
	}
}

func run(args []string) (err error) {
	if len(args) == 0 {
		err = errUsage
		return
	}

	oldFigure := args[0]
	if len(oldFigure) != 25 {
		err = errors.New("invalid figure string, must be 25 characters in length")
		return
	}

	for _, c := range oldFigure {
		if c < '0' || c > '9' {
			err = errors.New("invalid figure string, must consist only of numbers")
			return
		}
	}

	colorMap, err := loadColorMap()
	if err != nil {
		return
	}

	fmt.Fprintf(os.Stderr, "Loading origins figure data... ")
	fd, err := loadOriginsFigureData()
	if err != nil {
		return
	}
	fmt.Fprintln(os.Stderr, "ok")

	// map part set id -> part set
	setIds := map[int]FigurePartSet{}
	for _, genderSet := range []map[string]FigurePartSets{fd.M, fd.F} {
		for setType, items := range genderSet {
			for _, partSet := range items {
				partSet.Type = setType
				setIds[partSet.Id] = partSet
			}
		}
	}

	var figure nx.Figure

	for i := 0; i < 25; i += 5 {
		setId, _ := strconv.Atoi(oldFigure[i : i+3])
		colorId, _ := strconv.Atoi(oldFigure[i+3 : i+5])

		set := setIds[setId]
		nxPart := nx.FigurePart{
			Type: nx.FigurePartType(set.Type),
			Id:   setId,
		}

		colorId = colorMap[set.Type][strings.ToLower(set.Colors[colorId-1])]
		nxPart.Colors = append(nxPart.Colors, colorId)

		figure.Parts = append(figure.Parts, nxPart)
	}

	fmt.Println(figure.String())
	return
}

func loadColorMap() (cm ColorMap, err error) {
	f, err := os.Open("colormap.json")
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return makeColorMap()
		}
		return
	}
	defer f.Close()

	err = json.NewDecoder(f).Decode(&cm)
	return
}

func makeColorMap() (cm ColorMap, err error) {
	fmt.Fprintf(os.Stderr, "Loading modern figure data... ")
	gdm := gd.NewGamedataManager("www.habbo.com")
	err = gdm.Load(gd.GamedataFigure)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Fprintln(os.Stderr, "ok")

	cm = map[string]map[string]int{}
	for partType, paletteId := range gdm.Figure.SetPalettes {
		cm[string(partType)] = map[string]int{}
		palette := gdm.Figure.Palettes[paletteId]
		for _, color := range palette {
			cm[string(partType)][strings.ToLower(color.Value)] = color.Id
		}
	}

	f, err := os.OpenFile(colorMapFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	err = json.NewEncoder(f).Encode(&cm)
	return
}

func loadOriginsFigureData() (fd FigureData, err error) {
	data, err := loadOrFetch(originsFigureDataFile, originsFigureDataUrl)
	if err == nil {
		fixOriginsFigureData(data)
		err = json.Unmarshal(data, &fd)
	}
	return
}
