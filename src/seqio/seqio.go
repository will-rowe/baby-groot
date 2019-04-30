/*
	the seqio package contains custom types and methods for holding and processing sequence data
*/
package seqio

import (
	"errors"
	"unicode"

	"github.com/will-rowe/baby-groot/src/minhash"
)

const (
	ENCODING = 33 // fastq encoding used
)

var complementBases = []byte{
	'A': 'T',
	'T': 'A',
	'C': 'G',
	'G': 'C',
	'N': 'N',
}

/*
  the base type
*/
type Sequence struct {
	ID  []byte
	Seq []byte
}

/*
  struct to hold FASTQ data and seed locations for a single read
*/
type FASTQread struct {
	Sequence
	Misc       []byte
	Qual       []byte
	RC         bool
	Alignments []*Key
}

/*
  Key is a struct to project sketches onto graphs
*/
type Key struct {
	GraphID        int
	Node           uint64
	OffSet         int
	SubPath        []uint64 // GFA subpath in which this window is contained (assumes all paths are in the same orientation)
	Ref            int      // the first reference sequence this subpath was derived from (only used to differentiate duplicate sketches)
	Sketch         []uint64
	StringifiedKey string
	RC             bool
}

// method to check for ACTGN bases and to convert bases to upper case TODO: improve the efficiency of this...
func (self *Sequence) BaseCheck() error {
	for i, j := 0, len(self.Seq); i < j; i++ {
		switch base := unicode.ToUpper(rune(self.Seq[i])); base {
		case 'A':
			self.Seq[i] = byte(base)
		case 'C':
			self.Seq[i] = byte(base)
		case 'T':
			self.Seq[i] = byte(base)
		case 'G':
			self.Seq[i] = byte(base)
		case 'N':
			self.Seq[i] = byte(base)
		default:
			//return fmt.Errorf("non \"A\\C\\T\\G\\N\\-\" base (%v)", string(self.Seq[i]))
			self.Seq[i] = byte('N')
		}
	}
	return nil
}

// method to reverse complement a sequence
// if used on a fastq read, the quality scores will not be reversed, therefore it is assumed that the read has already been trimmed before using this reverse complement function
func (self *FASTQread) RevComplement() {
	for i, j := 0, len(self.Seq); i < j; i++ {
		self.Seq[i] = complementBases[self.Seq[i]]
	}
	for i, j := 0, len(self.Seq)-1; i <= j; i, j = i+1, j-1 {
		self.Seq[i], self.Seq[j] = self.Seq[j], self.Seq[i]
	}
	if self.RC == true {
		self.RC = false
	} else {
		self.RC = true
	}
}

// method to split sequence to k-mers + get minhash signature
func (self *Sequence) RunMinHash(k int, sigSize int) ([]uint64, error) {
	// create the MinHash
	minhash := minhash.NewMinHash(k, sigSize)
	// use the add method to initate rolling ntHash and populate MinHash
	err := minhash.Add(append(self.Seq))
	// return the signature and any error
	return minhash.Signature(), err
}

// method to quality trim the sequence held in a FASTQread
// the algorithm is based on bwa/cutadapt read quality trim functions: -1. for each index position, subtract qual cutoff from the quality score -2. sum these values across the read and trim at the index where the sum in minimal -3. return the high-quality region
func (self *FASTQread) QualTrim(minQual int) {
	start, qualSum, qualMax := 0, 0, 0
	end := len(self.Qual)
	for i, qual := range self.Qual {
		qualSum += minQual - (int(qual) - ENCODING)
		if qualSum < 0 {
			break
		}
		if qualSum > qualMax {
			qualMax = qualSum
			start = i + 1
		}
	}
	qualSum, qualMax = 0, 0
	for i, j := 0, len(self.Qual)-1; j >= i; j-- {
		qualSum += minQual - (int(self.Qual[j]) - ENCODING)
		if qualSum < 0 {
			break
		}
		if qualSum > qualMax {
			qualMax = qualSum
			end = j
		}
	}
	if start >= end {
		start, end = 0, 0
	}
	self.Seq = self.Seq[start:end]
	self.Qual = self.Qual[start:end]
}

/*
  function to generate new fastq read from 4 lines of a fastq
*/
func NewFASTQread(l1 []byte, l2 []byte, l3 []byte, l4 []byte) (FASTQread, error) {
	// check that it looks like a fastq read TODO: need more fastq checks
	if len(l2) != len(l4) {
		return FASTQread{}, errors.New("sequence and quality score lines are unequal lengths in fastq file")
	}
	if l1[0] != 64 {
		return FASTQread{}, errors.New("read ID in fastq file does not begin with @")
	}
	// create a FASTQread struct
	read := new(FASTQread)
	read.ID = l1
	read.Seq = l2
	read.Misc = l3
	read.Qual = l4
	return *read, nil
}
