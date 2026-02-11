package evaluator

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"time"
)

var idModuleMeta = ModuleMeta{
	Description: "ID generation (UUID, nanoid, etc.)",
	Exports: map[string]ExportMeta{
		"new":    {Kind: "function", Arity: "0", Description: "Generate new unique ID"},
		"uuid":   {Kind: "function", Arity: "0", Description: "Generate UUID v4"},
		"uuidv4": {Kind: "function", Arity: "0", Description: "Generate UUID v4"},
		"uuidv7": {Kind: "function", Arity: "0", Description: "Generate UUID v7"},
		"nanoid": {Kind: "function", Arity: "0-1", Description: "Generate nanoid (length?)"},
		"cuid":   {Kind: "function", Arity: "0", Description: "Generate CUID"},
	},
}

// loadIDModule returns the id module as a StdlibModuleDict
func loadIDModule(env *Environment) Object {
	return &StdlibModuleDict{
		Meta: &idModuleMeta,
		Exports: map[string]Object{
			"new":    &Builtin{Fn: idNew},
			"uuid":   &Builtin{Fn: idUUID},
			"uuidv4": &Builtin{Fn: idUUID},
			"uuidv7": &Builtin{Fn: idUUIDv7},
			"nanoid": &Builtin{Fn: idNanoID},
			"cuid":   &Builtin{Fn: idCUID},
		},
	}
}

// =============================================================================
// ID Generators
// =============================================================================

// ULID encoding alphabet (Crockford's Base32)
const ulidAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NanoID default alphabet
const nanoidAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz-"

// Counter for CUID (thread-safe)
var (
	cuidCounter uint32
	cuidMutex   sync.Mutex
)

// idNew generates a new ULID-like ID (sortable, URL-safe)
// Format: 26 characters, time-sortable, uses Crockford's Base32
func idNew(args ...Object) Object {
	if len(args) > 0 {
		return newArityError("id.new", len(args), 0)
	}

	// Get current timestamp in milliseconds
	now := time.Now().UnixMilli()

	// Generate 10 bytes: 6 for timestamp, 10 for randomness (80 bits)
	var id [26]byte

	// Encode timestamp (first 10 chars)
	timestamp := uint64(now)
	for i := 9; i >= 0; i-- {
		id[i] = ulidAlphabet[timestamp&0x1F]
		timestamp >>= 5
	}

	// Generate random bytes for the rest (16 chars = 80 bits)
	var randomBytes [10]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return &Error{Message: "Failed to generate random bytes"}
	}

	// Encode random bytes
	var randBits uint64
	randBits = binary.BigEndian.Uint64(randomBytes[:8])
	for i := 17; i >= 10; i-- {
		id[i] = ulidAlphabet[randBits&0x1F]
		randBits >>= 5
	}

	// Last 8 chars from remaining random bits
	randBits = uint64(randomBytes[8])<<8 | uint64(randomBytes[9])
	for i := 25; i >= 18; i-- {
		id[i] = ulidAlphabet[randBits&0x1F]
		randBits >>= 5
	}

	return &String{Value: string(id[:])}
}

// idUUID generates a UUID v4 (random)
func idUUID(args ...Object) Object {
	if len(args) > 0 {
		return newArityError("id.uuid", len(args), 0)
	}

	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		return &Error{Message: "Failed to generate random bytes"}
	}

	// Set version 4 and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant RFC 4122

	return &String{Value: fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])}
}

// idUUIDv7 generates a UUID v7 (time-sortable)
func idUUIDv7(args ...Object) Object {
	if len(args) > 0 {
		return newArityError("id.uuidv7", len(args), 0)
	}

	var uuid [16]byte

	// Get current timestamp in milliseconds
	now := time.Now().UnixMilli()

	// First 48 bits: timestamp
	uuid[0] = byte(now >> 40)
	uuid[1] = byte(now >> 32)
	uuid[2] = byte(now >> 24)
	uuid[3] = byte(now >> 16)
	uuid[4] = byte(now >> 8)
	uuid[5] = byte(now)

	// Generate random bytes for the rest
	if _, err := rand.Read(uuid[6:]); err != nil {
		return &Error{Message: "Failed to generate random bytes"}
	}

	// Set version 7 and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x70 // Version 7
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant RFC 4122

	return &String{Value: fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])}
}

// idNanoID generates a NanoID (compact, URL-safe)
func idNanoID(args ...Object) Object {
	length := 21 // Default length

	if len(args) == 1 {
		if l, ok := args[0].(*Integer); ok {
			length = int(l.Value)
			if length < 1 || length > 256 {
				return &Error{Message: "NanoID length must be between 1 and 256"}
			}
		} else {
			return newTypeError("TYPE-0001", "id.nanoid", "integer", args[0].Type())
		}
	} else if len(args) > 1 {
		return newArityError("id.nanoid", len(args), 1)
	}

	// Generate random bytes
	randomBytes := make([]byte, length)
	if _, err := rand.Read(randomBytes); err != nil {
		return &Error{Message: "Failed to generate random bytes"}
	}

	// Build ID using alphabet
	var id strings.Builder
	id.Grow(length)
	for i := 0; i < length; i++ {
		id.WriteByte(nanoidAlphabet[randomBytes[i]&63]) // 64 chars in alphabet
	}

	return &String{Value: id.String()}
}

// idCUID generates a CUID2-like ID (secure, collision-resistant)
func idCUID(args ...Object) Object {
	if len(args) > 0 {
		return newArityError("id.cuid", len(args), 0)
	}

	// CUID2 format: c + timestamp (base36) + counter + random
	var id strings.Builder
	id.Grow(25)

	// Prefix
	id.WriteByte('c')

	// Timestamp (base36, 8 chars)
	timestamp := time.Now().UnixMilli()
	timestampStr := base36Encode(uint64(timestamp))
	// Pad to 8 chars
	for i := len(timestampStr); i < 8; i++ {
		id.WriteByte('0')
	}
	id.WriteString(timestampStr)

	// Counter (4 chars)
	cuidMutex.Lock()
	cuidCounter++
	counter := cuidCounter
	cuidMutex.Unlock()
	counterStr := base36Encode(uint64(counter))
	for i := len(counterStr); i < 4; i++ {
		id.WriteByte('0')
	}
	id.WriteString(counterStr)

	// Random (12 chars)
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		return &Error{Message: "Failed to generate random bytes"}
	}
	randNum := binary.BigEndian.Uint64(randomBytes)
	randStr := base36Encode(randNum)
	// Take last 12 chars or pad
	if len(randStr) > 12 {
		id.WriteString(randStr[len(randStr)-12:])
	} else {
		for i := len(randStr); i < 12; i++ {
			id.WriteByte('0')
		}
		id.WriteString(randStr)
	}

	return &String{Value: id.String()}
}

// base36Encode encodes a number to base36
func base36Encode(n uint64) string {
	if n == 0 {
		return "0"
	}

	const alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"
	var result strings.Builder

	for n > 0 {
		result.WriteByte(alphabet[n%36])
		n /= 36
	}

	// Reverse the string
	s := result.String()
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}
