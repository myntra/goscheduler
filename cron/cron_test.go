package cron

import (
	"testing"
	"time"
)

func assertEquals(output, expected []int64) bool {
	if len(output) != len(expected) {
		return false
	}

	for i := range output {
		if output[i] != expected[i] {
			return false
		}
	}

	return true
}

func TestParseStep(t *testing.T) {

	for _, test := range []struct {
		Input    string
		Range    int64
		Expected string
	}{
		{"*/abc", 60, "Cannot parse step value from */abc"},
		{"abc/20", 10, "Cannot parse start value from abc/20"},
		{"*/7/10", 10, "Invalid cron format */7/10"},
		{"15/10", 10, "Start cannot be greater than range"},
	} {
		if _, err := ParseStep(test.Input, test.Range); err != test.Expected {
			t.Errorf("Got error \"%s\" for input \"%s\"", err, test.Input)
		}
	}

	for _, test := range []struct {
		Input    string
		Range    int64
		Expected []int64
	}{
		{"*/10", 60, []int64{0, 10, 20, 30, 40, 50}},
		{"*/2", 10, []int64{0, 2, 4, 6, 8}},
		{"*/25", 100, []int64{0, 25, 50, 75}},
		{"*/20", 60, []int64{0, 20, 40}},
		{"4/2", 10, []int64{4, 6, 8}},
		{"5/2", 10, []int64{6, 8}},
		{"30/25", 100, []int64{50, 75}},
		{"40/25", 100, []int64{50, 75}},
	} {
		if steps, err := ParseStep(test.Input, test.Range); !assertEquals(steps, test.Expected) || len(err) != 0 {
			t.Errorf("Got result \"%v\", error \"%v\" for input \"%s\"", steps, err, test.Input)
		}
	}
}

func TestParseRange(t *testing.T) {
	for _, test := range []struct {
		Input    string
		Min      int64
		Max      int64
		Expected string
	}{
		{"abc-10", 0, 10, "Cannot parse start value from abc-10"},
		{"20-abc", 0, 10, "Cannot parse end value from 20-abc"},
		{"20-10", 0, 10, "Invalid start and/or end in range 20-10"},
		{"10-100", 0, 10, "Invalid start and/or end in range 10-100"},
		{"2-12", 5, 10, "Invalid start and/or end in range 2-12"},
		{"7-11", 5, 10, "Invalid start and/or end in range 7-11"},
		{"7-10-100", 0, 10, "Invalid cron format 7-10-100"},
	} {
		if _, err := ParseRange(test.Input, test.Min, test.Max); err != test.Expected {
			t.Errorf("Got error \"%s\" for input \"%s\"", err, test.Input)
		}
	}

	for _, test := range []struct {
		Input    string
		Min      int64
		Max      int64
		Expected []int64
	}{
		{"5-10", 0, 11, []int64{5, 6, 7, 8, 9, 10}},
		{"25-27", 0, 100, []int64{25, 26, 27}},
		{"70-80", 0, 100, []int64{70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80}},
		{"25-25", 0, 100, []int64{25}},
	} {
		if steps, err := ParseRange(test.Input, test.Min, test.Max); !assertEquals(steps, test.Expected) || len(err) != 0 {
			t.Errorf("Got result \"%v\", error \"%v\" for input \"%s\"", steps, err, test.Input)
		}
	}
}

