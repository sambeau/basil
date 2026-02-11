package evaluator

import (
	"math"
	"math/rand/v2"
	"sort"
	"sync"
)

// mathRNG is the random number generator for std/math
// It uses a seeded PCG generator for reproducibility when seed() is called
var (
	mathRNG   *rand.Rand
	mathRNGMu sync.Mutex
)

func init() {
	// Initialize with a random seed by default
	mathRNG = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
}

var mathModuleMeta = ModuleMeta{
	Description: "Mathematical functions and constants",
	Exports: map[string]ExportMeta{
		// Constants
		"PI":  {Kind: "constant", Description: "Pi (3.14159...)"},
		"E":   {Kind: "constant", Description: "Euler's number (2.71828...)"},
		"TAU": {Kind: "constant", Description: "Tau (2*Pi)"},
		// Rounding
		"floor": {Kind: "function", Arity: "1", Description: "Round down to integer"},
		"ceil":  {Kind: "function", Arity: "1", Description: "Round up to integer"},
		"round": {Kind: "function", Arity: "1-2", Description: "Round to nearest (decimals?)"},
		"trunc": {Kind: "function", Arity: "1", Description: "Truncate to integer"},
		// Comparison & Clamping
		"abs":   {Kind: "function", Arity: "1", Description: "Absolute value"},
		"sign":  {Kind: "function", Arity: "1", Description: "Sign (-1, 0, or 1)"},
		"clamp": {Kind: "function", Arity: "3", Description: "Clamp value between min and max"},
		// Aggregation
		"min":     {Kind: "function", Arity: "1+", Description: "Minimum of values or array"},
		"max":     {Kind: "function", Arity: "1+", Description: "Maximum of values or array"},
		"sum":     {Kind: "function", Arity: "1+", Description: "Sum of values or array"},
		"avg":     {Kind: "function", Arity: "1+", Description: "Average of values or array"},
		"mean":    {Kind: "function", Arity: "1+", Description: "Mean (alias for avg)"},
		"product": {Kind: "function", Arity: "1+", Description: "Product of values or array"},
		"count":   {Kind: "function", Arity: "1", Description: "Count elements in array"},
		// Statistics
		"median":   {Kind: "function", Arity: "1", Description: "Median of array"},
		"mode":     {Kind: "function", Arity: "1", Description: "Mode of array"},
		"stddev":   {Kind: "function", Arity: "1", Description: "Standard deviation"},
		"variance": {Kind: "function", Arity: "1", Description: "Variance"},
		"range":    {Kind: "function", Arity: "1", Description: "Range (max - min)"},
		// Random
		"random":    {Kind: "function", Arity: "0", Description: "Random float 0-1"},
		"randomInt": {Kind: "function", Arity: "1-2", Description: "Random int (max) or (min, max)"},
		"seed":      {Kind: "function", Arity: "1", Description: "Seed random generator"},
		// Powers & Logarithms
		"sqrt":  {Kind: "function", Arity: "1", Description: "Square root"},
		"pow":   {Kind: "function", Arity: "2", Description: "Power (base, exponent)"},
		"exp":   {Kind: "function", Arity: "1", Description: "e^x"},
		"log":   {Kind: "function", Arity: "1", Description: "Natural logarithm"},
		"log10": {Kind: "function", Arity: "1", Description: "Base-10 logarithm"},
		// Trigonometry
		"sin":   {Kind: "function", Arity: "1", Description: "Sine (radians)"},
		"cos":   {Kind: "function", Arity: "1", Description: "Cosine (radians)"},
		"tan":   {Kind: "function", Arity: "1", Description: "Tangent (radians)"},
		"asin":  {Kind: "function", Arity: "1", Description: "Arc sine"},
		"acos":  {Kind: "function", Arity: "1", Description: "Arc cosine"},
		"atan":  {Kind: "function", Arity: "1", Description: "Arc tangent"},
		"atan2": {Kind: "function", Arity: "2", Description: "Arc tangent of y/x"},
		// Angular Conversion
		"degrees": {Kind: "function", Arity: "1", Description: "Radians to degrees"},
		"radians": {Kind: "function", Arity: "1", Description: "Degrees to radians"},
		// Geometry & Interpolation
		"hypot": {Kind: "function", Arity: "2", Description: "Hypotenuse length"},
		"dist":  {Kind: "function", Arity: "4", Description: "Distance between points"},
		"lerp":  {Kind: "function", Arity: "3", Description: "Linear interpolation"},
		"map":   {Kind: "function", Arity: "5", Description: "Map value from one range to another"},
	},
}

