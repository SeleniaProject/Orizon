# Orizon Fuzzing Corpus

This directory contains small seed corpora for fuzz targets.

- parser_corpus.txt: one input per line (raw or 0xHEX). Used by `--corpus`.
- lexer_corpus.txt: one input per line for lexer target.
- astbridge_corpus.txt: inputs that parse and are useful for AST bridge.

Notes:
- Lines starting with `0x` are treated as hex-encoded bytes by `orizon-fuzz`.
- Keep inputs small; the fuzzer mutates and expands them on the fly.


