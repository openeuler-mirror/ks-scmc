package args

import (
	"math"
	"net"
	"regexp"
	"unicode/utf8"
)

const (
	PatternContainerName = `^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`
	PatternImageName     = `^[a-z0-9]([a-z0-9_.-]*[a-z0-9])?$`
	PatternImageVersion  = `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`
)

type StringArgument struct {
	length  func(string) bool
	pattern string
}

type Float64Argument struct {
	valid func(float64) bool
}

type Int64Argument struct {
	valid func(int64) bool
}
type Int32Argument struct {
	valid func(int32) bool
}

func (a *StringArgument) Check(s string) bool {
	if !a.length(s) {
		return false
	}

	if a.pattern == "" {
		return true
	}
	return regexp.MustCompile(a.pattern).MatchString(s)
}

func (r *Float64Argument) Check(in ...float64) bool {
	for _, i := range in {
		if r.valid(i) {
			continue
		}
		return false
	}
	return true
}

func (r *Int64Argument) Check(in ...int64) bool {
	for _, i := range in {
		if r.valid(i) {
			continue
		}
		return false
	}
	return true
}
func (r *Int32Argument) Check(in ...int32) bool {
	for _, i := range in {
		if r.valid(i) {
			continue
		}
		return false
	}
	return true
}

func lengthChecker(min, max uint32) func(string) bool {
	return func(s string) bool {
		c := uint32(utf8.RuneCountInString(s))
		return c >= min && c <= max
	}
}

var (
	NodeAddr = &StringArgument{
		length: lengthChecker(1, 100),
	}
	NodeName = &StringArgument{
		length: lengthChecker(1, 50),
	}
	NodeComment = &StringArgument{
		length: lengthChecker(0, 200),
	}
	ContainerName = &StringArgument{
		length:  lengthChecker(1, 50),
		pattern: PatternContainerName,
	}
	ContainerDesc = NodeComment
	ImageName     = &StringArgument{
		length:  lengthChecker(1, 50),
		pattern: PatternImageName,
	}
	ImageVersion = &StringArgument{
		length:  lengthChecker(1, 20),
		pattern: PatternImageVersion,
	}
	ImageDesc  = NodeComment
	BackupDesc = NodeComment
	NicName    = &StringArgument{
		length: lengthChecker(1, 200),
	}
	Float64Resource = Float64Argument{
		valid: func(f float64) bool {
			return f == math.Max(f, 0)
		},
	}
	Int64Resource = Int64Argument{
		valid: func(i int64) bool {
			return i > 0
		},
	}
)

func IsValidNodeAddr(s string) bool {
	c := utf8.RuneCountInString(s)
	if c < 1 || c > 200 {
		return false
	}

	ip := net.ParseIP(s)
	isDomain := regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-.]{0,253}[a-zA-Z0-9]$").MatchString(s)
	return ip != nil || isDomain
}
