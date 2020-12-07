package config

import "fmt"

var (
	Second = "second"
	Minute = "minute"
	Hour   = "hour"
	Day    = "day"
)

type unitConversion func(int) float64

func divideBy(factor int) unitConversion {
	return func(n int) float64 {
		return float64(n) / float64(factor)
	}
}

func multiplyBy(factor int) unitConversion {
	return func(n int) float64 {
		return float64(n * factor)
	}
}

func identity(n int) float64 {
	return float64(n)
}

var conversionFactors map[string]map[string]unitConversion = map[string]map[string]unitConversion{
	Second: {
		Second: identity,
		Minute: multiplyBy(60),
		Hour:   multiplyBy(3600),
		Day:    multiplyBy(3600 * 24),
	},
	Minute: {
		Second: divideBy(60),
		Minute: identity,
		Hour:   multiplyBy(60),
		Day:    multiplyBy(60 * 24),
	},
	Hour: {
		Second: divideBy(3600),
		Minute: divideBy(60),
		Hour:   identity,
		Day:    multiplyBy(24),
	},
	Day: {
		Second: divideBy(3600 * 24),
		Minute: divideBy(60 * 24),
		Hour:   divideBy(24),
		Day:    identity,
	},
}

// ConvertRate converts a rate of <value> requests/<from> to requests/<to>
//
// Example: ConvertRate("hour", "minute", 1200) will convert 100 requests/hour
// to 20 requests/minute
func ConvertRate(from string, to string, value int) (float64, error) {
	toFuncs, ok := conversionFactors[from]
	if !ok {
		return 0, fmt.Errorf(`rate to convert from "%s" not supported`, from)
	}

	convert, ok := toFuncs[to]
	if !ok {
		return 0, fmt.Errorf(`rate to convert to "%s" not supported`, to)
	}

	return convert(value), nil
}