// loadMathModule returns the math module as a StdlibModuleDict
func loadMathModule(env *Environment) Object {
	return &StdlibModuleDict{
		Meta: &mathModuleMeta,
		Exports: map[string]Object{
			// Constants
			"PI":  &Float{Value: math.Pi},
			"E":   &Float{Value: math.E},
			"TAU": &Float{Value: math.Pi * 2},

			// Rounding
			"floor": &Builtin{Fn: mathFloor},
			"ceil":  &Builtin{Fn: mathCeil},
			"round": &Builtin{Fn: mathRound},
			"trunc": &Builtin{Fn: mathTrunc},

			// Comparison & Clamping
			"abs":   &Builtin{Fn: mathAbs},
			"sign":  &Builtin{Fn: mathSign},
			"clamp": &Builtin{Fn: mathClamp},

			// Aggregation (2 args OR array)
			"min":     &Builtin{Fn: mathMin},
			"max":     &Builtin{Fn: mathMax},
			"sum":     &Builtin{Fn: mathSum},
			"avg":     &Builtin{Fn: mathAvg},
			"mean":    &Builtin{Fn: mathAvg}, // alias
			"product": &Builtin{Fn: mathProduct},
			"count":   &Builtin{Fn: mathCount},

			// Statistics (array only)
			"median":   &Builtin{Fn: mathMedian},
			"mode":     &Builtin{Fn: mathMode},
			"stddev":   &Builtin{Fn: mathStddev},
			"variance": &Builtin{Fn: mathVariance},
			"range":    &Builtin{Fn: mathRange},

			// Random
			"random":    &Builtin{Fn: mathRandom},
			"randomInt": &Builtin{Fn: mathRandomInt},
			"seed":      &Builtin{Fn: mathSeed},

			// Powers & Logarithms
			"sqrt":  &Builtin{Fn: mathSqrt},
			"pow":   &Builtin{Fn: mathPow},
			"exp":   &Builtin{Fn: mathExp},
			"log":   &Builtin{Fn: mathLog},
			"log10": &Builtin{Fn: mathLog10},

			// Trigonometry
			"sin":   &Builtin{Fn: mathSin},
			"cos":   &Builtin{Fn: mathCos},
			"tan":   &Builtin{Fn: mathTan},
			"asin":  &Builtin{Fn: mathAsin},
			"acos":  &Builtin{Fn: mathAcos},
			"atan":  &Builtin{Fn: mathAtan},
			"atan2": &Builtin{Fn: mathAtan2},

			// Angular Conversion
			"degrees": &Builtin{Fn: mathDegrees},
			"radians": &Builtin{Fn: mathRadians},

			// Geometry & Interpolation
			"hypot": &Builtin{Fn: mathHypot},
			"dist":  &Builtin{Fn: mathDist},
			"lerp":  &Builtin{Fn: mathLerp},
			"map":   &Builtin{Fn: mathMap},
		},
	}
}

// =============================================================================
// Helper functions
// =============================================================================

// toFloat64 converts a Parsley number (Integer or Float) to float64
func toFloat64(obj Object) (float64, bool) {
	switch v := obj.(type) {
	case *Integer:
		return float64(v.Value), true
	case *Float:
		return v.Value, true
	default:
		return 0, false
	}
}

// toInt64 converts a Parsley number (Integer or Float) to int64
func toInt64(obj Object) (int64, bool) {
	switch v := obj.(type) {
	case *Integer:
		return v.Value, true
	case *Float:
		return int64(v.Value), true
	default:
		return 0, false
	}
}

