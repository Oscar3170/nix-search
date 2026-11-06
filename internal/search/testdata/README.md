# Testdata provenance

The JSON files here were generated once from the original Python
implementation of this tool (`nix-search.py`), which used the same backend
queries and the same output formatting. They are committed as a regression
baseline: the Go port must keep producing byte-identical request bodies and
rendered output.

- `query/*.json` — Elasticsearch `_search` bodies, one per subcommand, match
  mode, sort order and query shape (including multi-word queries).

  Rendered from `build_body(query, source, mode, sort, size)`.

- `responses/*.json` — `_search` responses used by the formatting tests.

  `pkgs`, `options`, `home_manager` and `empty` are unmodified responses from
  the live backend. `pkgs_edge` and `options_edge` are hand-written documents
  covering cases the live results did not exercise: empty program lists, a main
  program that needs reordering, multi-line descriptions, HTML entities, an
  option without a type or default (trailing padding), and an over-long default
  that gets truncated.

- `output/<fixture>_<format>.txt` — expected output for each fixture and
  `--output` style, produced by the Python `print_results`.

If the backend changes the wire format, regenerate the fixtures rather than
editing the golden output by hand.
