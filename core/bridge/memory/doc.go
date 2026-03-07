// Package memory implements the Bridge memory subsystem.
//
// Layout overview:
//   - `memory_*`: shared L1/L2/L3 manager, query, storage, lifecycle and helpers.
//   - `anchor*`: structured anchor extraction and ranking signals.
//   - `graph_*`: graph sidecar models, storage, recall and rebuild flow.
//   - `decision_*`: decision memo/recipe sidecar capture, recall and distill flow.
//
// The package keeps a single public facade (`MemoryManager`) while organizing
// internal code by concern so query flow, persistence and sidecars can evolve
// independently without changing the external API surface.
package memory
