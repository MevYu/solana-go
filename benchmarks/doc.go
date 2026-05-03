// Package benchmarks is the home of Phase A performance
// measurements. It is kept out of the root solana package so that
// benchmark-only fixtures, target structs, and helper generators do
// not bleed into the shipping API.
//
// Every benchmark in this package reads from checked-in fixtures
// under testdata/ so that results are reproducible across machines
// and Go versions. The current fixtures are hand-crafted
// shape-accurate synthetic samples; replacing them with real
// mainnet captures is a Phase A follow-up.
package benchmarks