func TestParseMinute(t *testing.T) {
	for _, test := range []struct {
		Input    string
		Expected string
	}{
		{"", "Cannot parse  to int"},
		{"**", "Cannot parse ** to int"},
		{"abc", "Cannot parse abc to int"},
		{"110", "Minute should be between 0 and 59"},
		{"1000", "Minute should be between 0 and 59"},
		{"*/abc", "Cannot parse step value from */abc"},
		{"*/7/10", "Invalid cron format */7/10"},
		{"10-20-30", "Invalid cron format 10-20-30"},
		{"10-abc", "Cannot parse end value from 10-abc"},
		{"abc-20", "Cannot parse start value from abc-20"},
		{"10,100,200", "Minute should be between 0 and 59"},
		{"10,abc,200", "Cannot parse abc to int"},
	} {
		if _, err := ParseMinute(test.Input); err != test.Expected {
			t.Errorf("Got error \"%s\" for input \"%s\"", err, test.Input)
		}
	}

	for _, test := range []struct {
		Input    string
		Expected []Minute
	}{
		{"*", []Minute{}},
		{"10", []Minute{10}},
		{"45", []Minute{45}},
		{"*/10", []Minute{0, 10, 20, 30, 40, 50}},
		{"*/15", []Minute{0, 15, 30, 45}},
		{"10-15", []Minute{10, 11, 12, 13, 14, 15}},
		{"20-22", []Minute{20, 21, 22}},
		{"20-22,25,35", []Minute{20, 21, 22, 25, 35}},
		{"10-15,17,20-22,25,35", []Minute{10, 11, 12, 13, 14, 15, 17, 20, 21, 22, 25, 35}},
		{"20/15", []Minute{30, 45}},
		{"10,20,30", []Minute{10, 20, 30}},
	} {
		if minutes, err := ParseMinute(test.Input); !assertEquals(toInt64(minutes), toInt64(test.Expected)) || len(err) != 0 {
			t.Errorf("Got result \"%v\", error \"%v\" for input \"%s\"", minutes, err, test.Input)
		}
	}
}

func TestParseHour(t *testing.T) {
	for _, test := range []struct {
		Input    string
		Expected string
	}{
		{"", "Cannot parse  to int"},
		{"**", "Cannot parse ** to int"},
		{"abc", "Cannot parse abc to int"},
		{"10,25", "Hour should be between 0 and 24"},
		{"10,abc,200", "Cannot parse abc to int"},
		{"1000", "Hour should be between 0 and 24"},
		{"*/abc", "Cannot parse step value from */abc"},
		{"*/5/10", "Invalid cron format */5/10"},
		{"10-20-30", "Invalid cron format 10-20-30"},
		{"10-abc", "Cannot parse end value from 10-abc"},
		{"abc-20", "Cannot parse start value from abc-20"},
		{"10-100", "Invalid start and/or end in range 10-100"},
	} {
		if _, err := ParseHour(test.Input); err != test.Expected {
			t.Errorf("Got error \"%s\" for input \"%s\"", err, test.Input)
		}
	}

	for _, test := range []struct {
		Input    string
		Expected []Hour
	}{
		{"*", []Hour{}},
		{"10", []Hour{10}},
		{"5", []Hour{5}},
		{"*/4", []Hour{0, 4, 8, 12, 16, 20}},
		{"*/6", []Hour{0, 6, 12, 18}},
		{"12/4", []Hour{12, 16, 20}},
		{"6,12,18", []Hour{6, 12, 18}},
		{"10-15", []Hour{10, 11, 12, 13, 14, 15}},
		{"5-6,10-15,17,20", []Hour{5, 6, 10, 11, 12, 13, 14, 15, 17, 20}},
		{"20-22", []Hour{20, 21, 22}},
	} {
		if hours, err := ParseHour(test.Input); !assertEquals(toInt64(hours), toInt64(test.Expected)) || len(err) != 0 {
			t.Errorf("Got result \"%v\", error \"%v\" for input \"%s\"", hours, err, test.Input)
		}
	}
}

