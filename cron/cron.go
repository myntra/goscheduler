package cron

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Generate a list of values from start to end with increment as common difference.
func steps(start, end, increment int64) []int64 {
	var steps []int64

	for ; start < end; start += increment {
		steps = append(steps, start)
	}

	return steps
}

// Parse a string and return a list of corresponding values
// A step string is represented by a start and increment value separated by a forward slash(/).
// If the start value is "*" in the input string, the range will start from zero. Else the range will start from
// the immediate multiple of increment vale.
// The outputs a list of values from start to _range with common difference as increment.
// For example,
//		Input		Output
//		*/10		0, 10, 20, .... _range
// 		12/10		20, 30, 40, ... _range
//
// Return a non empty error string if the string cannot be parsed to a range.
func ParseStep(s string, _range int64) ([]int64, string) {
	parts := strings.Split(s, "/")
	if len(parts) == 2 {
		increment, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return []int64{}, fmt.Sprintf("Cannot parse step value from %s", s)
		}

		var start int64 = 0
		if parts[0] != "*" {
			var err error

			start, err = strconv.ParseInt(parts[0], 10, 64)
			if err != nil {
				return []int64{}, fmt.Sprintf("Cannot parse start value from %s", s)
			}

			// Start from the next multiple of increment.
			start = (start + increment - 1) / increment * increment
		}

		if start > _range {
			return []int64{}, fmt.Sprintf("Start cannot be greater than range")
		}

		return steps(start, _range, increment), ""
	}

	if len(parts) != 1 {
		return []int64{}, fmt.Sprintf("Invalid cron format %s", s)
	}

	return []int64{}, ""
}

// Parse a string and return a list of values within the range.
// A range string is represented by a start and end value separated by a dash(-).
// The generated list will be inclusive of both start and end values, [start, end]. The start and end values should
// be within the min and max values supplied in the params.
// Return a non empty error string if the string cannot be parsed to a range.
func ParseRange(s string, min, max int64) ([]int64, string) {
	switch parts := strings.Split(s, "-"); {
	case len(parts) == 2:
		start, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil {
			return []int64{}, fmt.Sprintf("Cannot parse start value from %s", s)
		}

		end, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return []int64{}, fmt.Sprintf("Cannot parse end value from %s", s)
		}

		if start <= end && start >= min && end <= max {
			return steps(start, end+1, 1), ""
		} else {
			return []int64{}, fmt.Sprintf("Invalid start and/or end in range %s", s)
		}
	case len(parts) != 1:
		return []int64{}, fmt.Sprintf("Invalid cron format %s", s)
	default:
		return []int64{}, ""
	}
}

// Minute represents int64 value of Minute on which the cron will be active.
// Allowed value should be within range [0-59]
type Minute int64

// Parse a string to Minute type.
// Returns a non empty string error message if the string cannot be parsed to a valid Minute.
//
// Allowed values for Minute are,
//	- [0-59]
//
// These values can be either be,
//	- single
//	- comma separated, indicating distinct values.
//	- dash(-) separated indicating a range of values(Both inclusive).
func ParseMinute(s string) ([]Minute, string) {
	toMinute := func(list []int64) []Minute {
		var minutes []Minute

		for _, minute := range list {
			minutes = append(minutes, Minute(minute))
		}

		return minutes
	}

	if s != "*" {
		if steps, message := ParseStep(s, 60); len(message) != 0 {
			return []Minute{}, message
		} else if len(steps) > 0 {
			return toMinute(steps), ""
		}

		var minutes []Minute
		for _, part := range strings.Split(s, ",") {
			switch steps, message := ParseRange(part, 0, 59); {
			case len(message) > 0:
				return []Minute{}, message
			case len(steps) > 0:
				minutes = append(minutes, toMinute(steps)...)
			default:
				switch minute, err := strconv.ParseInt(part, 10, 64); {
				case err != nil:
					return []Minute{}, fmt.Sprintf("Cannot parse %s to int", part)
				case minute < 0 || minute > 59:
					return []Minute{}, "Minute should be between 0 and 59"
				default:
					minutes = append(minutes, Minute(minute))
				}
			}
		}

		return minutes, ""

	}

	return []Minute{}, ""
}

// Hour represents int64 value of Hour on which the cron will be active.
// Allowed value should be within range [0-23]
type Hour int64

