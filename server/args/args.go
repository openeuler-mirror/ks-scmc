package args

import (
	"net"
	"regexp"
	"unicode/utf8"
)

const (
	containerNamePattern = `^[a-zA-Z0-9][a-zA-Z0-9_.-]+$`
	imageNamePattern     = `^[a-z0-9]([a-z0-9_.-]*[a-z0-9])?$`
	imageVersionPattern  = `^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`
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

func IsValidNodeName(s string) bool {
	c := utf8.RuneCountInString(s)
	return c >= 1 && c <= 50
}

func IsValidNodeComment(s string) bool {
	c := utf8.RuneCountInString(s)
	return c <= 200
}

func IsValidContainerName(s string) bool {
	c := utf8.RuneCountInString(s)
	if c < 1 || c > 50 {
		return false
	}

	return regexp.MustCompile(containerNamePattern).MatchString(s)
}

func IsValidContainerDesc(s string) bool {
	c := utf8.RuneCountInString(s)
	return c <= 200
}

func IsValidImageName(s string) bool {
	c := utf8.RuneCountInString(s)
	if c < 1 || c > 50 {
		return false
	}

	return regexp.MustCompile(imageNamePattern).MatchString(s)
}

func IsValidImageVersion(s string) bool {
	c := utf8.RuneCountInString(s)
	if c < 1 || c > 20 {
		return false
	}

	return regexp.MustCompile(imageVersionPattern).MatchString(s)
}

func IsValidImageDesc(s string) bool {
	c := utf8.RuneCountInString(s)
	return c <= 200
}

func IsValidBackupDesc(s string) bool {
	return utf8.RuneCountInString(s) <= 200
}