func TestParseDay(t *testing.T) {
	for _, test := range []struct {
		Input    string
		Expected string
	}{
		{"", "Cannot parse  to int"},
		{"**", "Cannot parse ** to int"},
		{"abc", "Cannot parse abc to int"},
		{"0", "Day should be between 1 and 31"},
		{"1000", "Day should be between 1 and 31"},
		{"*/abc", "Cannot parse step value from */abc"},
		{"*/5/10", "Invalid cron format */5/10"},
		{"10,35", "Day should be between 1 and 31"},
		{"10,abc,200", "Cannot parse abc to int"},
		{"10-20-30", "Invalid cron format 10-20-30"},
		{"10-abc", "Cannot parse end value from 10-abc"},
		{"abc-20", "Cannot parse start value from abc-20"},
		{"10-32", "Invalid start and/or end in range 10-32"},
	} {
		if _, err := ParseDay(test.Input); err != test.Expected {
			t.Errorf("Got error \"%s\" for input \"%s\"", err, test.Input)
		}
	}

	for _, test := range []struct {
		Input    string
		Expected []Day
	}{
		{"*", []Day{}},
		{"10", []Day{10}},
		{"5", []Day{5}},
		{"*/5", []Day{5, 10, 15, 20, 25, 30}},
		{"16/5", []Day{20, 25, 30}},
		{"6,12,18", []Day{6, 12, 18}},
		{"10-15", []Day{10, 11, 12, 13, 14, 15}},
		{"20-22", []Day{20, 21, 22}},
		{"10-12,15-15,20-22", []Day{10, 11, 12, 15, 20, 21, 22}},
	} {
		if days, err := ParseDay(test.Input); !assertEquals(toInt64(days), toInt64(test.Expected)) || len(err) != 0 {
			t.Errorf("Got result \"%v\", error \"%v\" for input \"%s\"", days, err, test.Input)
		}
	}
}

func TestParseMonth(t *testing.T) {
	for _, test := range []struct {
		Input    string
		Expected string
	}{
		{"", "Cannot parse  to int"},
		{"**", "Cannot parse ** to int"},
		{"abc", "Cannot parse abc to int"},
		{"0", "Month should be between 1 and 12"},
		{"1000", "Month should be between 1 and 12"},
		{"10,35", "Month should be between 1 and 12"},
		{"10,abc,200", "Cannot parse abc to int"},
		{"10-20-30", "Invalid cron format 10-20-30"},
		{"10-abc", "Cannot parse end value from 10-abc"},
		{"abc-20", "Cannot parse start value from abc-20"},
		{"10-13", "Invalid start and/or end in range 10-13"},
		{"0-9", "Invalid start and/or end in range 0-9"},
	} {
		if _, err := ParseMonth(test.Input); err != test.Expected {
			t.Errorf("Got error \"%s\" for input \"%s\"", err, test.Input)
		}
	}

	for _, test := range []struct {
		Input    string
		Expected []Month
	}{
		{"*", []Month{}},
		{"10", []Month{10}},
		{"5", []Month{5}},
		{"JAN", []Month{1}},
		{"jan", []Month{1}},
		{"SEP", []Month{9}},
		{"DEC", []Month{12}},
		{"1,6,12", []Month{1, 6, 12}},
		{"JAN,JUN,DEC", []Month{1, 6, 12}},
		{"9-12", []Month{9, 10, 11, 12}},
		{"JAN,3,9-12", []Month{1, 3, 9, 10, 11, 12}},
	} {
		if months, err := ParseMonth(test.Input); !assertEquals(toInt64(months), toInt64(test.Expected)) || len(err) != 0 {
			t.Errorf("Got result \"%v\", error \"%v\" for input \"%s\"", months, err, test.Input)
		}
	}
}