// Parse a string to Hour type.
// Returns a non empty string error message if the string cannot be parsed to a valid Hour.
//
// Allowed values for Hour are,
//	- [0-23]
//
// These values can be either be,
//	- single
//	- comma separated, indicating distinct values.
//	- dash(-) separated indicating a range of values(Both inclusive).
func ParseHour(s string) ([]Hour, string) {
	toHour := func(list []int64) []Hour {
		var hours []Hour

		for _, hour := range list {
			hours = append(hours, Hour(hour))
		}

		return hours
	}

	if s != "*" {
		if steps, message := ParseStep(s, 24); len(message) != 0 {
			return []Hour{}, message
		} else if len(steps) > 0 {
			return toHour(steps), ""
		}

		var hours []Hour
		for _, part := range strings.Split(s, ",") {
			switch steps, message := ParseRange(part, 0, 23); {
			case len(message) > 0:
				return []Hour{}, message
			case len(steps) > 0:
				hours = append(hours, toHour(steps)...)
			default:
				switch hour, err := strconv.ParseInt(part, 10, 64); {
				case err != nil:
					return []Hour{}, fmt.Sprintf("Cannot parse %s to int", part)
				case hour < 0 || hour > 23:
					return []Hour{}, "Hour should be between 0 and 24"
				default:
					hours = append(hours, Hour(hour))
				}
			}
		}

		return hours, ""
	}

	return []Hour{}, ""
}

// Day represents int64 value of day on which the cron will be active.
// Allowed value should be within range [1-31]
type Day int64

// Parse a string to Day type.
// Returns a non empty string error message if the string cannot be parsed to a valid Day.
//
// Allowed values for Day are,
//	- [1-31]
//
// These values can be either be,
//	- single
//	- comma separated, indicating distinct values.
//	- dash(-) separated indicating a range of values(Both inclusive).
func ParseDay(s string) ([]Day, string) {
	toDay := func(list []int64) []Day {
		var days []Day
		for _, day := range list {
			if day > 0 {
				days = append(days, Day(day))
			}
		}

		return days
	}

	if s != "*" {
		if steps, message := ParseStep(s, 31); len(message) != 0 {
			return []Day{}, message
		} else if len(steps) > 0 {
			return toDay(steps), ""
		}

		var days []Day
		for _, part := range strings.Split(s, ",") {
			switch steps, message := ParseRange(part, 1, 31); {
			case len(message) > 0:
				return []Day{}, message
			case len(steps) > 0:
				days = append(days, toDay(steps)...)
			default:
				switch day, err := strconv.ParseInt(part, 10, 64); {
				case err != nil:
					return []Day{}, fmt.Sprintf("Cannot parse %s to int", part)
				case day < 1 || day > 31:
					return []Day{}, "Day should be between 1 and 31"
				default:
					days = append(days, Day(day))
				}
			}
		}

		return days, ""
	}

	return []Day{}, ""
}

// Month represents int64 value of month on which the cron will be active.
// Allowed value should be within range [1-12]
type Month int64

// Parse a string to Month type.
// Returns a non empty string error message if the string cannot be parsed to a valid Month.
//
// Allowed values for Month are,
//	- [1-12]
//	- Short names such as JAN/FEB etc in either cases.
//
// These values can be either be,
//	- single
//	- comma separated, indicating distinct values.
//	- dash(-) separated indicating a range of values(Both inclusive).
func ParseMonth(s string) ([]Month, string) {
	shortNames := []string{
		"JAN",
		"FEB",
		"MAR",
		"APR",
		"MAY",
		"JUN",
		"JUL",
		"AUG",
		"SEP",
		"OCT",
		"NOV",
		"DEC",
	}

	if s != "*" {
		var months []Month

	Parts:
		for _, part := range strings.Split(s, ",") {
			for i, val := range shortNames {
				if val == strings.ToUpper(part) {
					months = append(months, Month(i+1))
					continue Parts
				}
			}

			switch steps, message := ParseRange(part, 1, 12); {
			case len(message) > 0:
				return []Month{}, message
			case len(steps) > 0:
				for _, month := range steps {
					months = append(months, Month(month))
				}
			default:
				switch month, err := strconv.ParseInt(part, 10, 64); {
				case err != nil:
					return []Month{}, fmt.Sprintf("Cannot parse %s to int", part)
				case month < 1 || month > 12:
					return []Month{}, fmt.Sprintf("Month should be between 1 and 12")
				default:
					months = append(months, Month(month))
				}
			}
		}

		return months, ""
	}

	return []Month{}, ""
}

// Weekday represents int64 value of day within a week on which the cron will be active.
// Allowed value should be within range [0-6]
type Weekday int64

