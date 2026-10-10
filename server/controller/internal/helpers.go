package internal

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

func isValidNodeAddr(s string) bool {
	c := utf8.RuneCountInString(s)
	if c < 1 || c > 200 {
		return false
	}

	ip := net.ParseIP(s)
	isDomain := regexp.MustCompile("^[a-zA-Z][a-zA-Z0-9-.]{0,253}[a-zA-Z0-9]$").MatchString(s)
	return ip != nil || isDomain
}

func isValidNodeName(s string) bool {
	c := utf8.RuneCountInString(s)
	return c >= 1 && c <= 50
}

func isValidNodeComment(s string) bool {
	c := utf8.RuneCountInString(s)
	return c <= 200
}

func isValidContainerName(s string) bool {
	c := utf8.RuneCountInString(s)
	if c < 1 || c > 50 {
		return false
	}

	return regexp.MustCompile(containerNamePattern).MatchString(s)
}

func isValidContainerDesc(s string) bool {
	c := utf8.RuneCountInString(s)
	return c <= 200
}

func isValidImageName(s string) bool {
	c := utf8.RuneCountInString(s)
	if c < 1 || c > 50 {
		return false
	}

	return regexp.MustCompile(imageNamePattern).MatchString(s)
}

func isValidImageVersion(s string) bool {
	c := utf8.RuneCountInString(s)
	if c < 1 || c > 20 {
		return false
	}

	return regexp.MustCompile(imageVersionPattern).MatchString(s)
}

func isValidImageDesc(s string) bool {
	c := utf8.RuneCountInString(s)
	return c <= 200
}

func isValidBackupDesc(s string) bool {
	return utf8.RuneCountInString(s) <= 200
}
