// Package benchmarks holds repository-wide benchmarks and the smoke
// tests that gate their fixtures. Bench targets, fixture loaders, and
// synthetic data generators live here so they don't bleed into the
// shipping API of the packages under test.
//
// Fixtures are checked-in JSON / binary captures under testdata/ so
// results are reproducible across machines and Go versions.
package benchmarks
