package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/asticode/go-astisub"
	"github.com/joho/godotenv"
	"github.com/will-x86/subtitle-poisoning/libgosubs/ass"
)

func init() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}
}

var beeLines []string

func main() {
	bee, err := astisub.OpenFile("./bee.srt")
	if err != nil {
		panic(err)
	}
	for _, x := range bee.Items {
		for _, d := range x.Lines {
			beeLines = append(beeLines, d.String())
		}
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fs := http.FileServer(http.Dir("ui"))
	http.Handle("/", fs)

	http.HandleFunc("/convert", handleConvert)

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Unable to parse form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("subtitle")
	if err != nil {
		http.Error(w, "Error retrieving file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tempDir := "./temp"
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	tempFile := filepath.Join(tempDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename))
	f, err := os.Create(tempFile)
	if err != nil {
		http.Error(w, "Unable to create temp file", http.StatusInternalServerError)
		return
	}
	defer f.Close()
	defer os.Remove(tempFile)

	_, err = io.Copy(f, file)
	if err != nil {
		http.Error(w, "Unable to save file", http.StatusInternalServerError)
		return
	}

	outputFile := filepath.Join(tempDir, fmt.Sprintf("%d_output.ass", time.Now().UnixNano()))
	if err := processSubtitle(tempFile, outputFile); err != nil {
		http.Error(w, "Error processing subtitle", http.StatusInternalServerError)
		return
	}
	defer os.Remove(outputFile)

	w.Header().Set("Content-Disposition", "attachment; filename=converted.ass")
	w.Header().Set("Content-Type", "application/octet-stream")

	http.ServeFile(w, r, outputFile)
}

func processSubtitle(inputFile, outputFile string) error {
	s1, err := astisub.OpenFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}

	tempParsedAss := filepath.Join(filepath.Dir(outputFile), "parsed.ass")
	f, err := os.Create(tempParsedAss)
	if err != nil {
		return fmt.Errorf("failed to create temporary ASS file: %w", err)
	}
	defer os.Remove(tempParsedAss)
	defer f.Close()

	if s1.Metadata == nil {
		s1.Metadata = &astisub.Metadata{
			SSAScriptType: "v4.00+",
		}
	} else {
		s1.Metadata.SSAScriptType = "v4.00+"
	}

	err = s1.WriteToSSA(f)
	if err != nil {
		return fmt.Errorf("failed to write SSA: %w", err)
	}

	a, err := ass.ParseAss(tempParsedAss)
	if err != nil {
		return fmt.Errorf("failed to parse ASS: %w", err)
	}

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
	}

	a.Styles.Body = []ass.Style{defaultStyle}
	for i := range a.Events.Body {
		a.Events.Body[i].Style = "real"
	}

	removeOverlappingEvents(&a.Events.Body)

	fakeStyle := ass.Style{
		Name:            "fake",
		Fontname:        "Arial",
		Fontsize:        0,
		PrimaryColour:   "&HFF000000",
		SecondaryColour: "&HFF000000",
		OutlineColour:   "&HFF000000",
		Backcolour:      "&HFF000000",
		Bold:            0,
		Italic:          0,
		Underline:       0,
		StrikeOut:       0,
		ScaleX:          0,
		ScaleY:          0,
		Spacing:         0,
		Angle:           0,
		BorderStyle:     1,
		Outline:         0,
		Shadow:          0,
		Alignment:       4,
		MarginL:         10,
		MarginR:         10,
		MarginV:         10,
		Encoding:        0,
	}

	a.Styles.Body = append(a.Styles.Body, fakeStyle)
	var events []ass.Event
	for x, y := range beeLines {
		event := ass.Createevent(
			"Dialogue",
			0,
			a.Events.Body[x%len(a.Events.Body)].Start,
			a.Events.Body[x%len(a.Events.Body)].End,
			"fake",
			"",
			0,
			0,
			0,
			"",
			y,
		)
		events = append(events, *event)
	}
	for k := range events {
		a.Events.Body = append(a.Events.Body, events[k])
	}
	rand.Shuffle(len(a.Events.Body), func(i, j int) { a.Events.Body[i], a.Events.Body[j] = a.Events.Body[j], a.Events.Body[i] })

	err = ass.WriteAss(a, outputFile)
	if err != nil {
		return fmt.Errorf("failed to write final ASS: %w", err)
	}

	return nil
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
