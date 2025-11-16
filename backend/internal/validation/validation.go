package validation

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
)

var (
	ErrInvalidPath   = errors.New("invalid file path")
	ErrPathNotExists = errors.New("path does not exist")
	ErrInvalidAddr   = errors.New("invalid listen address")
	ErrEmptyString   = errors.New("value must not be empty")
	ErrOutOfRange    = errors.New("value out of range")
)

func ValidateFilePath(p string, mustExist bool) error {
	if p == "" { return ErrInvalidPath }
	if !filepath.IsAbs(p) {
		// Allow relative but normalize; disallow traversal outside working dir if needed
		p = filepath.Clean(p)
	}
	if mustExist {
		if _, err := os.Stat(p); err != nil {
			return fmt.Errorf("%w: %v", ErrPathNotExists, err)
		}
	}
	return nil
}

func ValidateAddr(addr string) error {
	if addr == "" { return ErrInvalidAddr }
	_, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil { return fmt.Errorf("%w: %v", ErrInvalidAddr, err) }
	return nil
}

func ValidateStringNonEmpty(s string) error {
	if s == "" { return ErrEmptyString }
	return nil
}

func ValidateRangeInt(v, min, max int) error {
	if v < min || v > max {
		return fmt.Errorf("%w: %d not in [%d,%d]", ErrOutOfRange, v, min, max)
	}
	return nil
}
