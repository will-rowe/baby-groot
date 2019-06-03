<div align="center">
					<object type="image/png" data="wasm/assets/css/images/baby-groot.svg" width="250">
						<img src="https://github.com/will-rowe/baby-groot/raw/master/baby-groot-logo.png" alt="logo" width="250">
					</object>
    <h3>BABY GROOT</h3>
</div>

***

## Overview

BABY GROOT is a dummer, smaller version of [GROOT](https://github.com/will-rowe/groot). This repo is essentially for the development of GROOT v2.0 and for trying out some new ideas in my spare time.

The main difference between BABY GROOT and GROOT is that we no longer use exact alignments to detect genes. The focus is on creating `approximately weighted variation graphs` from a sample, and then using these weighted graphs to predict the most likely haplotypes. The hope is that this new method will be less sensitive to sequencing error and variable read lengths, whilst still performing as well as GROOT v1.0

## The idea

* use sketching + LSH forest to approximately map reads to variation graphs
* use the mappings to create approximately weighted variation graphs (GFA format)
  * we know a read approximately maps between nodes X and Z on a graph (with approx. error)
  * we used k-mer decomposition to generate sketch
  * so nodes X through Z will have A k-mers, where A = length(X..Z) - k + 1 
  * we can distribute A k-mers across nodes X..Z
* use Markov Chain Monte Carlo to probabilistically determine the best paths through the weighted graphs
  * the best paths are equivalent to the haplotypes in our sample

## Changes and new features since the GROOT publication

The original [GROOT paper](https://academic.oup.com/bioinformatics/article/34/21/3601/4995843) was published in Bioinformatics last year. BABY GROOT has seen a few updates:

* replaced existing hash functions with ntHash
  * also, now only consider canonical k-mers
* added extra sketching algorithms for mapping reads to graph traversals
  * added bottom-k minhash algorithm, in addition to KMV minhash
  * added bloom filter to stop unique k-mers being added to sketches
* removed exact alignment step and BAM outputs
* replaced report subcommand with haplotype
* output weighted GFA as the primary output
* swapping serialisation to messagepack (json derivative)
* general code improvements and bug fixes
* added support for [Go modules](https://github.com/golang/go/wiki/Modules) and Go 1.12

## Still to come

* additional support for variable read lengths and long reads
  * I'm thinking that some sort of windowing strategy of the query sequences may work out better than containment, now that we're not doing exact alignment
* additional sketching algorithms to benchmark against KMV, Bottom-K, Bloom Filter
* add in phasing and paired-end information for informing path prediction during haplotype recovery
* improving node re-weighting scheme during MCMC path finding
* add in the option to augment graphs with new paths during mapping
* add a WASM ui
