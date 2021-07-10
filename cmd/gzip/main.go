package main

import (
	"fmt"
	"io"
	"io/ioutil"
	stdlog "log"
	"os"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/chronos-tachyon/flate"
	getopt "github.com/pborman/getopt/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	flagVersion   = false
	flagDebug     = false
	flagTrace     = false
	flagLogStderr = false

	flagStdout     = false
	flagDecompress = false
	flagForce      = false
	flagKeep       = false

	flag0 = false
	flag1 = false
	flag2 = false
	flag3 = false
	flag4 = false
	flag5 = false
	flag6 = false
	flag7 = false
	flag8 = false
	flag9 = false

	flagFormat       = FormatFlag{flate.DefaultFormat}
	flagStrategy     = StrategyFlag{flate.DefaultStrategy}
	flagCLevel       = CompressLevelFlag{flate.DefaultCompression}
	flagMLevel       = MemoryLevelFlag{flate.DefaultMemory}
	flagWBits        = WindowBitsFlag{flate.DefaultWindowBits}
	flagDict         = ""
	flagFileName     = ""
	flagComment      = ""
	flagLastModified = TimeFlag{time.Time{}}

	flagCPUProfile = ""
	flagMemProfile = ""
)

func init() {
	getopt.SetParameters("[<input>]")

	getopt.FlagLong(&flagVersion, "version", 'V', "print version and exit")

	getopt.FlagLong(&flagDebug, "verbose", 'v', "enable debug logging")
	getopt.FlagLong(&flagTrace, "debug", 'D', "enable debug and trace logging")
	getopt.FlagLong(&flagLogStderr, "log-stderr", 'L', "log JSON to stderr")

	getopt.FlagLong(&flagCPUProfile, "cpu-profile", 0, "CPU profile output file")
	getopt.FlagLong(&flagMemProfile, "mem-profile", 0, "memory profile output file")

	getopt.FlagLong(&flagFormat, "format", 'F', "file format; one of auto, gzip, zlib, or raw")
	getopt.FlagLong(&flagStrategy, "strategy", 'S', "strategy; one of default, huffman-only, huffman-rle, or fixed")
	getopt.FlagLong(&flagCLevel, "compress-level", 'C', "compression level; one of default, 0, 1, 2, 3, 4, 5, 6, 7, 8, or 9").SetGroup("clevel")
	getopt.FlagLong(&flagMLevel, "memory-level", 'M', "memory level; one of default, 1, 2, 3, 4, 5, 6, 7, 8, or 9")
	getopt.FlagLong(&flagWBits, "window-size-bits", 'W', "base-2 logarithm of window size; one of default, 8, 9, 10, 11, 12, 13, 14, or 15")
	getopt.FlagLong(&flagDict, "dictionary", 0, "contents of pre-set dictionary, or @filename")
	getopt.FlagLong(&flagFileName, "filename", 0, "filename to store in gzip header")
	getopt.FlagLong(&flagComment, "comment", 0, "comment to store in gzip header")
	getopt.FlagLong(&flagLastModified, "last-modified", 0, "last-modified time to store in gzip header")

	getopt.FlagLong(&flagStdout, "stdout", 'c', "write on standard output, keep original files unchanged")
	getopt.FlagLong(&flagDecompress, "decompress", 'd', "decompress")
	getopt.FlagLong(&flagForce, "force", 'f', "force overwrite of output file and compress links")
	getopt.FlagLong(&flagKeep, "keep", 'k', "keep (don't delete) input files")

	getopt.Flag(&flag0, '0', "don't compress").SetGroup("clevel")
	getopt.Flag(&flag1, '1', "fastest compression").SetGroup("clevel")
	getopt.Flag(&flag2, '2', "fast compression").SetGroup("clevel")
	getopt.Flag(&flag3, '3', "fast compression").SetGroup("clevel")
	getopt.Flag(&flag4, '4', "fast compression").SetGroup("clevel")
	getopt.Flag(&flag5, '5', "fast compression").SetGroup("clevel")
	getopt.Flag(&flag6, '6', "balanced compression").SetGroup("clevel")
	getopt.Flag(&flag7, '7', "good compression").SetGroup("clevel")
	getopt.Flag(&flag8, '8', "good compression").SetGroup("clevel")
	getopt.Flag(&flag9, '9', "best compression").SetGroup("clevel")
}

