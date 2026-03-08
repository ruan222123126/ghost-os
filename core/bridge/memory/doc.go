// Package memory implements the Bridge memory subsystem.
//
// Directory contract:
//   - Root package (`core/bridge/memory`) is the only public façade: `MemoryManager`,
//     exported DTO/config types, assembly, and façade-level regression tests live here.
//   - `memory_*`, `graph_service.go`, and `decision_service.go` keep the entry points,
//     query wiring, lifecycle, and thin adapters that define the package boundary.
//   - Concrete graph and decision implementations live in `internal/graph` and
//     `internal/decision`; keep new logic there instead of rebuilding flat copies here.
//   - Do not keep duplicate root-level implementation snapshots once code has moved into
//     `internal/*`; delete the old copy instead of letting two versions drift.
//   - Newcomer map: start with `memory_manager*.go` / `*_service.go`, then read
//     `internal/*` for implementation details, then `*_test.go` for regression guards.
package memory
