package dotosu

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	EARLY_VERSION_TIMING_OFFSET = 24
	CONTROL_POINT_LENIENCY      = 5.0
	MAX_MANIA_KEY_COUNT         = 18
	LATEST_VERSION              = 14
)

type section int

const (
	secNone section = iota
	secGeneral
	secEditor
	secMetadata
	secDifficulty
	secEvents
	secTimingPoints
	secHitObjects
)

// ---------- Beatmap model (unchanged parts condensed) ----------

type Beatmap struct {
	FormatVersion int
	General       General
	Editor        Editor
	Metadata      Metadata
	Difficulty    Difficulty

	Breaks          []BreakPeriod
	TimingPoints    []TimingPoint
	HitObjects      []HitObject
	UnhandledEvents []string

	Bookmarks    []int
	BeatDivisor  int
	GridSize     int
	TimelineZoom float64
}

type General struct {
	AudioFilename            string
	AudioLeadIn              int
	PreviewTime              int
	SampleSet                string
	SampleVolume             int
	StackLeniency            float64
	Mode                     int
	LetterboxInBreaks        bool
	SpecialStyle             bool
	WidescreenStoryboard     bool
	EpilepsyWarning          bool
	SamplesMatchPlaybackRate bool
	Countdown                int
	CountdownOffset          int
}

type Editor struct{ DistanceSpacing float64 }

type Metadata struct {
	Title, TitleUnicode            string
	Artist, ArtistUnicode          string
	Creator, Version, Source, Tags string
	BeatmapID, BeatmapSetID        int
	BackgroundFile, VideoFile      string
}

type Difficulty struct {
	HPDrainRate, CircleSize, OverallDifficulty, ApproachRate float64
	SliderMultiplier, SliderTickRate                         float64
}

type BreakPeriod struct{ Start, End float64 }

type TimingPoint struct {
	Time                     int
	BeatLength               float64
	TimeSignature            int
	SampleSet                string
	CustomSampleBank         int
	SampleVolume             int
	TimingChange             bool
	Kiai                     bool
	OmitFirstBarSignature    bool
	SliderVelocityMultiplier float64
	ScrollSpeed              float64
}

// ---------- HitObject enums & typed variants (no raw strings) ----------

type ObjectKind uint8

const (
	KindCircle ObjectKind = iota
	KindSlider
	KindSpinner
	KindHold
)

type HitSoundFlags uint8

const (
	HitSoundNormal  HitSoundFlags = 1 << iota // 1
	HitSoundWhistle                           // 2
	HitSoundFinish                            // 4
	HitSoundClap                              // 8
)

type SampleSet uint8

const (
	SampleNone SampleSet = iota
	SampleNormal
	SampleSoft
	SampleDrum
)

type HitObjectTypeFlags int

const (
	TypeCircle     HitObjectTypeFlags = 1 << iota // 1
	TypeSlider                                    // 2
	TypeNewCombo                                  // 4
	TypeSpinner                                   // 8
	TypeComboSkip1                                // 16
	TypeComboSkip2                                // 32
	TypeComboSkip3                                // 64
	TypeHold       HitObjectTypeFlags = 1 << 7    // 128
)

type Vec2 struct{ X, Y int }

type HitSampleSpec struct {
	NormalSet   SampleSet
	AdditionSet SampleSet
	Index       int // custom sample bank
	Volume      int
	Filename    string // may be empty; kept as a small literal path
}

type EdgeAdd struct {
	NormalSet   SampleSet
	AdditionSet SampleSet
}

type SliderPathType uint8

const (
	PathBezier SliderPathType = iota
	PathLinear
	PathCatmull
	PathPerfect
)

type SliderSegment struct {
	// Points for this segment INCLUDING its starting point.
	// For the FIRST segment, the first point == slider head (x,y).
	Points []Vec2
}

type SliderPath struct {
	Type     SliderPathType
	Segments []SliderSegment // For Bezier, split when a control point repeats (red anchor).
}

type HitObject interface {
	Kind() ObjectKind
	StartTime() int
	NewCombo() bool
	Flags() HitObjectTypeFlags
	Pos() Vec2
	HitSound() HitSoundFlags
	Sample() HitSampleSpec
}