func main() {
	getopt.Parse()

	if flagVersion {
		fmt.Println(strings.TrimSpace(version))
		os.Exit(0)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.DurationFieldUnit = time.Second
	zerolog.DurationFieldInteger = false
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if flagDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if flagTrace {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}

	switch {
	case flagLogStderr:
		// do nothing

	default:
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	stdlog.SetFlags(0)
	stdlog.SetOutput(log.Logger)

	switch {
	case flag0:
		flagCLevel.Value = 0
	case flag1:
		flagCLevel.Value = 1
	case flag2:
		flagCLevel.Value = 2
	case flag3:
		flagCLevel.Value = 3
	case flag4:
		flagCLevel.Value = 4
	case flag5:
		flagCLevel.Value = 5
	case flag6:
		flagCLevel.Value = 6
	case flag7:
		flagCLevel.Value = 7
	case flag8:
		flagCLevel.Value = 8
	case flag9:
		flagCLevel.Value = 9
	}

	if getopt.NArgs() == 0 {
		flagStdout = true
	}

	var dict []byte
	if flagDict != "" {
		if flagDict[0] == '@' {
			raw, err := ioutil.ReadFile(flagDict[1:])
			if err != nil {
				log.Logger.Fatal().
					Str("filename", flagDict[1:]).
					Msg("ioutil.ReadFile failed")
			}
			dict = raw
		} else {
			dict = []byte(flagDict)
		}
	}

	if flagFormat.Value == flate.DefaultFormat && !flagDecompress {
		flagFormat.Value = flate.GZIPFormat
	}

	if flagCPUProfile != "" {
		f, err := os.OpenFile(flagCPUProfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			log.Logger.Fatal().
				Str("filename", flagCPUProfile).
				Err(err).
				Msg("os.OpenFile(O_WRONLY|O_CREATE|O_TRUNC) failed")
		}

		defer func() {
			err := f.Close()
			if err != nil {
				log.Logger.Error().
					Str("filename", flagCPUProfile).
					Err(err).
					Msg("failed to Close CPU profiling output file")
			}
		}()

		err = pprof.StartCPUProfile(f)
		if err != nil {
			log.Logger.Fatal().
				Err(err).
				Msg("pprof.StartCPUProfile failed")
		}

		defer pprof.StopCPUProfile()
	}

	if flagStdout {
		doCompressOrDecompress(os.Stdout, os.Stdin, dict)
	} else {
		log.Logger.Fatal().
			Msg("flag combination is not implemented")
	}

	if flagMemProfile != "" {
		f, err := os.OpenFile(flagMemProfile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			log.Logger.Fatal().
				Str("filename", flagMemProfile).
				Err(err).
				Msg("failed to Open memory profiling output file")
		}
		err = pprof.Lookup("allocs").WriteTo(f, 0)
		if err != nil {
			_ = f.Close()
			log.Logger.Fatal().
				Str("filename", flagMemProfile).
				Err(err).
				Msg("failed to Write memory profile to output file")
		}
		err = f.Close()
		if err != nil {
			log.Logger.Fatal().
				Str("filename", flagMemProfile).
				Err(err).
				Msg("failed to Close memory profile output file")
		}
	}
}

func doCompressOrDecompress(w io.Writer, r io.Reader, dict []byte) {
	opts := make([]flate.Option, 6, 7)
	opts[0] = flate.WithTracers(flate.Log(log.Logger))
	opts[1] = flate.WithFormat(flagFormat.Value)
	opts[2] = flate.WithStrategy(flagStrategy.Value)
	opts[3] = flate.WithCompressLevel(flagCLevel.Value)
	opts[4] = flate.WithMemoryLevel(flagMLevel.Value)
	opts[5] = flate.WithWindowBits(flagWBits.Value)
	if dict != nil {
		opts = append(opts, flate.WithDictionary(dict))
	}

	if flagDecompress {
		fr := flate.NewReader(r, opts...)

		nn, err := io.Copy(w, fr)
		if err != nil {
			log.Logger.Fatal().
				Int64("nn", nn).
				Err(err).
				Msg("io.Copy failed")
		}

		err = fr.Close()
		if err != nil {
			log.Logger.Fatal().
				Err(err).
				Msg("flate.Reader.Close failed")
		}
	} else {
		fw := flate.NewWriter(w, opts...)

		err := fw.SetHeader(flate.Header{
			FileName:     flagFileName,
			Comment:      flagComment,
			LastModified: flagLastModified.Value,
		})
		if err != nil {
			log.Logger.Fatal().
				Err(err).
				Msg("flate.Writer.SetHeader failed")
		}

		nn, err := io.Copy(fw, r)
		if err != nil {
			log.Logger.Fatal().
				Int64("nn", nn).
				Err(err).
				Msg("io.Copy failed")
		}

		err = fw.Flush(flate.FinishFlush)
		if err != nil {
			log.Logger.Fatal().
				Err(err).
				Msg("flate.Writer.Flush failed")
		}

		err = fw.Close()
		if err != nil {
			log.Logger.Fatal().
				Err(err).
				Msg("flate.Writer.Close failed")
		}
	}
}
