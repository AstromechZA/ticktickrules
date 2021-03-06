// Package ticktickrules provides a basic Cron-like rule matcher for doing simple calculations of
// cron expressions. It exposes functionality for determining the next time a cron expression is matched.
//
// Only the simple cron rules are available but this is pretty much good enough for most applications. If you
// want to support things like @hourly, @weekly, etc then you should combine this with higher level time windows.
package ticktickrules

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Rule is a structure encoding a Cron-like rule
type Rule struct {
	minute         []int
	minuteRule     string
	hour           []int
	hourRule       string
	dayOfWeek      []int
	dayOfWeekRule  string
	dayOfMonth     []int
	dayOfMonthRule string
	month          []int
	monthRule      string
}

// rule to support */10 */0 */1
var ruleType1 = regexp.MustCompile(`^\*/\d+$`)

// rule to support 0/10/20
var ruleType2 = regexp.MustCompile(`^\d+(?:/\d+)+$`)

func parseRuleItem(r string, maxsum int) ([]int, error) {
	var out []int
	if r == "*" {
		// noop
	} else if ruleType1.MatchString(r) {

		i := strings.Split(r, "/")[1]
		v, err := strconv.Atoi(i)
		if err != nil {
			return nil, fmt.Errorf("Rule item '%s' could not be parsed", r)
		}

		if v == 0 {
			return nil, fmt.Errorf("Rule item '%s' cannot be 0", r)
		}

		if v >= maxsum {
			return nil, fmt.Errorf("Rule item '%s' does not divide", r)
		}

		var sum int
		for {
			out = append(out, sum)
			sum += v
			if sum >= maxsum {
				break
			}
		}

	} else if ruleType2.MatchString(r) {

		parts := strings.Split(r, "/")
		lst := 0
		for _, p := range parts {
			v, err := strconv.Atoi(p)
			if err != nil {
				return nil, fmt.Errorf("Rule item '%s' could not be parsed", r)
			} else if v < 0 {
				return nil, fmt.Errorf("Rule item '%s' cannot have negative value", r)
			}

			if len(out) == 0 {
				out = append(out, v)
			} else if v <= lst {
				return nil, fmt.Errorf("Rule item '%s' has bad ordering", r)
			} else {
				out = append(out, v)
				lst = v
			}
		}

	} else {

		v, err := strconv.Atoi(r)
		if err != nil {
			return nil, fmt.Errorf("Rule item '%s' is not supported", r)
		}
		out = append(out, v)

	}
	return out, nil
}

func validateItemsRange(items []int, min int, max int) error {
	for _, i := range items {
		if i > max {
			return fmt.Errorf("%d is > %d", i, max)
		} else if i < min {
			return fmt.Errorf("%d is < %d", i, min)
		}
	}
	return nil
}

func doesMatch(v int, vs []int) bool {
	for _, i := range vs {
		if v == i {
			return true
		}
	}
	return false
}

const naiveMaxIterations = 31 * 8 * 12

func roundUp(current int, items []int, ceiling int) int {
	if len(items) == 0 {
		r := current + 1
		if r >= ceiling {
			return 0
		}
		return r
	}
	for _, i := range items {
		if i > current {
			return i
		}
	}
	return items[0]
}

// NewRule constructs and validates a new Rule structure from the cron-like arguments provided.
// Each rule string can be of the following forms:
//     "*" - matches any value
//     "*/N" - matches 0 and any multiple of N
//     "N/M/O.." - matches N or M or O, etc.
//
//     field         allowed values
//     -----         --------------
//     minute        0-59
//     hour          0-23
//     day of month  1-31
//     month         1-12
//     day of week   0-7 (0	or 7 is	Sun)
//
// An error will be returned if one of the rules is invalid.
func NewRule(minute, hour, dayOfMonth, month, dayOfWeek string) (*Rule, error) {
	output := new(Rule)

	m, err := parseRuleItem(minute, 60)
	if err != nil {
		return nil, err
	}
	output.minute = m
	if err := validateItemsRange(output.minute, 0, 59); err != nil {
		return nil, fmt.Errorf("Minute rule invalid: %s", err.Error())
	}
	output.minuteRule = minute

	h, err := parseRuleItem(hour, 24)
	if err != nil {
		return nil, err
	}
	output.hour = h
	if err := validateItemsRange(output.hour, 0, 23); err != nil {
		return nil, fmt.Errorf("Hour rule invalid: %s", err.Error())
	}
	output.hourRule = hour

	dow, err := parseRuleItem(dayOfWeek, 7)
	if err != nil {
		return nil, err
	}
	output.dayOfWeek = dow
	if err := validateItemsRange(output.dayOfWeek, 0, 7); err != nil {
		return nil, fmt.Errorf("Day of Week rule invalid: %s", err.Error())
	}
	output.dayOfWeekRule = dayOfWeek

	dom, err := parseRuleItem(dayOfMonth, 31)
	if err != nil {
		return nil, err
	}
	output.dayOfMonth = dom
	if err := validateItemsRange(output.dayOfMonth, 1, 31); err != nil {
		return nil, fmt.Errorf("Day of Month rule invalid: %s", err.Error())
	}
	output.dayOfMonthRule = dayOfMonth

	m, err = parseRuleItem(month, 24)
	if err != nil {
		return nil, err
	}
	output.month = m
	if err := validateItemsRange(output.month, 1, 12); err != nil {
		return nil, fmt.Errorf("Month rule invalid: %s", err.Error())
	}
	output.monthRule = month

	return output, nil
}