type BaseHO struct {
	PosXY    Vec2
	Time     int
	Type     HitObjectTypeFlags
	Sound    HitSoundFlags
	SampleHS HitSampleSpec
}

func (b BaseHO) StartTime() int            { return b.Time }
func (b BaseHO) NewCombo() bool            { return (b.Type & TypeNewCombo) != 0 }
func (b BaseHO) Flags() HitObjectTypeFlags { return b.Type }
func (b BaseHO) Pos() Vec2                 { return b.PosXY }
func (b BaseHO) HitSound() HitSoundFlags   { return b.Sound }
func (b BaseHO) Sample() HitSampleSpec     { return b.SampleHS }

type Circle struct{ BaseHO }

func (Circle) Kind() ObjectKind { return KindCircle }

type Slider struct {
	BaseHO
	Path          SliderPath
	Slides        int
	Length        float64
	EdgeSounds    []HitSoundFlags // len == Slides+1 (head, repeats..., tail)
	EdgeAdditions []EdgeAdd       // len == Slides+1
}

func (Slider) Kind() ObjectKind { return KindSlider }

type Spinner struct {
	BaseHO
	EndTime int
}

func (Spinner) Kind() ObjectKind { return KindSpinner }

type Hold struct {
	BaseHO
	EndTime int
}

func (Hold) Kind() ObjectKind { return KindHold }

// ---------- Public API ----------

func DecodeFile(path string) (*Beatmap, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Decode(f)
}

