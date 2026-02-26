package evaluator

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
)

var hashModuleMeta = ModuleMeta{
	Description: "Cryptographic hash functions for checksums and non-security hashing",
	Exports: map[string]ExportMeta{
		"md5":    {Kind: "function", Arity: "1", Description: "MD5 hash (hex string)"},
		"sha1":   {Kind: "function", Arity: "1", Description: "SHA1 hash (hex string)"},
		"sha256": {Kind: "function", Arity: "1", Description: "SHA256 hash (hex string)"},
		"sha512": {Kind: "function", Arity: "1", Description: "SHA512 hash (hex string)"},
	},
}

func loadHashModule(env *Environment) Object {
	return &StdlibModuleDict{
		Meta: &hashModuleMeta,
		Exports: map[string]Object{
			"md5":    &Builtin{Fn: hashMD5},
			"sha1":   &Builtin{Fn: hashSHA1},
			"sha256": &Builtin{Fn: hashSHA256},
			"sha512": &Builtin{Fn: hashSHA512},
		},
	}
}

func hashMD5(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("hash.md5", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "hash.md5", "a string", args[0].Type())
	}
	hash := md5.Sum([]byte(str.Value))
	return &String{Value: hex.EncodeToString(hash[:])}
}

func hashSHA1(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("hash.sha1", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "hash.sha1", "a string", args[0].Type())
	}
	hash := sha1.Sum([]byte(str.Value))
	return &String{Value: hex.EncodeToString(hash[:])}
}

func hashSHA256(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("hash.sha256", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "hash.sha256", "a string", args[0].Type())
	}
	hash := sha256.Sum256([]byte(str.Value))
	return &String{Value: hex.EncodeToString(hash[:])}
}

func hashSHA512(args ...Object) Object {
	if len(args) != 1 {
		return newArityError("hash.sha512", len(args), 1)
	}
	str, ok := args[0].(*String)
	if !ok {
		return newTypeError("TYPE-0012", "hash.sha512", "a string", args[0].Type())
	}
	hash := sha512.Sum512([]byte(str.Value))
	return &String{Value: hex.EncodeToString(hash[:])}
}
