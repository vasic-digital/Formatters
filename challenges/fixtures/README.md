# Bilingual Challenge fixtures (round 255)

Five fixtures across two locales (English + Serbian Latin) feeding the
`challenges/runner` exerciser:

| File                  | Language | Locale | Role in runner                       |
|-----------------------|----------|--------|--------------------------------------|
| `go_hello_en.go`      | go       | en     | check2 byte-exact round-trip         |
| `go_hello_sr.go`      | go       | sr     | check2 Serbian round-trip            |
| `python_hello_en.py`  | python   | en     | check2 multi-language fan-out        |
| `python_hello_sr.py`  | python   | sr     | check2 Serbian Python round-trip     |
| `sql_query_en.sql`    | sql      | en     | check2 SQL language path             |

The Go runner embeds string copies of each file's expected content for
deterministic comparisons; these on-disk versions exist so external
consumers can reproduce the inputs without rebuilding the runner.

A fixture mismatch between disk and runner is a CONST-035 regression
(check2 evidence becomes meaningless if the disk fixture has drifted).
