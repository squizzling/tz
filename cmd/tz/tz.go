package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/gdamore/tcell/v2"

	"github.com/squizzling/tz/internal/clipboard"
)

func Must(err error) {
	if err != nil {
		panic(err)
	}
}

func Must1[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

func main() {
	var names []string
	names = os.Args[1:]
	if len(os.Args) == 1 { // If there's no arguments, create 2 blank columns
		names = make([]string, 2)
	} else if len(os.Args) == 2 { // If there's exactly 1 argument, and it can be parsed as an integer, make that many blank columns.
		if n, err := strconv.Atoi(os.Args[1]); err == nil {
			names = make([]string, n)
		}
	}

	s := Must1(tcell.NewScreen())
	Must(s.Init())
	defer s.Fini()

	now := time.Now().Round(time.Minute).UTC()
	a := &app{
		s: s,
		timezones: []*time.Location{
			Must1(time.LoadLocation("US/Pacific")),
			Must1(time.LoadLocation("US/Mountain")),
			Must1(time.LoadLocation("US/Central")),
			Must1(time.LoadLocation("US/Eastern")),
			time.UTC,
			Must1(time.LoadLocation("Asia/Kolkata")),
			Must1(time.LoadLocation("Australia/Sydney")),
			Must1(time.LoadLocation("Australia/Perth")),
			Must1(time.LoadLocation("Pacific/Auckland")),
		},
	}

	a.names = names
	for i := range a.names {
		a.times = append(a.times, now.Add(time.Duration(i)*time.Hour))
	}

	for a.poll() {
		a.render()
	}
}

func drawText(
	s tcell.Screen,
	x1 int, y1 int,
	x2 int, y2 int,
	style tcell.Style,
	text string,
) {
	row := y1
	col := x1
	for _, r := range []rune(text) {
		s.SetContent(col, row, r, nil, style)
		col++
		if col >= x2 {
			row++
			col = x1
		}
		if row > y2 {
			break
		}
	}
}

type app struct {
	s tcell.Screen

	timeIndex int
	tzIndex   int
	names     []string
	times     []time.Time

	message string

	timezones []*time.Location
}

func (a *app) poll() bool {
	a.s.Show()
	switch ev := a.s.PollEvent().(type) {
	case *tcell.EventResize:
		a.s.Sync()
	case *tcell.EventKey:
		message := ""
		switch ev.Key() {
		case tcell.KeyEscape, tcell.KeyCtrlC:
			return false
		case tcell.KeyEnter:
			a.copyTable()
			message = fmt.Sprintf("Copied %s", a.times[0].Format("2006-01-02 15:04 MST"))
		case tcell.KeyLeft, tcell.KeyRight:
			moveAmount := time.Duration(0)
			if ev.Modifiers()&tcell.ModCtrl != 0 {
				moveAmount = 24 * time.Hour
			} else if ev.Modifiers()&tcell.ModShift != 0 {
				moveAmount = time.Hour
			} else if ev.Modifiers() == 0 {
				moveAmount = time.Minute
			}

			if ev.Key() == tcell.KeyLeft {
				moveAmount = -moveAmount
			}

			a.times[a.timeIndex] = a.times[a.timeIndex].Add(moveAmount)

		case tcell.KeyBacktab:
			a.timeIndex = (len(a.times) + a.timeIndex - 1) % len(a.times)
		case tcell.KeyTab:
			a.timeIndex = (len(a.times) + a.timeIndex + 1) % len(a.times)
		case tcell.KeyUp:
			a.tzIndex = (len(a.timezones) + a.tzIndex - 1) % len(a.timezones)
		case tcell.KeyDown:
			a.tzIndex = (len(a.timezones) + a.tzIndex + 1) % len(a.timezones)
		default:
			message = fmt.Sprintf("%v", ev.Key())
		}
		a.message = message
	}
	return true
}

func timeInLoc(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("2006-01-02 15:04 MST")
}

func (a *app) writeTable(
	write func(s string, reverse bool, bold bool),
	newLine func(),
) {
	maxLen := make([]int, len(a.times))
	for column, name := range a.names {
		maxLen[column] = len(name)
	}
	for _, loc := range a.timezones {
		for column, t := range a.times {
			maxLen[column] = max(maxLen[column], len(timeInLoc(t, loc)))
		}
	}

	renderHeader := false
	for _, n := range a.names {
		if n != "" {
			renderHeader = true
		}
	}

	// Print headers
	if renderHeader {
		for column, name := range a.names {
			if column > 0 {
				write("   ", false, false)
			}
			write(padRight(name, maxLen[column]), false, column == a.timeIndex)
		}
		newLine()
	}

	// Print each timezone
	for y, loc := range a.timezones {
		for column, t := range a.times {
			if column > 0 {
				write("   ", y == a.tzIndex, false)
			}
			write(padRight(timeInLoc(t, loc), maxLen[column]), y == a.tzIndex, column == a.timeIndex)
		}
		newLine()
	}
}

func (a *app) render() {
	a.s.Clear()
	w, h := a.s.Size()

	startX := 0
	y := 0
	write := func(s string, reverse bool, bold bool) {
		drawText(a.s, startX, y, w, h, tcell.StyleDefault.Reverse(reverse).Bold(bold), s)
		startX += len(s)
	}
	newLine := func() {
		startX = 0
		y++
	}

	a.writeTable(write, newLine)
	write(a.message, false, false)
}

func (a *app) copyTable() {
	var sb strings.Builder
	write := func(s string, reverse bool, bold bool) { sb.WriteString(s) }
	newLine := func() { sb.WriteByte('\n') }
	a.writeTable(write, newLine)
	clipboard.Set(sb.String())
}

func padRight(s string, n int) string {
	return s + strings.Repeat(" ", n-len(s))
}