// numberResult returns an Integer if the value is a whole number, Float otherwise
func numberResult(v float64) Object {
	if v == math.Trunc(v) && v >= math.MinInt64 && v <= math.MaxInt64 {
		return &Integer{Value: int64(v)}
	}
	return &Float{Value: v}
}

// extractNumbers extracts float64 values from an array of Objects
func extractNumbers(arr *Array, funcName string) ([]float64, Object) {
	if len(arr.Elements) == 0 {
		return nil, newValueError("VALUE-0001", map[string]any{"Function": funcName})
	}
	nums := make([]float64, len(arr.Elements))
	for i, elem := range arr.Elements {
		f, ok := toFloat64(elem)
		if !ok {
			return nil, newTypeError("TYPE-0005", funcName, "an array of numbers", elem.Type())
		}
		nums[i] = f
	}
	return nums, nil
}

// =============================================================================
// Rounding Functions
// =============================================================================

func mathFloor(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.floor", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.floor", "a number", args[0].Type())
	}
	return &Integer{Value: int64(math.Floor(f))}
}

func mathCeil(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.ceil", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.ceil", "a number", args[0].Type())
	}
	return &Integer{Value: int64(math.Ceil(f))}
}

func mathRound(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.round", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.round", "a number", args[0].Type())
	}
	return &Integer{Value: int64(math.Round(f))}
}

func mathTrunc(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.trunc", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.trunc", "a number", args[0].Type())
	}
	return &Integer{Value: int64(math.Trunc(f))}
}

// =============================================================================
// Comparison & Clamping Functions
// =============================================================================

func mathAbs(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.abs", len(args), 1)
	}
	switch v := args[0].(type) {
	case *Integer:
		if v.Value < 0 {
			return &Integer{Value: -v.Value}
		}
		return v
	case *Float:
		return &Float{Value: math.Abs(v.Value)}
	default:
		return newTypeError("TYPE-0005", "math.abs", "a number", args[0].Type())
	}
}

func mathSign(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.sign", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.sign", "a number", args[0].Type())
	}
	if f > 0 {
		return &Integer{Value: 1}
	} else if f < 0 {
		return &Integer{Value: -1}
	}
	return &Integer{Value: 0}
}

func mathClamp(args ...Object) Object {
	if len(args) != 3 {
		return newArityError("math.clamp", len(args), 3)
	}
	x, ok1 := toFloat64(args[0])
	min, ok2 := toFloat64(args[1])
	max, ok3 := toFloat64(args[2])
	if !ok1 {
		return newTypeError("TYPE-0005", "math.clamp", "a number", args[0].Type())
	}
	if !ok2 {
		return newTypeError("TYPE-0006", "math.clamp", "a number (min)", args[1].Type())
	}
	if !ok3 {
		return newTypeError("TYPE-0006", "math.clamp", "a number (max)", args[2].Type())
	}
	result := math.Max(min, math.Min(max, x))
	return numberResult(result)
}

// =============================================================================
// Aggregation Functions (2 args OR array)
// =============================================================================

func mathMin(args ...Object) Object {
	if len(args) == 1 {
		// Array mode
		arr, ok := args[0].(*Array)
		if !ok {
			return newTypeError("TYPE-0005", "math.min", "an array or two numbers", args[0].Type())
		}
		nums, err := extractNumbers(arr, "math.min")
		if err != nil {
			return err
		}
		minVal := nums[0]
		for _, v := range nums[1:] {
			if v < minVal {
				minVal = v
			}
		}
		return numberResult(minVal)
	} else if len(args) == 2 {
		// Two args mode
		a, ok1 := toFloat64(args[0])
		b, ok2 := toFloat64(args[1])
		if !ok1 {
			return newTypeError("TYPE-0005", "math.min", "a number", args[0].Type())
		}
		if !ok2 {
			return newTypeError("TYPE-0006", "math.min", "a number", args[1].Type())
		}
		return numberResult(math.Min(a, b))
	}
	return newArityErrorExact("math.min", len(args), 1, 2)
}

