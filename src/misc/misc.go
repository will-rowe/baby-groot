// contains some misc helper functions etc. for GROOT
package misc

import (
	"encoding/binary"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/vmihailenco/msgpack.v2"
)

// HASH_SIZE is set to 2/4/8 for 16bit/32bit/64bit hash values
const HASH_SIZE = 8

// a function to throw error to the log and exit the program
func ErrorCheck(msg error) {
	if msg != nil {
		log.Fatal("encountered error: ", msg)
	}
}

// a function to check for required flags
func CheckRequiredFlags(flags *pflag.FlagSet) error {
	requiredError := false
	flagName := ""

	flags.VisitAll(func(flag *pflag.Flag) {
		requiredAnnotation := flag.Annotations[cobra.BashCompOneRequiredFlag]
		if len(requiredAnnotation) == 0 {
			return
		}

		flagRequired := requiredAnnotation[0] == "true"

		if flagRequired && !flag.Changed {
			requiredError = true
			flagName = flag.Name
		}
	})

	if requiredError {
		return errors.New("Required flag `" + flagName + "` has not been set")
	}

	return nil
}

// StartLogging is a function to start the log...
func StartLogging(logFile string) *os.File {
	logPath := strings.Split(logFile, "/")
	joinedLogPath := strings.Join(logPath[:len(logPath)-1], "/")
	if len(logPath) > 1 {
		if _, err := os.Stat(joinedLogPath); os.IsNotExist(err) {
			if err := os.MkdirAll(joinedLogPath, 0700); err != nil {
				log.Fatal("can't create specified directory for log")
			}
		}
	}
	logFH, err := os.OpenFile(logFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}
	return logFH
}

// Stringify is a function to print an array of unsigned integers as a string - taken from https://github.com/ekzhu/minhash-lsh TODO: benchmark with other stringify options
func Stringify(sig []uint64) string {
	s := make([]byte, HASH_SIZE*len(sig))
	buf := make([]byte, 8)
	for i, v := range sig {
		binary.LittleEndian.PutUint64(buf, v)
		copy(s[i*HASH_SIZE:(i+1)*HASH_SIZE], buf[:HASH_SIZE])
	}
	return string(s)
}

// Uint64SliceEqual returns true if two uint64[] are identical
func Uint64SliceEqual(a []uint64, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}

/*
  A type to save the command information
*/
type IndexInfo struct {
	Version    string
	Ksize      int
	SigSize    int
	KMVsketch  bool
	JSthresh   float64
	WindowSize int
}

// method to dump the info to file
func (self *IndexInfo) Dump(path string) error {
	b, err := msgpack.Marshal(self)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, b, 0644)
}

// method to load info from file
func (self *IndexInfo) Load(path string) error {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return msgpack.Unmarshal(b, self)
}