func TestParseWeekday(t *testing.T) {
	for _, test := range []struct {
		Input    string
		Expected string
	}{
		{"", "Cannot parse  to int"},
		{"**", "Cannot parse ** to int"},
		{"abc", "Cannot parse abc to int"},
		{"1000", "Weekday should be between 0 and 6"},
		{"10,35", "Weekday should be between 0 and 6"},
		{"1,abc,30", "Cannot parse abc to int"},
		{"10-20-30", "Invalid cron format 10-20-30"},
		{"10-abc", "Cannot parse end value from 10-abc"},
		{"abc-20", "Cannot parse start value from abc-20"},
		{"10-13", "Invalid start and/or end in range 10-13"},
	} {
		if _, err := ParseWeekday(test.Input); err != test.Expected {
			t.Errorf("Got error \"%s\" for input \"%s\"", err, test.Input)
		}
	}

	for _, test := range []struct {
		Input    string
		Expected []Weekday
	}{
		{"*", []Weekday{}},
		{"0", []Weekday{0}},
		{"5", []Weekday{5}},
		{"SUN", []Weekday{0}},
		{"sun", []Weekday{0}},
		{"WED", []Weekday{3}},
		{"SAT", []Weekday{6}},
		{"0,3,6", []Weekday{0, 3, 6}},
		{"SUN,3,6", []Weekday{0, 3, 6}},
		{"SUN,WED,SAT", []Weekday{0, 3, 6}},
		{"0-3", []Weekday{0, 1, 2, 3}},
		{"SUN,1-3", []Weekday{0, 1, 2, 3}},
	} {
		if weekday, err := ParseWeekday(test.Input); !assertEquals(toInt64(weekday), toInt64(test.Expected)) || len(err) != 0 {
			t.Errorf("Got result \"%v\" error \"%v\" for input \"%s\"", weekday, err, test.Input)
		}
	}
}

func TestParse(t *testing.T) {
	assertErrorEquals := func(output, expected []string) bool {
		if len(output) != len(expected) {
			return false
		}

		for i := range output {
			if output[i] != expected[i] {
				return false
			}
		}

		return true
	}

	for _, test := range []struct {
		Input    string
		Expected []string
	}{
		{"", []string{
			"String doesn't match valid cron format, \"* * * * *\""}},
		{"* * * *", []string{
			"String doesn't match valid cron format, \"* * * * *\""}},
		{"* * * * * *", []string{
			"String doesn't match valid cron format, \"* * * * *\""}},
		{"abc * * * *", []string{
			"Cannot parse abc to int"}},
		{"10 abc * xyc 10", []string{
			"Cannot parse abc to int",
			"Cannot parse xyc to int",
			"Weekday should be between 0 and 6"}},
		{"100 50 * 25 *", []string{
			"Minute should be between 0 and 59",
			"Hour should be between 0 and 24",
			"Month should be between 1 and 12"}},
		{"10,100 0,50 * 1,25 *", []string{
			"Minute should be between 0 and 59",
			"Hour should be between 0 and 24",
			"Month should be between 1 and 12"}},
	} {
		if _, errors := Parse(test.Input); !assertErrorEquals(errors, test.Expected) {
			t.Errorf("Got error \"%v\" for input \"%s\"", errors, test.Input)
		}
	}

	assertExpressionEquals := func(output, expected Expression) bool {
		return assertEquals(toInt64(output.Minute), toInt64(expected.Minute)) &&
			assertEquals(toInt64(output.Hour), toInt64(expected.Hour)) &&
			assertEquals(toInt64(output.Day), toInt64(expected.Day)) &&
			assertEquals(toInt64(output.Month), toInt64(expected.Month)) &&
			assertEquals(toInt64(output.Weekday), toInt64(expected.Weekday))
	}

	for _, test := range []struct {
		Input    string
		Expected Expression
	}{
		{"* * * * *", Expression{
			Minute:  []Minute{},
			Hour:    []Hour{},
			Day:     []Day{},
			Month:   []Month{},
			Weekday: []Weekday{},
		}},
		{"10 20 * * *", Expression{
			Minute:  []Minute{10},
			Hour:    []Hour{20},
			Day:     []Day{},
			Month:   []Month{},
			Weekday: []Weekday{},
		}},
		{"10-12 20-23 * * *", Expression{
			Minute:  []Minute{10, 11, 12},
			Hour:    []Hour{20, 21, 22, 23},
			Day:     []Day{},
			Month:   []Month{},
			Weekday: []Weekday{},
		}},
		{"10,20,30 0,12 * * *", Expression{
			Minute:  []Minute{10, 20, 30},
			Hour:    []Hour{0, 12},
			Day:     []Day{},
			Month:   []Month{},
			Weekday: []Weekday{},
		}},
		{"00 00 * * MON", Expression{
			Minute:  []Minute{0},
			Hour:    []Hour{0},
			Day:     []Day{},
			Month:   []Month{},
			Weekday: []Weekday{1},
		}},
		{"00 00 * JAN MON", Expression{
			Minute:  []Minute{0},
			Hour:    []Hour{0},
			Day:     []Day{},
			Month:   []Month{1},
			Weekday: []Weekday{1},
		}},
		{"00 00 15 JAN MON", Expression{
			Minute:  []Minute{0},
			Hour:    []Hour{0},
			Day:     []Day{15},
			Month:   []Month{1},
			Weekday: []Weekday{1},
		}},
		{"00 10/5 15 JAN MON", Expression{
			Minute:  []Minute{0},
			Hour:    []Hour{10, 15, 20},
			Day:     []Day{15},
			Month:   []Month{1},
			Weekday: []Weekday{1},
		}},
		{"*/20 00 15 JAN MON", Expression{
			Minute:  []Minute{0, 20, 40},
			Hour:    []Hour{0},
			Day:     []Day{15},
			Month:   []Month{1},
			Weekday: []Weekday{1},
		}},
		{"*/10 00 15 JAN MON", Expression{
			Minute:  []Minute{0, 10, 20, 30, 40, 50},
			Hour:    []Hour{0},
			Day:     []Day{15},
			Month:   []Month{1},
			Weekday: []Weekday{1},
		}},
	} {
		if expression, err := Parse(test.Input); !assertExpressionEquals(expression, test.Expected) || len(err) != 0 {
			t.Errorf("Got result \"%v\", error \"%v\" for input \"%v\"", expression, err, test.Input)
		}
	}
}