func Decode(r io.Reader) (*Beatmap, error) {
	sc := bufio.NewScanner(r)
	const maxLine = 1024 * 1024
	buf := make([]byte, 64*1024)
	sc.Buffer(buf, maxLine)

	// header
	var header string
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		header = line
		break
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if !strings.HasPrefix(strings.ToLower(header), "osu file format v") {
		return nil, fmt.Errorf("invalid .osu header: %q", header)
	}
	versionStr := strings.TrimSpace(strings.TrimPrefix(header, "osu file format v"))
	formatVersion, err := strconv.Atoi(versionStr)
	if err != nil {
		return nil, fmt.Errorf("invalid .osu version in header: %q: %w", header, err)
	}

	b := &Beatmap{
		FormatVersion: formatVersion,
		General: General{
			WidescreenStoryboard: false,
			SampleSet:            "normal",
			SampleVolume:         100,
		},
		BeatDivisor: 4, GridSize: 4,
	}

	offset := 0
	if formatVersion < 5 {
		offset = EARLY_VERSION_TIMING_OFFSET
	}

	sec := secNone
	seenAR := false

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			switch strings.ToLower(line) {
			case "[general]":
				sec = secGeneral
			case "[editor]":
				sec = secEditor
			case "[metadata]":
				sec = secMetadata
			case "[difficulty]":
				sec = secDifficulty
			case "[events]":
				sec = secEvents
			case "[timingpoints]":
				sec = secTimingPoints
			case "[hitobjects]":
				sec = secHitObjects
			default:
				sec = secNone
			}
			continue
		}

		switch sec {
		case secGeneral:
			k, v := splitKeyVal(line)
			switch strings.ToLower(k) {
			case "audiofilename":
				b.General.AudioFilename = standardisePath(v)
			case "audioleadin":
				b.General.AudioLeadIn = parseInt(v, 0)
			case "previewtime":
				t := parseInt(v, -1)
				if t != -1 {
					t += offset
				}
				b.General.PreviewTime = t
			case "sampleset":
				b.General.SampleSet = strings.ToLower(v)
			case "samplevolume":
				b.General.SampleVolume = parseInt(v, 100)
			case "stackleniency":
				b.General.StackLeniency = parseFloat(v, 0)
			case "mode":
				b.General.Mode = parseInt(v, 0)
			case "letterboxinbreaks":
				b.General.LetterboxInBreaks = parseBoolInt(v)
			case "specialstyle":
				b.General.SpecialStyle = parseBoolInt(v)
			case "widescreenstoryboard":
				b.General.WidescreenStoryboard = parseBoolInt(v)
			case "epilepsywarning":
				b.General.EpilepsyWarning = parseBoolInt(v)
			case "samplesmatchplaybackrate":
				b.General.SamplesMatchPlaybackRate = parseBoolInt(v)
			case "countdown":
				b.General.Countdown = parseInt(v, 0)
			case "countdownoffset":
				b.General.CountdownOffset = parseInt(v, 0)
			}

		case secEditor:
			k, v := splitKeyVal(line)
			switch strings.ToLower(k) {
			case "bookmarks":
				if strings.TrimSpace(v) != "" {
					for _, p := range strings.Split(v, ",") {
						if p = strings.TrimSpace(p); p != "" {
							b.Bookmarks = append(b.Bookmarks, parseInt(p, 0))
						}
					}
				}
			case "distancespacing":
				b.Editor.DistanceSpacing = parseFloat(v, 0)
			case "beatdivisor":
				b.BeatDivisor = clampInt(parseInt(v, 4), 1, 16)
			case "gridsize":
				b.GridSize = parseInt(v, 4)
			case "timelinezoom":
				b.TimelineZoom = math.Max(0, parseFloat(v, 0))
			}

		case secMetadata:
			k, v := splitKeyVal(line)
			switch strings.ToLower(k) {
			case "title":
				b.Metadata.Title = v
			case "titleunicode":
				b.Metadata.TitleUnicode = v
			case "artist":
				b.Metadata.Artist = v
			case "artistunicode":
				b.Metadata.ArtistUnicode = v
			case "creator":
				b.Metadata.Creator = v
			case "version":
				b.Metadata.Version = v
			case "source":
				b.Metadata.Source = v
			case "tags":
				b.Metadata.Tags = v
			case "beatmapid":
				b.Metadata.BeatmapID = parseInt(v, 0)
			case "beatmapsetid":
				b.Metadata.BeatmapSetID = parseInt(v, 0)
			}

		case secDifficulty:
			k, v := splitKeyVal(line)
			switch strings.ToLower(k) {
			case "hpdrainrate":
				b.Difficulty.HPDrainRate = parseFloat(v, 0)
			case "circlesize":
				b.Difficulty.CircleSize = parseFloat(v, 0)
			case "overalldifficulty":
				b.Difficulty.OverallDifficulty = parseFloat(v, 0)
				if !seenAR {
					b.Difficulty.ApproachRate = b.Difficulty.OverallDifficulty
				}
			case "approachrate":
				b.Difficulty.ApproachRate = parseFloat(v, 0)
				seenAR = true
			case "slidermultiplier":
				b.Difficulty.SliderMultiplier = parseFloat(v, 1)
			case "slidertickrate":
				b.Difficulty.SliderTickRate = parseFloat(v, 1)
			}

		case secEvents:
			parts := splitCSV(line)
			if len(parts) == 0 {
				continue
			}
			switch strings.ToLower(parts[0]) {
			case "0", "background":
				if len(parts) >= 3 {
					b.Metadata.BackgroundFile = cleanFilename(parts[2])
				} else {
					b.UnhandledEvents = append(b.UnhandledEvents, line)
				}
			case "1", "video":
				if len(parts) >= 3 {
					fn := cleanFilename(parts[2])
					ext := strings.ToLower(filepath.Ext(fn))
					switch ext {
					case ".avi", ".flv", ".mp4", ".mkv", ".mov", ".wmv", ".mpg", ".mpeg", ".ogv", ".webm":
						b.Metadata.VideoFile = fn
					default:
						b.Metadata.BackgroundFile = fn
					}
				} else {
					b.UnhandledEvents = append(b.UnhandledEvents, line)
				}
			case "2", "break":
				if len(parts) >= 3 {
					start := parseFloat(parts[1], 0) + float64(offset)
					end := parseFloat(parts[2], start) + float64(offset)
					if end < start {
						end = start
					}
					b.Breaks = append(b.Breaks, BreakPeriod{Start: start, End: end})
				} else {
					b.UnhandledEvents = append(b.UnhandledEvents, line)
				}
			default:
				b.UnhandledEvents = append(b.UnhandledEvents, line)
			}

		case secTimingPoints:
			parts := splitCSV(line)
			if len(parts) < 2 {
				continue
			}
			t := parseInt(parts[0], 0) + offset
			beatLen := parseFloatAllowNaN(parts[1])
			meter := 4
			if len(parts) >= 3 {
				meter = parseInt(parts[2], 4)
				if meter == 0 {
					meter = 4
				}
			}
			sampleSet := "normal"
			if len(parts) >= 4 {
				sampleSet = normaliseSampleSet(parseInt(parts[3], 0))
			}
			custom := 0
			if len(parts) >= 5 {
				custom = parseInt(parts[4], 0)
			}
			sampleVol := 100
			if len(parts) >= 6 {
				sampleVol = parseInt(parts[5], 100)
			}
			timingChange := true
			if len(parts) >= 7 {
				timingChange = strings.TrimSpace(parts[6]) == "1"
			}
			kiai, omitFirstBar := false, false
			if len(parts) >= 8 {
				e := parseInt(parts[7], 0)
				if e&1 != 0 {
					kiai = true
				}
				if e&8 != 0 {
					omitFirstBar = true
				}
			}
			sv := 1.0
			if !math.IsNaN(beatLen) && beatLen < 0 {
				sv = 100.0 / -beatLen
			}
			scroll := 1.0
			if !math.IsNaN(beatLen) && beatLen < 0 {
				scroll = 100.0 / -beatLen
			}
			if strings.ToLower(sampleSet) == "none" {
				sampleSet = "normal"
			}
			b.TimingPoints = append(b.TimingPoints, TimingPoint{
				Time: t, BeatLength: beatLen, TimeSignature: meter, SampleSet: sampleSet,
				CustomSampleBank: custom, SampleVolume: sampleVol, TimingChange: timingChange,
				Kiai: kiai, OmitFirstBarSignature: omitFirstBar, SliderVelocityMultiplier: sv, ScrollSpeed: scroll,
			})

		case secHitObjects:
			parts := splitCSVPreserveTail(line, 11) // keep trailing parameters grouped
			if len(parts) < 5 {
				continue
			}
			x := parseInt(parts[0], 0)
			y := parseInt(parts[1], 0)
			t := parseInt(parts[2], 0) + offset
			flags := HitObjectTypeFlags(parseInt(parts[3], 0))
			hs := HitSoundFlags(parseInt(parts[4], 0))

			base := BaseHO{PosXY: Vec2{X: x, Y: y}, Time: t, Type: flags, Sound: hs}

			switch {
			case (flags & TypeHold) != 0:
				// mania hold: "endTime:sample"
				if len(parts) >= 6 {
					end, samp := parseEndTimeAndSample(parts[5])
					base.SampleHS = samp
					b.HitObjects = append(b.HitObjects, Hold{BaseHO: base, EndTime: end + offset})
				} else {
					b.HitObjects = append(b.HitObjects, Hold{BaseHO: base})
				}

			case (flags & TypeSpinner) != 0:
				end := 0
				if len(parts) >= 6 && strings.TrimSpace(parts[5]) != "" {
					end = parseInt(parts[5], 0) + offset
				}
				if len(parts) >= 7 {
					base.SampleHS = parseHitSample(parts[6])
				}
				b.HitObjects = append(b.HitObjects, Spinner{BaseHO: base, EndTime: end})

			case (flags & TypeSlider) != 0:
				// params: path, slides, length, edgeSounds, edgeAdditions, (hitSample optional as last column)
				var pathSpec string
				if len(parts) >= 6 {
					pathSpec = parts[5]
				}
				slides := 1
				if len(parts) >= 7 && strings.TrimSpace(parts[6]) != "" {
					slides = parseInt(parts[6], 1)
				}
				length := 0.0
				if len(parts) >= 8 && strings.TrimSpace(parts[7]) != "" {
					length = parseFloat(parts[7], 0)
				}

				var edgeSounds []HitSoundFlags
				if len(parts) >= 9 && strings.TrimSpace(parts[8]) != "" {
					for _, n := range strings.Split(parts[8], "|") {
						edgeSounds = append(edgeSounds, HitSoundFlags(parseInt(n, 0)))
					}
				}
				var edgeAdds []EdgeAdd
				if len(parts) >= 10 && strings.TrimSpace(parts[9]) != "" {
					for _, p := range strings.Split(parts[9], "|") {
						ns, as := parseEdgeAddPair(p)
						edgeAdds = append(edgeAdds, EdgeAdd{NormalSet: ns, AdditionSet: as})
					}
				}
				// trailing hitSample (may be in parts[6+] depending on presence)
				if len(parts) >= 11 {
					base.SampleHS = parseHitSample(parts[10])
				}

				path := parseSliderPath(base.PosXY, pathSpec) // fully parsed (no strings)
				b.HitObjects = append(b.HitObjects, Slider{
					BaseHO:        base,
					Path:          path,
					Slides:        slides,
					Length:        length,
					EdgeSounds:    edgeSounds,
					EdgeAdditions: edgeAdds,
				})

			default:
				// Circle
				if len(parts) >= 6 {
					base.SampleHS = parseHitSample(parts[5])
				}
				b.HitObjects = append(b.HitObjects, Circle{BaseHO: base})
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	applyDifficultyRestrictions(&b.Difficulty, b.General.Mode)
	return b, nil
}

// ---------- parsing helpers ----------

func splitKeyVal(line string) (key, val string) {
	i := strings.Index(line, ":")
	if i < 0 {
		return strings.TrimSpace(line), ""
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:])
}