func mathMax(args ...Object) Object {
	if len(args) == 1 {
		// Array mode
		arr, ok := args[0].(*Array)
		if !ok {
			return newTypeError("TYPE-0005", "math.max", "an array or two numbers", args[0].Type())
		}
		nums, err := extractNumbers(arr, "math.max")
		if err != nil {
			return err
		}
		maxVal := nums[0]
		for _, v := range nums[1:] {
			if v > maxVal {
				maxVal = v
			}
		}
		return numberResult(maxVal)
	} else if len(args) == 2 {
		// Two args mode
		a, ok1 := toFloat64(args[0])
		b, ok2 := toFloat64(args[1])
		if !ok1 {
			return newTypeError("TYPE-0005", "math.max", "a number", args[0].Type())
		}
		if !ok2 {
			return newTypeError("TYPE-0006", "math.max", "a number", args[1].Type())
		}
		return numberResult(math.Max(a, b))
	}
	return newArityErrorExact("math.max", len(args), 1, 2)
}

func mathSum(args ...Object) Object {
	if len(args) == 1 {
		// Array mode
		arr, ok := args[0].(*Array)
		if !ok {
			return newTypeError("TYPE-0005", "math.sum", "an array or two numbers", args[0].Type())
		}
		if len(arr.Elements) == 0 {
			return &Integer{Value: 0}
		}
		nums, err := extractNumbers(arr, "math.sum")
		if err != nil {
			return err
		}
		sum := 0.0
		for _, v := range nums {
			sum += v
		}
		return numberResult(sum)
	} else if len(args) == 2 {
		// Two args mode
		a, ok1 := toFloat64(args[0])
		b, ok2 := toFloat64(args[1])
		if !ok1 {
			return newTypeError("TYPE-0005", "math.sum", "a number", args[0].Type())
		}
		if !ok2 {
			return newTypeError("TYPE-0006", "math.sum", "a number", args[1].Type())
		}
		return numberResult(a + b)
	}
	return newArityErrorExact("math.sum", len(args), 1, 2)
}

func mathAvg(args ...Object) Object {
	if len(args) == 1 {
		// Array mode
		arr, ok := args[0].(*Array)
		if !ok {
			return newTypeError("TYPE-0005", "math.avg", "an array or two numbers", args[0].Type())
		}
		nums, err := extractNumbers(arr, "math.avg")
		if err != nil {
			return err
		}
		sum := 0.0
		for _, v := range nums {
			sum += v
		}
		return &Float{Value: sum / float64(len(nums))}
	} else if len(args) == 2 {
		// Two args mode
		a, ok1 := toFloat64(args[0])
		b, ok2 := toFloat64(args[1])
		if !ok1 {
			return newTypeError("TYPE-0005", "math.avg", "a number", args[0].Type())
		}
		if !ok2 {
			return newTypeError("TYPE-0006", "math.avg", "a number", args[1].Type())
		}
		return &Float{Value: (a + b) / 2}
	}
	return newArityErrorExact("math.avg", len(args), 1, 2)
}

func mathProduct(args ...Object) Object {
	if len(args) == 1 {
		// Array mode
		arr, ok := args[0].(*Array)
		if !ok {
			return newTypeError("TYPE-0005", "math.product", "an array or two numbers", args[0].Type())
		}
		if len(arr.Elements) == 0 {
			return &Integer{Value: 1}
		}
		nums, err := extractNumbers(arr, "math.product")
		if err != nil {
			return err
		}
		product := 1.0
		for _, v := range nums {
			product *= v
		}
		return numberResult(product)
	} else if len(args) == 2 {
		// Two args mode
		a, ok1 := toFloat64(args[0])
		b, ok2 := toFloat64(args[1])
		if !ok1 {
			return newTypeError("TYPE-0005", "math.product", "a number", args[0].Type())
		}
		if !ok2 {
			return newTypeError("TYPE-0006", "math.product", "a number", args[1].Type())
		}
		return numberResult(a * b)
	}
	return newArityErrorExact("math.product", len(args), 1, 2)
}

func mathCount(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.count", len(args), 1)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0005", "math.count", "an array", args[0].Type())
	}
	return &Integer{Value: int64(len(arr.Elements))}
}

// =============================================================================
// Statistics Functions (array only)
// =============================================================================