func TestMatch(t *testing.T) {
	asTime := func(layout, value string) time.Time {
		t, _ := time.Parse(layout, value)
		return t
	}

	type params struct {
		Cron Expression
		Time time.Time
	}

	for _, test := range []struct {
		Input    params
		Expected bool
	}{
		{
			params{
				Expression{
					Minute:  []Minute{},
					Hour:    []Hour{},
					Day:     []Day{},
					Month:   []Month{},
					Weekday: []Weekday{},
				},
				time.Now(),
			},
			true,
		},
		{
			params{
				Expression{
					Minute:  []Minute{5, 10, 15, 20},
					Hour:    []Hour{},
					Day:     []Day{},
					Month:   []Month{},
					Weekday: []Weekday{},
				},
				asTime("15:04", "18:10"),
			},
			true,
		},
		{
			params{
				Expression{
					Minute:  []Minute{20},
					Hour:    []Hour{},
					Day:     []Day{},
					Month:   []Month{},
					Weekday: []Weekday{},
				},
				asTime("15:04", "18:10"),
			},
			false,
		},
		{
			params{
				Expression{
					Minute:  []Minute{18},
					Hour:    []Hour{18},
					Day:     []Day{20},
					Month:   []Month{3},
					Weekday: []Weekday{},
				},
				asTime("Jan 2 15:04", "Mar 20 18:18"),
			},
			true,
		},
		{
			params{
				Expression{
					Minute:  []Minute{18},
					Hour:    []Hour{0, 6, 12, 18},
					Day:     []Day{1},
					Month:   []Month{5},
					Weekday: []Weekday{5},
				},
				asTime("Mon Jan 2 15:04 2006", "Fri May 1 18:18 2020"),
			},
			true,
		},
		{
			params{
				Expression{
					Minute:  []Minute{18},
					Hour:    []Hour{},
					Day:     []Day{1},
					Month:   []Month{5},
					Weekday: []Weekday{5},
				},
				asTime("Mon Jan 2 15:04 2006", "Fri May 1 10:18 2020"),
			},
			true,
		},
	} {
		if match := test.Input.Cron.Match(test.Input.Time); match != test.Expected {
			t.Errorf("Got result \"%v\" for input \"%v\"", match, test.Input)
		}
	}
}
