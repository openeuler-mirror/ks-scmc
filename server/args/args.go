package args

import (
	"net"
	"regexp"
	"unicode/utf8"
)

const (
	PatternContainerName = `^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`
	PatternImageName     = `^[a-z0-9]([a-z0-9_.-]*[a-z0-9])?$`
	PatternImageVersion  = `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`
)

type Argument struct {
	length  func(string) bool
	pattern string
}

func (a *Argument) Check(s string) bool {
	if !a.length(s) {
		return false
	}

	if a.pattern == "" {
		return true
	}
	return regexp.MustCompile(a.pattern).MatchString(s)
}

func lengthChecker(min, max uint32) func(string) bool {
	return func(s string) bool {
		c := uint32(utf8.RuneCountInString(s))
		return c >= min && c <= max
	}
}

var (
	NodeAddr = &Argument{
		length: lengthChecker(1, 100),
	}
	NodeName = &Argument{
		length: lengthChecker(1, 50),
	}
	NodeComment = &Argument{
		length: lengthChecker(0, 200),
	}
	ContainerName = &Argument{
		length:  lengthChecker(1, 50),
		pattern: PatternContainerName,
	}
	ContainerDesc = NodeComment
	ImageName     = &Argument{
		length:  lengthChecker(1, 50),
		pattern: PatternImageName,
	}
	ImageVersion = &Argument{
		length:  lengthChecker(1, 20),
		pattern: PatternImageVersion,
	}
	ImageDesc  = NodeComment
	BackupDesc = NodeComment
	NicName    = &Argument{
		length: lengthChecker(1, 200),
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