func parseInt(s string, def int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}
func parseFloat(s string, def float64) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return def
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return def
	}
	return v
}
func parseFloatAllowNaN(s string) float64 {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, "nan") {
		return math.NaN()
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return math.NaN()
	}
	return v
}
func parseBoolInt(s string) bool { return strings.TrimSpace(s) == "1" }

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func standardisePath(p string) string {
	p = strings.Trim(p, "\"")
	return strings.ReplaceAll(p, "\\", "/")
}
func cleanFilename(s string) string {
	s = strings.Trim(s, "\"")
	return strings.ReplaceAll(s, "\\", "/")
}

func splitCSV(line string) []string {
	var out []string
	var cur strings.Builder
	inQ := false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch c {
		case '"':
			inQ = !inQ
		case ',':
			if inQ {
				cur.WriteByte(c)
			} else {
				out = append(out, strings.TrimSpace(cur.String()))
				cur.Reset()
			}
		default:
			cur.WriteByte(c)
		}
	}
	out = append(out, strings.TrimSpace(cur.String()))
	return out
}
func splitCSVPreserveTail(line string, n int) []string {
	parts := splitCSV(line)
	if len(parts) <= n {
		return parts
	}
	head := parts[:n-1]
	tail := strings.Join(parts[n-1:], ",")
	return append(head, tail)
}