// MustNewRule is like NewRule but panics if there is an error parsing the rule
func MustNewRule(minute, hour, dayOfMonth, month, dayOfWeek string) *Rule {
	r, err := NewRule(minute, hour, dayOfMonth, month, dayOfWeek)
	if err != nil {
		panic(err)
	}
	return r
}

// String converts the rule back to its native 5-part cron expression.
func (r *Rule) String() string {
	return fmt.Sprintf("%s %s %s %s %s", r.minuteRule, r.hourRule, r.dayOfMonthRule, r.monthRule, r.dayOfWeekRule)
}

// NextUTC returns the next UTC time this rule is true.
func (r *Rule) NextUTC() time.Time {
	return r.NextAfter(time.Now().UTC())
}

// NextAfter returns the next time this rule will match after the given time.
func (r *Rule) NextAfter(from time.Time) time.Time {
	originalFrom := from
	originalMinute := from.Minute()
	originalHour := from.Hour()

	// first attempt to match in the current hour
	nextMinute := roundUp(originalMinute, r.minute, 60)
	from = time.Date(from.Year(), from.Month(), from.Day(), from.Hour(), nextMinute, 0, 0, from.Location())
	if from.After(originalFrom) {
		// double check (this shouldn't be needed)
		if r.Matches(from) {
			return from
		}
	}

	// either in the future but not matched, or in the past
	nextHour := roundUp(originalHour, r.hour, 24)
	from = time.Date(from.Year(), from.Month(), from.Day(), nextHour, from.Minute(), 0, 0, from.Location())

	if from.After(originalFrom) {
		if r.Matches(from) {
			return from
		}
	}

	// otherwise truncate to the first minute and hour and into the next day
	nextMinute = roundUp(59, r.minute, 60)
	nextHour = roundUp(23, r.hour, 24)
	from = from.Add(24 * time.Hour)
	from = time.Date(from.Year(), from.Month(), from.Day(), nextHour, nextMinute, 0, 0, from.Location())

	// now iterate in days until we hit a day that matches
	numIterations := 0
	for {
		if r.Matches(from) {
			return from.Truncate(time.Minute)
		}
		from = from.Add(24 * time.Hour)
		numIterations++
		if numIterations > naiveMaxIterations {
			return time.Unix(1<<62, 0)
		}
	}
}

// UntilNext returns the duration until the next match.
func (r *Rule) UntilNext(from time.Time) time.Duration {
	next := r.NextAfter(from)
	return next.Sub(from)
}

// UntilNextUTC returns the duration until the next match from the current UTC time.
func (r *Rule) UntilNextUTC() time.Duration {
	now := time.Now().UTC()
	next := r.NextAfter(now)
	return next.Sub(now)
}

// Matches returns whether the given time is matched by the rule.
func (r *Rule) Matches(t time.Time) bool {
	if len(r.month) > 0 {
		if !doesMatch(int(t.Month()), r.month) {
			return false
		}
	}
	if len(r.dayOfWeek) > 0 {
		if !doesMatch(int(t.Weekday()), r.dayOfWeek) {
			return false
		}
	}
	if len(r.dayOfMonth) > 0 {
		if !doesMatch(t.Day(), r.dayOfMonth) {
			return false
		}
	}
	if len(r.hour) > 0 {
		if !doesMatch(t.Hour(), r.hour) {
			return false
		}
	}
	if len(r.minute) > 0 {
		if !doesMatch(t.Minute(), r.minute) {
			return false
		}
	}
	return true
}
