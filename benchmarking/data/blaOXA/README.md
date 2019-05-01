# Recovering a blaOXA allele

This example shows how we can recover a blaOXA allele - blaOXA-90.

This allele clusters with 81 other variants of blaOXA (90% clustering identity).

I'll put this example in notebook form later, for now you will need to build groot and install BBMap (for simulating reads)

***

## using the KMV MinHash algorithm

* simulate reads with BBMap, one set with errors, the other without

```bash
randomreads.sh ref=blaOXA-90.fna out=blaOXA-90-reads.1000.errors.fq length=100 reads=1000 adderrors=t
randomreads.sh ref=blaOXA-90.fna out=blaOXA-90-reads.1000.fq length=100 reads=1000 adderrors=f
```

* create and index the graphs (using KMV MinHash + LSH Forest)

```bash
time ./baby-groot index -i arg-annot.90 -o test-index -k 31 -s 42 -j 0.97 --kmvSketch -p 4
```

* align the perfect reads and call the haplotype

```bash
time ./baby-groot align -i test-index -f blaOXA-90-reads.1000.fq -o test-graphs --minBaseCov 1.0 -p 4
time ./baby-groot haplotype -i test-index -g test-graphs -o test-haplotype -b 100 -z 0.97 -s 0.005 -p 4
```

> 100% of reads align, with 91% of reads mapping multiple times

> the reads only map to one graph (the blaOXA graph!)

> the graph is approximately weighted using the k-mer content derived from projected sketches

> the correct haplotype is then called using the approximately weighted variation graph

* now try aligning the reads with errors and call haplotype

```bash
time ./baby-groot align -i test-index -f blaOXA-90-reads.1000.errors.fq -o test-graphs --minBaseCov 1.0 -p 4
time ./baby-groot haplotype -i test-index -g test-graphs -o test-haplotype -b 100 -z 0.97 -s 0.005 -p 4
```

> only 80% of reads align this time, but still to the right graph

> the correct haplotype is still called

* now let's try with bottom-k sketching and bloom filter

```bash
time ./baby-groot index -i arg-annot.90 -o test-index -k 31 -s 17 -j 0.95 -p 4
time ./baby-groot align -i test-index -f blaOXA-90-reads.1000.errors.fq -o test-graphs --minBaseCov 0.9 --minKmerCov 2 --bloomFilter
time ./baby-groot haplotype -i test-index -g test-graphs -o test-haplotype -b 100 -z 1.0 -s 0.005 -p 4
```