func mathMedian(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.median", len(args), 1)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0005", "math.median", "an array", args[0].Type())
	}
	nums, err := extractNumbers(arr, "math.median")
	if err != nil {
		return err
	}

	// Sort a copy
	sorted := make([]float64, len(nums))
	copy(sorted, nums)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		// Even: average of two middle values
		return &Float{Value: (sorted[n/2-1] + sorted[n/2]) / 2}
	}
	// Odd: middle value
	return numberResult(sorted[n/2])
}

func mathMode(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.mode", len(args), 1)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0005", "math.mode", "an array", args[0].Type())
	}
	nums, err := extractNumbers(arr, "math.mode")
	if err != nil {
		return err
	}

	// Count frequencies
	freq := make(map[float64]int)
	for _, v := range nums {
		freq[v]++
	}

	// Find mode (highest frequency, smallest value on tie)
	var mode float64
	maxCount := 0
	first := true
	for v, count := range freq {
		if count > maxCount || (count == maxCount && (first || v < mode)) {
			mode = v
			maxCount = count
			first = false
		}
	}

	return numberResult(mode)
}

func mathStddev(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.stddev", len(args), 1)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0005", "math.stddev", "an array", args[0].Type())
	}
	nums, err := extractNumbers(arr, "math.stddev")
	if err != nil {
		return err
	}

	variance := calculateVariance(nums)
	return &Float{Value: math.Sqrt(variance)}
}

func mathVariance(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.variance", len(args), 1)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0005", "math.variance", "an array", args[0].Type())
	}
	nums, err := extractNumbers(arr, "math.variance")
	if err != nil {
		return err
	}

	return &Float{Value: calculateVariance(nums)}
}

// calculateVariance computes population variance
func calculateVariance(nums []float64) float64 {
	if len(nums) == 0 {
		return 0
	}
	// Calculate mean
	sum := 0.0
	for _, v := range nums {
		sum += v
	}
	mean := sum / float64(len(nums))

	// Calculate sum of squared differences
	sumSq := 0.0
	for _, v := range nums {
		diff := v - mean
		sumSq += diff * diff
	}
	return sumSq / float64(len(nums))
}

func mathRange(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.range", len(args), 1)
	}
	arr, ok := args[0].(*Array)
	if !ok {
		return newTypeError("TYPE-0005", "math.range", "an array", args[0].Type())
	}
	nums, err := extractNumbers(arr, "math.range")
	if err != nil {
		return err
	}

	minVal, maxVal := nums[0], nums[0]
	for _, v := range nums[1:] {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}
	return numberResult(maxVal - minVal)
}

// =============================================================================
// Random Functions
// =============================================================================

func mathRandom(args ...Object) Object {
	mathRNGMu.Lock()
	defer mathRNGMu.Unlock()

	switch len(args) {
	case 0:
		// random() - returns 0.0 to <1.0
		return &Float{Value: mathRNG.Float64()}
	case 1:
		// random(max) - returns 0.0 to <max
		max, ok := toFloat64(args[0])
		if !ok {
			return newTypeError("TYPE-0005", "math.random", "a number", args[0].Type())
		}
		return &Float{Value: mathRNG.Float64() * max}
	case 2:
		// random(min, max) - returns min to <max
		min, ok1 := toFloat64(args[0])
		max, ok2 := toFloat64(args[1])
		if !ok1 {
			return newTypeError("TYPE-0005", "math.random", "a number", args[0].Type())
		}
		if !ok2 {
			return newTypeError("TYPE-0006", "math.random", "a number", args[1].Type())
		}
		return &Float{Value: min + mathRNG.Float64()*(max-min)}
	default:
		return newArityErrorRange("math.random", len(args), 0, 2)
	}
}

