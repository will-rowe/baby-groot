// Copyright © 2019 Will Rowe <w.p.m.rowe@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// the command line arguments
var (
	indexDir  *string // directory for groot to write/read index files
	logFile   *string // name to use for log file
	proc      *int    // number of processors to use
	profiling *bool   // create profile for go pprof
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "groot",
	Short: "annotate Antibiotic Resistance Genes (ARGs) from metagenomes using variation graphs",
	Long: `
#####################################################################################
		GROOT: Graphing Resistance genes Out Of meTagenomes
#####################################################################################

 GROOT is a tool to type Antibiotic Resistance Genes (ARGs) in metagenomic samples.

 It combines variation graph representation of gene sets with an LSH indexing scheme
 to allow for fast classification of metagenomic reads. Subsequent hierarchical local
 alignment of classified reads against graph traversals facilitates accurate reconstruction
 of full-length gene sequences using a simple scoring scheme.

 GROOT can output an ARG alignment file, as well as a typing report and the variation graphs
 with aligned reads.`,
}

// Execute is a function to add all child commands to the root command and sets flags appropriately
func Execute() {
	// launch subcommand
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// init the command line arguments
func init() {
	indexDir = RootCmd.PersistentFlags().StringP("indexDir", "i", "", "directory for to write/read the GROOT index files - required")
	logFile = RootCmd.PersistentFlags().String("log", "", "filename for log file, STDOUT used by default")
	proc = RootCmd.PersistentFlags().IntP("processors", "p", 1, "number of processors to use")
	profiling = RootCmd.PersistentFlags().Bool("profiling", false, "create the files needed to profile GROOT using the go tool pprof")
	RootCmd.MarkPersistentFlagRequired("indexDir")
}