func normaliseSampleSet(id int) string {
	switch id {
	case 1:
		return "normal"
	case 2:
		return "soft"
	case 3:
		return "drum"
	default:
		return "none"
	}
}

func applyDifficultyRestrictions(d *Difficulty, mode int) {
	d.HPDrainRate = clampFloat(d.HPDrainRate, 0, 10)
	d.OverallDifficulty = clampFloat(d.OverallDifficulty, 0, 10)
	d.ApproachRate = clampFloat(d.ApproachRate, 0, 10)
	if mode == 3 {
		d.CircleSize = clampFloat(d.CircleSize, 1, MAX_MANIA_KEY_COUNT)
	} else {
		d.CircleSize = clampFloat(d.CircleSize, 0, 10)
	}
	d.SliderMultiplier = clampFloat(d.SliderMultiplier, 0.4, 3.6)
	d.SliderTickRate = clampFloat(d.SliderTickRate, 0.5, 8.0)
}

// --- object-param parsing (typed, no raw strings) ---

func parseHitSample(s string) HitSampleSpec {
	// normalSet:additionSet:customIndex:volume:filename
	parts := strings.Split(s, ":")
	get := func(i int) string {
		if i < len(parts) {
			return parts[i]
		}
		return ""
	}
	ss := HitSampleSpec{}
	ss.NormalSet = toSampleSet(parseInt(get(0), 0))
	ss.AdditionSet = toSampleSet(parseInt(get(1), 0))
	ss.Index = parseInt(get(2), 0)
	ss.Volume = parseInt(get(3), 0)
	ss.Filename = strings.Trim(get(4), " ")
	ss.Filename = strings.Trim(ss.Filename, "\"")
	return ss
}

func toSampleSet(id int) SampleSet {
	switch id {
	case 1:
		return SampleNormal
	case 2:
		return SampleSoft
	case 3:
		return SampleDrum
	default:
		return SampleNone
	}
}

