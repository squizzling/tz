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
	n := 2
	if len(os.Args) > 1 {
		i, err := strconv.ParseInt(os.Args[1], 10, 64)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Failed to parse column count: %s\n", err)
			os.Exit(1)
		} else {
			n = int(i)
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
			Must1(time.LoadLocation("Pacific/Auckland")),
		},
	}

	for i := range n {
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

func (a *app) render() {
	a.s.Clear()
	w, h := a.s.Size()
	maxLen := make([]int, len(a.times))
	for _, loc := range a.timezones {
		for x, t := range a.times {
			maxLen[x] = max(maxLen[x], len(t.In(loc).Format("2006-01-02 15:04 MST")))
		}
	}
	for y, loc := range a.timezones {
		startX := 0
		for x, t := range a.times {
			style := tcell.StyleDefault
			style = style.Reverse(y == a.tzIndex)

			ts := padRight(t.In(loc).Format("2006-01-02 15:04 MST"), maxLen[x])

			if x > 0 {
				drawText(a.s, startX, y, w, h, style.Reverse(y == a.tzIndex), "   ")
				startX += 3
			}
			drawText(a.s, startX, y, w, h, style.Reverse(y == a.tzIndex).Bold(x == a.timeIndex), ts)
			startX += len(ts)
		}
	}
	drawText(a.s, 0, len(a.timezones), w, h, tcell.StyleDefault, a.message)
}

func (a *app) copyTable() {
	maxLen := make([]int, len(a.times))
	for _, loc := range a.timezones {
		for x, t := range a.times {
			maxLen[x] = max(maxLen[x], len(t.In(loc).Format("2006-01-02 15:04 MST")))
		}
	}

	var sb strings.Builder
	for _, loc := range a.timezones {
		for x, t := range a.times {
			if x > 0 {
				sb.WriteString("   ")
			}
			sb.WriteString(padRight(t.In(loc).Format("2006-01-02 15:04 MST"), maxLen[x]))
		}
		sb.WriteString("\r\n")
	}
	clipboard.Set(sb.String())
}

func padRight(s string, n int) string {
	return s + strings.Repeat(" ", n-len(s))
}
