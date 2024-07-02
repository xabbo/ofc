package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
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

// Hair -> hat map
var hairToHatMap = map[int]int{
	// m
	120: 1001,
	130: 1010,
	140: 1004,
	150: 1003,
	160: 1004,
	175: 1006,
	176: 1007,
	177: 1008,
	178: 1009,
	800: 1012,
	801: 1011,
	802: 1013,
	// f
	525: 1002,
	535: 1003,
	565: 1004,
	570: 1005,
	580: 1007,
	585: 1006,
	590: 1008,
	595: 1009,
	810: 1012,
	811: 1013,
}

var opts struct {
	quiet bool
}

var verbose = io.Discard

func main() {
	flag.Usage = func() {
		name := os.Args[0]
		if filepath.IsAbs(name) {
			name = filepath.Base(os.Args[0])
		}
		if runtime.GOOS == "windows" {
			name = strings.TrimSuffix(name, filepath.Ext(name))
		}
		fmt.Fprintf(os.Stderr, "Usage: %s [-q] [figureString]\n", name)
	}
	flag.BoolVar(&opts.quiet, "q", false, "Quiet output")
	flag.Parse()

	if !opts.quiet {
		verbose = os.Stderr
	}
	if err := run(flag.Args()); err != nil {
		if errors.Is(err, errUsage) {
			flag.Usage()
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

	fmt.Fprintf(verbose, "Loading origins figure data... ")
	fd, err := loadOriginsFigureData()
	if err != nil {
		return
	}
	fmt.Fprintln(verbose, "ok")

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
		colorIndex, _ := strconv.Atoi(oldFigure[i+3 : i+5])

		set := setIds[setId]
		nxPart := nx.FigurePart{
			Type: nx.FigurePartType(set.Type),
			Id:   setId,
		}

		partColor := strings.ToLower(set.Colors[colorIndex-1])
		colorId := colorMap[set.Type][partColor]
		nxPart.Colors = append(nxPart.Colors, colorId)

		figure.Parts = append(figure.Parts, nxPart)

		if nxPart.Type == nx.Hair {
			if hatId, ok := hairToHatMap[nxPart.Id]; ok {
				figure.Parts = append(figure.Parts, nx.FigurePart{Type: nx.Hat, Id: hatId, Colors: []int{colorId}})
			}
		}
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
	fmt.Fprintf(verbose, "Loading modern figure data... ")
	gdm := gd.NewGamedataManager("www.habbo.com")
	err = gdm.Load(gd.GamedataFigure)
	if err != nil {
		return
	}
	fmt.Fprintln(verbose, "ok")

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
	f, err := os.OpenFile(originsFigureDataFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return
	}

	var b []byte
	if fi.Size() == 0 {
		var res *http.Response
		res, err = http.Get(originsFigureDataUrl)
		if err != nil {
			return
		}
		if res.StatusCode != 200 {
			err = errors.New(res.Status)
			return
		}
		buf := bytes.NewBuffer(make([]byte, 0, res.ContentLength))
		_, err = io.Copy(buf, res.Body)
		if err == nil {
			b = buf.Bytes()
		}
		fixOriginsFigureData(b)
		_, err = f.Write(b)
		if err != nil {
			return
		}
	} else {
		b = make([]byte, fi.Size())
		_, err = io.ReadFull(f, b)
		if err != nil {
			return
		}
	}
	err = json.Unmarshal(b, &fd)
	if err != nil {
		err = fmt.Errorf("weirdness in figure data!!! %w", err)
	}
	return
}

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