func mathRandomInt(args ...Object) Object {
	mathRNGMu.Lock()
	defer mathRNGMu.Unlock()

	switch len(args) {
	case 1:
		// randomInt(max) - returns 0 to max (inclusive)
		max, ok := toInt64(args[0])
		if !ok {
			return newTypeError("TYPE-0005", "math.randomInt", "an integer", args[0].Type())
		}
		return &Integer{Value: mathRNG.Int64N(max + 1)}
	case 2:
		// randomInt(min, max) - returns min to max (inclusive)
		min, ok1 := toInt64(args[0])
		max, ok2 := toInt64(args[1])
		if !ok1 {
			return newTypeError("TYPE-0005", "math.randomInt", "an integer", args[0].Type())
		}
		if !ok2 {
			return newTypeError("TYPE-0006", "math.randomInt", "an integer", args[1].Type())
		}
		return &Integer{Value: min + mathRNG.Int64N(max-min+1)}
	default:
		return newArityErrorExact("math.randomInt", len(args), 1, 2)
	}
}

func mathSeed(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.seed", len(args), 1)
	}
	seed, ok := toInt64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.seed", "an integer", args[0].Type())
	}

	mathRNGMu.Lock()
	mathRNG = rand.New(rand.NewPCG(uint64(seed), 0))
	mathRNGMu.Unlock()

	return NULL
}

// =============================================================================
// Powers & Logarithms
// =============================================================================

func mathSqrt(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.sqrt", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.sqrt", "a number", args[0].Type())
	}
	if f < 0 {
		return newValueError("VALUE-0003", map[string]any{
			"Function": "math.sqrt",
			"Reason":   "cannot take square root of negative number",
		})
	}
	return numberResult(math.Sqrt(f))
}

func mathPow(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("math.pow", len(args), 2)
	}
	base, ok1 := toFloat64(args[0])
	exp, ok2 := toFloat64(args[1])
	if !ok1 {
		return newTypeError("TYPE-0005", "math.pow", "a number", args[0].Type())
	}
	if !ok2 {
		return newTypeError("TYPE-0006", "math.pow", "a number", args[1].Type())
	}
	return numberResult(math.Pow(base, exp))
}

func mathExp(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.exp", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.exp", "a number", args[0].Type())
	}
	return &Float{Value: math.Exp(f)}
}

func mathLog(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.log", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.log", "a number", args[0].Type())
	}
	if f <= 0 {
		return newValueError("VALUE-0003", map[string]any{
			"Function": "math.log",
			"Reason":   "logarithm requires a positive number",
		})
	}
	return &Float{Value: math.Log(f)}
}

func mathLog10(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.log10", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.log10", "a number", args[0].Type())
	}
	if f <= 0 {
		return newValueError("VALUE-0003", map[string]any{
			"Function": "math.log10",
			"Reason":   "logarithm requires a positive number",
		})
	}
	return &Float{Value: math.Log10(f)}
}

// =============================================================================
// Trigonometry Functions
// =============================================================================

func mathSin(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.sin", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.sin", "a number", args[0].Type())
	}
	return &Float{Value: math.Sin(f)}
}

func mathCos(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.cos", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.cos", "a number", args[0].Type())
	}
	return &Float{Value: math.Cos(f)}
}

func mathTan(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.tan", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.tan", "a number", args[0].Type())
	}
	return &Float{Value: math.Tan(f)}
}

func mathAsin(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.asin", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.asin", "a number", args[0].Type())
	}
	if f < -1 || f > 1 {
		return newValueError("VALUE-0003", map[string]any{
			"Function": "math.asin",
			"Reason":   "input must be in range [-1, 1]",
		})
	}
	return &Float{Value: math.Asin(f)}
}

func mathAcos(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.acos", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.acos", "a number", args[0].Type())
	}
	if f < -1 || f > 1 {
		return newValueError("VALUE-0003", map[string]any{
			"Function": "math.acos",
			"Reason":   "input must be in range [-1, 1]",
		})
	}
	return &Float{Value: math.Acos(f)}
}

func mathAtan(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.atan", len(args), 1)
	}
	f, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.atan", "a number", args[0].Type())
	}
	return &Float{Value: math.Atan(f)}
}

func mathAtan2(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("math.atan2", len(args), 2)
	}
	y, ok1 := toFloat64(args[0])
	x, ok2 := toFloat64(args[1])
	if !ok1 {
		return newTypeError("TYPE-0005", "math.atan2", "a number (y)", args[0].Type())
	}
	if !ok2 {
		return newTypeError("TYPE-0006", "math.atan2", "a number (x)", args[1].Type())
	}
	return &Float{Value: math.Atan2(y, x)}
}

