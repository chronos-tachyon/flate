package main

import (
	"time"

	"github.com/chronos-tachyon/flate"
	getopt "github.com/pborman/getopt/v2"
)

// type FormatFlag {{{

// FormatFlag implements getopt.Value for flate.Format.
type FormatFlag struct {
	Value flate.Format
}

// Set fulfills getopt.Value.
func (flag *FormatFlag) Set(str string, opt getopt.Option) error {
	return flag.Value.Parse(str)
}

// String fulfills getopt.Value.
func (flag FormatFlag) String() string {
	return flag.Value.String()
}

var _ getopt.Value = (*FormatFlag)(nil)

// }}}

// type StrategyFlag {{{

// StrategyFlag implements getopt.Value for flate.Strategy.
type StrategyFlag struct {
	Value flate.Strategy
}

// Set fulfills getopt.Value.
func (flag *StrategyFlag) Set(str string, opt getopt.Option) error {
	return flag.Value.Parse(str)
}

// String fulfills getopt.Value.
func (flag StrategyFlag) String() string {
	return flag.Value.String()
}

var _ getopt.Value = (*StrategyFlag)(nil)

// }}}

// type CompressLevelFlag {{{

// CompressLevelFlag implements getopt.Value for flate.CompressLevel.
type CompressLevelFlag struct {
	Value flate.CompressLevel
}

// Set fulfills getopt.Value.
func (flag *CompressLevelFlag) Set(str string, opt getopt.Option) error {
	return flag.Value.Parse(str)
}

// String fulfills getopt.Value.
func (flag CompressLevelFlag) String() string {
	return flag.Value.String()
}

var _ getopt.Value = (*CompressLevelFlag)(nil)

// }}}

// type MemoryLevelFlag {{{

// MemoryLevelFlag implements getopt.Value for flate.MemoryLevel.
type MemoryLevelFlag struct {
	Value flate.MemoryLevel
}

// Set fulfills getopt.Value.
func (flag *MemoryLevelFlag) Set(str string, opt getopt.Option) error {
	return flag.Value.Parse(str)
}

// String fulfills getopt.Value.
func (flag MemoryLevelFlag) String() string {
	return flag.Value.String()
}

var _ getopt.Value = (*MemoryLevelFlag)(nil)

// }}}

// type WindowBitsFlag {{{

// WindowBitsFlag implements getopt.Value for flate.WindowBits.
type WindowBitsFlag struct {
	Value flate.WindowBits
}

// Set fulfills getopt.Value.
func (flag *WindowBitsFlag) Set(str string, opt getopt.Option) error {
	return flag.Value.Parse(str)
}

// String fulfills getopt.Value.
func (flag WindowBitsFlag) String() string {
	return flag.Value.String()
}

var _ getopt.Value = (*WindowBitsFlag)(nil)

// }}}

// type TimeFlag {{{

// TimeFlag implements getopt.Value for time.Time.
type TimeFlag struct {
	Value time.Time
}

// Set fulfills getopt.Value.
func (flag *TimeFlag) Set(str string, opt getopt.Option) error {
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}
	flag.Value = t
	return nil
}

// String fulfills getopt.Value.
func (flag TimeFlag) String() string {
	return flag.Value.Format(time.RFC3339)
}

var _ getopt.Value = (*TimeFlag)(nil)

// }}}
