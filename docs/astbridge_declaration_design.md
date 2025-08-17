# AST Bridge Declaration Node Design (struct/enum/trait/impl/import/export)

This note documents the minimal-intrusion design for bridging between the parser AST (`internal/parser`) and the core AST (`internal/ast`) regarding declaration nodes.

Goals:
- Keep the core AST stable and compact for downstream tools (optimizers, visitors).
- Avoid proliferating declaration node variants unless required by downstream passes.
- Provide round-trip conversion for declarations that are consumed by tooling already using core AST.

Decision:
- import/export: Added as first-class nodes in core AST and fully supported by the bridge (parser <-> core) including alias and basic lists. This matches project needs (tooling, packaging) and is already tested.
- type alias/newtype: Represented in core AST using a single `TypeDeclaration` with flag `IsAlias` (true for alias, false for newtype). The bridge maps:
  - parser `TypeAliasDeclaration` -> core `TypeDeclaration{IsAlias: true}`
  - parser `NewtypeDeclaration` -> core `TypeDeclaration{IsAlias: false}`
  Round-trip is implemented and tested.
- struct/enum/trait/impl: These remain in the parser AST and are consumed directly by the parser-side HIR transformer (`internal/parser/ast_to_hir.go`). Core AST does not define these nodes to avoid duplication and churn across layers. The bridge intentionally does not convert these declarations for now. If core-level tools later require them, we will add minimal core AST nodes and extend the bridge with round-trip conversions.

Rationale:
- The HIR pipeline already transforms parser declarations (struct/enum/trait/impl) without involving the core AST, so introducing parallel node kinds in core AST would add maintenance cost with limited benefit today.
- import/export and type declarations are needed by formatting, packaging, and codegen tools that work on the core AST; hence, they are modeled in core AST and bridged.

Implications:
- Using `astbridge.FromParserProgram` on sources containing struct/enum/trait/impl will return an error for those declarations. This is by design until core AST introduces these nodes. Existing tests do not rely on such conversions.
- Downstream consumers that require information about these declarations should use the parser AST to HIR path.

Next steps (if requirements evolve):
- Introduce minimal `StructDecl`, `EnumDecl`, `TraitDecl`, `ImplDecl` in core AST with only the data needed by the target tools.
- Extend `internal/astbridge/convert.go` with round-trip conversions and add unit tests.