// =============================================================================
// Angular Conversion Functions
// =============================================================================

func mathDegrees(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.degrees", len(args), 1)
	}
	rad, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.degrees", "a number", args[0].Type())
	}
	return numberResult(rad * 180 / math.Pi)
}

func mathRadians(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("math.radians", len(args), 1)
	}
	deg, ok := toFloat64(args[0])
	if !ok {
		return newTypeError("TYPE-0005", "math.radians", "a number", args[0].Type())
	}
	return &Float{Value: deg * math.Pi / 180}
}

// =============================================================================
// Geometry & Interpolation Functions
// =============================================================================

func mathHypot(args ...Object) Object {
	if len(args) != 2 {
		return newArityError("math.hypot", len(args), 2)
	}
	x, ok1 := toFloat64(args[0])
	y, ok2 := toFloat64(args[1])
	if !ok1 {
		return newTypeError("TYPE-0005", "math.hypot", "a number", args[0].Type())
	}
	if !ok2 {
		return newTypeError("TYPE-0006", "math.hypot", "a number", args[1].Type())
	}
	return numberResult(math.Hypot(x, y))
}

func mathDist(args ...Object) Object {
	if len(args) != 4 {
		return newArityError("math.dist", len(args), 4)
	}
	x1, ok1 := toFloat64(args[0])
	y1, ok2 := toFloat64(args[1])
	x2, ok3 := toFloat64(args[2])
	y2, ok4 := toFloat64(args[3])
	if !ok1 {
		return newTypeError("TYPE-0005", "math.dist", "a number (x1)", args[0].Type())
	}
	if !ok2 {
		return newTypeError("TYPE-0006", "math.dist", "a number (y1)", args[1].Type())
	}
	if !ok3 {
		return newTypeError("TYPE-0006", "math.dist", "a number (x2)", args[2].Type())
	}
	if !ok4 {
		return newTypeError("TYPE-0006", "math.dist", "a number (y2)", args[3].Type())
	}
	dx := x2 - x1
	dy := y2 - y1
	return numberResult(math.Sqrt(dx*dx + dy*dy))
}

func mathLerp(args ...Object) Object {
	if len(args) != 3 {
		return newArityError("math.lerp", len(args), 3)
	}
	a, ok1 := toFloat64(args[0])
	b, ok2 := toFloat64(args[1])
	t, ok3 := toFloat64(args[2])
	if !ok1 {
		return newTypeError("TYPE-0005", "math.lerp", "a number (start)", args[0].Type())
	}
	if !ok2 {
		return newTypeError("TYPE-0006", "math.lerp", "a number (end)", args[1].Type())
	}
	if !ok3 {
		return newTypeError("TYPE-0006", "math.lerp", "a number (t)", args[2].Type())
	}
	return numberResult(a + (b-a)*t)
}

func mathMap(args ...Object) Object {
	if len(args) != 5 {
		return newArityError("math.map", len(args), 5)
	}
	value, ok1 := toFloat64(args[0])
	inMin, ok2 := toFloat64(args[1])
	inMax, ok3 := toFloat64(args[2])
	outMin, ok4 := toFloat64(args[3])
	outMax, ok5 := toFloat64(args[4])
	if !ok1 {
		return newTypeError("TYPE-0005", "math.map", "a number (value)", args[0].Type())
	}
	if !ok2 {
		return newTypeError("TYPE-0006", "math.map", "a number (inMin)", args[1].Type())
	}
	if !ok3 {
		return newTypeError("TYPE-0006", "math.map", "a number (inMax)", args[2].Type())
	}
	if !ok4 {
		return newTypeError("TYPE-0006", "math.map", "a number (outMin)", args[3].Type())
	}
	if !ok5 {
		return newTypeError("TYPE-0006", "math.map", "a number (outMax)", args[4].Type())
	}

	// Linear interpolation: map value from [inMin, inMax] to [outMin, outMax]
	t := (value - inMin) / (inMax - inMin)
	return numberResult(outMin + (outMax-outMin)*t)
}