// Parse a string to Weekday type.
// Returns a non empty string error message if the string cannot be parsed to a valid Weekday
//
// Allowed values for Weekday are,
//	- [0-6]
//	- Short names such as SUN/MON etc in either cases.
//
// These values can be either be,
//	- single
//	- comma separated, indicating distinct values.
//	- dash(-) separated indicating a range of values(Both inclusive).
func ParseWeekday(s string) ([]Weekday, string) {
	shortNames := []string{
		"SUN",
		"MON",
		"TUE",
		"WED",
		"THU",
		"FRI",
		"SAT",
	}

	if s != "*" {
		var weekdays []Weekday

	Parts:
		for _, part := range strings.Split(s, ",") {
			for i, val := range shortNames {
				if val == strings.ToUpper(part) {
					weekdays = append(weekdays, Weekday(i))
					continue Parts
				}
			}

			switch steps, message := ParseRange(part, 0, 6); {
			case len(message) > 0:
				return []Weekday{}, message
			case len(steps) > 0:
				for _, weekday := range steps {
					weekdays = append(weekdays, Weekday(weekday))
				}
			default:
				switch weekday, err := strconv.ParseInt(part, 10, 64); {
				case err != nil:
					return []Weekday{}, fmt.Sprintf("Cannot parse %s to int", part)
				case weekday < 0 || weekday > 6:
					return []Weekday{}, fmt.Sprintf("Weekday should be between 0 and 6")
				default:
					weekdays = append(weekdays, Weekday(weekday))
				}
			}
		}

		return weekdays, ""
	}

	return []Weekday{}, ""
}

// Convert a list of type to int64 values.
func toInt64(list interface{}) []int64 {
	var output []int64

	switch list.(type) {
	case []Minute:
		for _, value := range list.([]Minute) {
			output = append(output, int64(value))
		}
	case []Hour:
		for _, value := range list.([]Hour) {
			output = append(output, int64(value))
		}
	case []Day:
		for _, value := range list.([]Day) {
			output = append(output, int64(value))
		}
	case []Month:
		for _, value := range list.([]Month) {
			output = append(output, int64(value))
		}
	case []Weekday:
		for _, value := range list.([]Weekday) {
			output = append(output, int64(value))
		}
	default:
		panic(fmt.Sprintf("Unknonw type %T supplied in toInt64", list))
	}

	return output
}

// Expression represents a cron expression.
// Each filed is a list of types. Each value in the fields corresponds to the time field where it's active.
type Expression struct {
	Minute  []Minute
	Hour    []Hour
	Day     []Day
	Month   []Month
	Weekday []Weekday
}

// Parse a string to a cron expression of type Expresion.
// A non empty list of error messages is returned if the supplied string cannot be parsed to Expression.
func Parse(s string) (Expression, []string) {
	var expression Expression
	var errors []string

	parts := strings.Split(s, " ")
	if len(parts) != 5 {
		return expression, []string{"String doesn't match valid cron format, \"* * * * *\""}
	}

	if minutes, err := ParseMinute(parts[0]); len(err) != 0 {
		errors = append(errors, err)
	} else {
		expression.Minute = minutes
	}

	if hours, err := ParseHour(parts[1]); len(err) != 0 {
		errors = append(errors, err)
	} else {
		expression.Hour = hours
	}

	if days, err := ParseDay(parts[2]); len(err) != 0 {
		errors = append(errors, err)
	} else {
		expression.Day = days
	}

	if months, err := ParseMonth(parts[3]); len(err) != 0 {
		errors = append(errors, err)
	} else {
		expression.Month = months
	}

	if weekdays, err := ParseWeekday(parts[4]); len(err) != 0 {
		errors = append(errors, err)
	} else {
		expression.Weekday = weekdays
	}

	return expression, errors
}

// Check if the con expression matches with the time provided.
// The match is true if the value of the 5 fields in time is found in the corresponding field list in the cron.
func (expression Expression) Match(time time.Time) bool {
	contains := func(list []int64, val int64) bool {
		if len(list) == 0 {
			return true
		}

		for _, item := range list {
			if item == val {
				return true
			}
		}

		return false
	}

	return contains(toInt64(expression.Minute), int64(time.Minute())) &&
		contains(toInt64(expression.Hour), int64(time.Hour())) &&
		contains(toInt64(expression.Day), int64(time.Day())) &&
		contains(toInt64(expression.Month), int64(time.Month())) &&
		contains(toInt64(expression.Weekday), int64(time.Weekday()))
}
