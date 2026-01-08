package evaluator

import (
"strings"

"github.com/sambeau/basil/pkg/parsley/ast"
"github.com/sambeau/basil/pkg/parsley/lexer"
)

// Collection operations: set operations, chunking, repetition, ranges

func evalArrayIntersection(left, right *Array) Object {
	// Build hash set of right array elements for O(n) lookup
	rightSet := make(map[string]bool)
	for _, elem := range right.Elements {
		rightSet[elem.Inspect()] = true
	}

	// Keep elements from left that exist in right, deduplicate
	seen := make(map[string]bool)
	result := []Object{}
	for _, elem := range left.Elements {
		key := elem.Inspect()
		if rightSet[key] && !seen[key] {
			result = append(result, elem)
			seen[key] = true
		}
	}

	return &Array{Elements: result}
}

// evalDictionaryIntersection returns keys present in both dictionaries with values from left
func evalDictionaryIntersection(left, right *Dictionary) Object {
	result := &Dictionary{
		Pairs: make(map[string]ast.Expression),
		Env:   left.Env,
	}

	// Keep only keys that exist in both dictionaries
	for k, v := range left.Pairs {
		if _, exists := right.Pairs[k]; exists {
			result.Pairs[k] = v
		}
	}

	return result
}

// evalArrayUnion returns all unique elements from both arrays
func evalArrayUnion(left, right *Array) Object {
	seen := make(map[string]bool)
	result := []Object{}

	// Add elements from left
	for _, elem := range left.Elements {
		key := elem.Inspect()
		if !seen[key] {
			result = append(result, elem)
			seen[key] = true
		}
	}

	// Add elements from right
	for _, elem := range right.Elements {
		key := elem.Inspect()
		if !seen[key] {
			result = append(result, elem)
			seen[key] = true
		}
	}

	return &Array{Elements: result}
}

// evalArraySubtraction removes elements present in right from left
func evalArraySubtraction(left, right *Array) Object {
	// Build hash set of elements to remove
	removeSet := make(map[string]bool)
	for _, elem := range right.Elements {
		removeSet[elem.Inspect()] = true
	}

	// Keep elements from left that are not in removeSet
	result := []Object{}
	for _, elem := range left.Elements {
		if !removeSet[elem.Inspect()] {
			result = append(result, elem)
		}
	}

	return &Array{Elements: result}
}

// evalDictionarySubtraction removes keys present in right from left
func evalDictionarySubtraction(left, right *Dictionary) Object {
	result := &Dictionary{
		Pairs: make(map[string]ast.Expression),
		Env:   left.Env,
	}

	// Keep keys from left that don't exist in right
	for k, v := range left.Pairs {
		if _, exists := right.Pairs[k]; !exists {
			result.Pairs[k] = v
		}
	}

	return result
}

// evalArrayChunking splits array into chunks of specified size
func evalArrayChunking(tok lexer.Token, array *Array, size *Integer) Object {
	chunkSize := int(size.Value)

	if chunkSize <= 0 {
		return newValidationError("VAL-0012", map[string]any{"Got": chunkSize})
	}

	result := []Object{}
	for i := 0; i < len(array.Elements); i += chunkSize {
		end := i + chunkSize
		if end > len(array.Elements) {
			end = len(array.Elements)
		}
		chunk := &Array{Elements: array.Elements[i:end]}
		result = append(result, chunk)
	}

	return &Array{Elements: result}
}

// evalStringRepetition repeats a string n times
func evalStringRepetition(str *String, count *Integer) Object {
	n := int(count.Value)

	if n <= 0 {
		return &String{Value: ""}
	}

	var builder strings.Builder
	builder.Grow(len(str.Value) * n)
	for i := 0; i < n; i++ {
		builder.WriteString(str.Value)
	}

	return &String{Value: builder.String()}
}

// evalArrayRepetition repeats an array n times
func evalArrayRepetition(array *Array, count *Integer) Object {
	n := int(count.Value)

	if n <= 0 {
		return &Array{Elements: []Object{}}
	}

	result := make([]Object, 0, len(array.Elements)*n)
	for i := 0; i < n; i++ {
		result = append(result, array.Elements...)
	}

	return &Array{Elements: result}
}

// evalRangeExpression creates an inclusive range from start to end
func evalRangeExpression(tok lexer.Token, left, right Object) Object {
	if left.Type() != INTEGER_OBJ {
		return newValidationError("VAL-0013", map[string]any{"Got": left.Type()})
	}
	if right.Type() != INTEGER_OBJ {
		return newValidationError("VAL-0014", map[string]any{"Got": right.Type()})
	}

	start := left.(*Integer).Value
	end := right.(*Integer).Value

	// Calculate size and direction
	var size int64
	var step int64
	if start <= end {
		size = end - start + 1
		step = 1
	} else {
		size = start - end + 1
		step = -1
	}

	// Pre-allocate array
	elements := make([]Object, size)
	val := start
	for i := int64(0); i < size; i++ {
		elements[i] = &Integer{Value: val}
		val += step
	}

	return &Array{Elements: elements}
}
