package main

import (
	"log"
	"os"
	"sort"

	"github.com/asticode/go-astisub"
	"github.com/will-x86/subtitle-poisoning/libgosubs/ass"
)

func main() {
	s1, err := astisub.OpenFile("./subtitles/youtube-orig.srt")
	if err != nil {
		panic(err)
	}
	// Create the output file
	f, err := os.Create("./subtitles/parsed.ass")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// Initialize Metadata if nil
	if s1.Metadata == nil {
		s1.Metadata = &astisub.Metadata{
			SSAScriptType: "v4.00+",
		}
	} else {
		s1.Metadata.SSAScriptType = "v4.00+"
	}

	// Write using WriteSSA
	err = s1.WriteToSSA(f)
	if err != nil {
		panic(err)
	}
	a, err := ass.ParseAss("./subtitles/parsed.ass")
	if err != nil {
		panic(err)
	}

	// Add default style
	defaultStyle := ass.Style{
		Name:            "real",
		Fontname:        "Arial",
		Fontsize:        16,
		PrimaryColour:   "&H00FFFFFF",
		SecondaryColour: "&H00FFFFFF",
		OutlineColour:   "&H00000000",
		Backcolour:      "&H00000000",
		Bold:            0,
		Italic:          0,
		Underline:       0,
		StrikeOut:       0,
		ScaleX:          100,
		ScaleY:          100,
		Spacing:         0,
		Angle:           0,
		Outline:         1,
		BorderStyle:     0,
		Shadow:          1,
		Alignment:       2,
		MarginL:         10,
		MarginR:         10,
		MarginV:         10,
		Encoding:        1,
		/*        [V4+ Styles]
		Format: , Bold, Italic, Underline, StrikeOut, ScaleX, ScaleY, Spacing, Angle, BorderStyle, Outline, Shadow, Alignment,
		Style: ,H0,0,0,0,0,100,100,0,0,1,1,0
		*/

	}

	// Clear existing styles and add default
	a.Styles.Body = []ass.Style{defaultStyle}

	// Apply Default style to all existing events
	for i := range a.Events.Body {
		a.Events.Body[i].Style = "real"
	}

	removeOverlappingEvents(&a.Events.Body)

	/*	log.Printf("%+v\n", a.ScriptInfo.Body)
		log.Printf("%+v\n", a.PGarbage)
		log.Printf("%+v\n", a.Styles)
		log.Printf("%+v\n", a.Events)
		for _, v := range a.Styles.Body {
			log.Printf("%+v\n", v)
		}
	*/
	// Create a new style for invisible text
	fakeStyle := ass.Style{
		Name:            "fake",
		Fontname:        "Arial",
		Fontsize:        0,            // Minimal size
		PrimaryColour:   "&HFF000000", // Fully transparent (FF alpha)
		SecondaryColour: "&HFF000000", // Fully transparent
		OutlineColour:   "&HFF000000", // Fully transparent
		Backcolour:      "&HFF000000", // Fully transparent
		Bold:            0,
		Italic:          0,
		Underline:       0,
		StrikeOut:       0,
		ScaleX:          0,
		ScaleY:          0,
		Spacing:         0,
		Angle:           0,
		BorderStyle:     0,
		Outline:         0,
		Shadow:          0,
		Alignment:       4,
		MarginL:         10,
		MarginR:         10,
		MarginV:         10,
		Encoding:        0,
	}

	a.Styles.Body = append(a.Styles.Body, fakeStyle)
	log.Println(a.Styles.Format)

	for _, v := range a.Events.Body {
		event := ass.Createevent(
			"Dialogue",
			0,
			v.Start,
			v.End,
			"fake",
			"",
			0,
			0,
			0,
			"",
			"ahh",
		)
		a.Events.Body = append(a.Events.Body, *event)
	}
	err = ass.WriteAss(a, "./subtitles/out/bold.ass")
	if err != nil {
		panic(err)
	}
}

func removeOverlappingEvents(events *[]ass.Event) {
	eventList := *events

	sort.SliceStable(eventList, func(i, j int) bool {
		return eventList[i].Start < eventList[j].Start
	})

	for i := 0; i < len(eventList)-1; i++ {
		if eventList[i].End > eventList[i+1].Start {
			eventList[i].End = eventList[i+1].Start
		}
	}

	*events = eventList
}