func parseEdgeAddPair(s string) (SampleSet, SampleSet) {
	// "x:y"
	p := strings.Split(s, ":")
	var a, b int
	if len(p) >= 1 {
		a = parseInt(p[0], 0)
	}
	if len(p) >= 2 {
		b = parseInt(p[1], 0)
	}
	return toSampleSet(a), toSampleSet(b)
}

func parseEndTimeAndSample(s string) (int, HitSampleSpec) {
	// "endTime:hitSampleSpec"
	colon := strings.Index(s, ":")
	if colon < 0 {
		return parseInt(s, 0), HitSampleSpec{}
	}
	end := parseInt(s[:colon], 0)
	return end, parseHitSample(s[colon+1:])
}

// parseSliderPath converts "B|x:y|x:y|..." into a fully-typed SliderPath.
// The slider head (base) is the FIRST point; the string supplies the rest.
// Bézier: split into segments when a control point repeats (red anchor). :contentReference[oaicite:1]{index=1}
func parseSliderPath(head Vec2, spec string) SliderPath {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return SliderPath{Type: PathBezier, Segments: []SliderSegment{{Points: []Vec2{head}}}}
	}

	// Type is first token up to first '|'
	tokEnd := strings.Index(spec, "|")
	var typeStr string
	var rest string
	if tokEnd == -1 {
		typeStr, rest = spec, ""
	} else {
		typeStr, rest = spec[:tokEnd], spec[tokEnd+1:]
	}
	var pType SliderPathType
	switch strings.ToUpper(strings.TrimSpace(typeStr)) {
	case "L":
		pType = PathLinear
	case "C":
		pType = PathCatmull
	case "P":
		pType = PathPerfect
	default:
		pType = PathBezier
	}

	// Parse control points
	var cps []Vec2
	if strings.TrimSpace(rest) != "" {
		for _, t := range strings.Split(rest, "|") {
			xy := strings.Split(strings.TrimSpace(t), ":")
			if len(xy) != 2 {
				continue
			}
			cps = append(cps, Vec2{X: parseInt(xy[0], head.X), Y: parseInt(xy[1], head.Y)})
		}
	}

	switch pType {
	case PathPerfect:
		// Perfect circle expects exactly head + 2 points; otherwise fall back to Bezier (stable behaviour).
		if len(cps) != 2 {
			return buildBezierWithSegments(head, cps)
		}
		return SliderPath{Type: PathPerfect, Segments: []SliderSegment{{Points: append([]Vec2{head}, cps...)}}}
	case PathLinear:
		return SliderPath{Type: PathLinear, Segments: []SliderSegment{{Points: append([]Vec2{head}, cps...)}}}
	case PathCatmull:
		return SliderPath{Type: PathCatmull, Segments: []SliderSegment{{Points: append([]Vec2{head}, cps...)}}}
	default: // Bezier (with segment splitting by repeated points)
		return buildBezierWithSegments(head, cps)
	}
}

func buildBezierWithSegments(head Vec2, cps []Vec2) SliderPath {
	pts := append([]Vec2{head}, cps...)
	var segs []SliderSegment
	cur := []Vec2{pts[0]}
	for i := 1; i < len(pts); i++ {
		p := pts[i]
		prev := cur[len(cur)-1]
		if p.X == prev.X && p.Y == prev.Y {
			// segment boundary (red anchor)
			if len(cur) >= 2 {
				segs = append(segs, SliderSegment{Points: cur})
			}
			cur = []Vec2{p}
			continue
		}
		cur = append(cur, p)
	}
	if len(cur) >= 2 {
		segs = append(segs, SliderSegment{Points: cur})
	}
	if len(segs) == 0 {
		// degenerate; keep at least a 2-point segment (head duplicated)
		segs = []SliderSegment{{Points: []Vec2{head, head}}}
	}
	return SliderPath{Type: PathBezier, Segments: segs}
}

// ---------- optional validation ----------

func (b *Beatmap) Validate() error {
	if b.Metadata.Title == "" && b.Metadata.TitleUnicode == "" {
		return errors.New("missing title")
	}
	if b.Metadata.Artist == "" && b.Metadata.ArtistUnicode == "" {
		return errors.New("missing artist")
	}
	if b.General.AudioFilename == "" {
		return errors.New("missing AudioFilename in [General]")
	}
	return nil
}
