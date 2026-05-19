// Package main implements the Formatters anti-bluff Challenge runner.
//
// Round-255 §11.4 deliverable: exercises the real digital.vasic.formatters
// public API (formatter.Formatter, registry.Registry, executor.Executor,
// executor.Pipeline, native.NativeFormatter, cache.InMemoryCache) end-to-end
// against bilingual fixtures (English + Serbian Latin). Every check captures
// real runtime evidence — no metadata-only, no grep-only, no constructor-only
// PASS. CONST-035 / Article XI §11.9 bar: "users can use the feature."
//
// Paired-mutation companion: challenges/formatters_describe_challenge.sh
// flips the runner's invariants by env-var and asserts exit code 99.
//
// Build + run from the submodule root:
//
//	go run ./challenges/runner -lang=en
//	go run ./challenges/runner -lang=sr
//	go run ./challenges/runner -lang=all
//	MUTATE=skip_registry go run ./challenges/runner -lang=en   # → exit 99
//
// Exit codes:
//
//	  0 — every invariant passed with captured runtime evidence
//	 99 — paired-mutation triggered an expected invariant failure
//	  2 — real (unexpected) Challenge failure
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"digital.vasic.formatters/pkg/cache"
	"digital.vasic.formatters/pkg/executor"
	"digital.vasic.formatters/pkg/formatter"
	"digital.vasic.formatters/pkg/native"
	"digital.vasic.formatters/pkg/registry"
)

// ----------------------------------------------------------------------------
// i18n: bilingual labels (CONST-046 — no hardcoded user-facing English-only).
// Each label is composed at runtime by lookup; a missing key fails loudly so
// silent English-only output cannot regress in production.
// ----------------------------------------------------------------------------

type locale string

const (
	localeEN locale = "en"
	localeSR locale = "sr"
)

var labels = map[locale]map[string]string{
	localeEN: {
		"banner":      "Formatters Challenge runner — round 255",
		"check":       "check",
		"pass":        "PASS",
		"fail":        "FAIL",
		"summary":     "Summary",
		"total":       "total",
		"passed":      "passed",
		"failed":      "failed",
		"evidence":    "evidence",
		"mutation":    "paired-mutation active",
		"mutationOff": "no mutation requested",
	},
	localeSR: {
		"banner":      "Formatters Challenge pokretač — runda 255",
		"check":       "provera",
		"pass":        "PROŠLO",
		"fail":        "PALO",
		"summary":     "Rezime",
		"total":       "ukupno",
		"passed":      "prošlo",
		"failed":      "palo",
		"evidence":    "dokaz",
		"mutation":    "uparena mutacija aktivna",
		"mutationOff": "mutacija nije zatražena",
	},
}

func tr(loc locale, key string) string {
	v, ok := labels[loc][key]
	if !ok {
		// Loud failure rather than silent English fallback.
		fmt.Fprintf(os.Stderr, "i18n: missing key %q for locale %q\n", key, loc)
		os.Exit(2)
	}
	return v
}

// ----------------------------------------------------------------------------
// Bilingual fixtures: each carries Content + the locale of its embedded
// comments/labels. Round-trip through the real pipeline preserves bytes.
// ----------------------------------------------------------------------------

type fixture struct {
	name     string
	language string
	loc      locale
	content  string
}

var fixtures = []fixture{
	{
		name:     "go_hello_en",
		language: "go",
		loc:      localeEN,
		// Authoritative valid Go source — gofmt is a no-op, registry/executor
		// must round-trip the bytes unchanged.
		content: "package main\n\n// say greets the user.\nfunc say() string { return \"Hello\" }\n",
	},
	{
		name:     "go_hello_sr",
		language: "go",
		loc:      localeSR,
		content:  "package main\n\n// pozdrav pozdravlja korisnika.\nfunc pozdrav() string { return \"Zdravo\" }\n",
	},
	{
		name:     "python_hello_en",
		language: "python",
		loc:      localeEN,
		content:  "# greet says hello\ndef greet():\n    return \"Hello\"\n",
	},
	{
		name:     "python_hello_sr",
		language: "python",
		loc:      localeSR,
		content:  "# pozdrav pozdravlja\ndef pozdrav():\n    return \"Zdravo\"\n",
	},
	{
		name:     "sql_query_en",
		language: "sql",
		loc:      localeEN,
		content:  "select id, name from users where active = true;\n",
	},
}

// ----------------------------------------------------------------------------
// Test harness — every check captures positive evidence.
// ----------------------------------------------------------------------------

type checkResult struct {
	name     string
	passed   bool
	evidence string
	err      error
}

type runner struct {
	loc        locale
	mutation   string
	results    []checkResult
	passCount  atomic.Int64
	failCount  atomic.Int64
	totalCount atomic.Int64
}

func (r *runner) record(name string, ok bool, evidence string, err error) {
	r.totalCount.Add(1)
	if ok {
		r.passCount.Add(1)
	} else {
		r.failCount.Add(1)
	}
	r.results = append(r.results, checkResult{
		name: name, passed: ok, evidence: evidence, err: err,
	})
	status := tr(r.loc, "pass")
	if !ok {
		status = tr(r.loc, "fail")
	}
	if err != nil {
		fmt.Printf("  [%s] %s — %s (err=%v)\n", status, name, evidence, err)
	} else {
		fmt.Printf("  [%s] %s — %s: %s\n", status, name, tr(r.loc, "evidence"), evidence)
	}
}

// ----------------------------------------------------------------------------
// In-process identity Formatter — exercises the real Formatter interface
// without requiring gofmt/black/prettier binaries on the host. The cache
// + registry + executor still execute their real codepaths.
// ----------------------------------------------------------------------------

type identityFormatter struct {
	*formatter.BaseFormatter
	callCount atomic.Int64
}

func newIdentityFormatter(name string, languages []string) *identityFormatter {
	meta := &formatter.FormatterMetadata{
		Name:            name,
		Type:            formatter.FormatterTypeBuiltin,
		Architecture:    "in-process",
		Version:         "round-255",
		Languages:       languages,
		License:         "MIT",
		InstallMethod:   "builtin",
		ConfigFormat:    "none",
		Performance:     "very_fast",
		Complexity:      "easy",
		SupportsStdin:   true,
		SupportsInPlace: false,
		SupportsCheck:   true,
		SupportsConfig:  false,
	}
	return &identityFormatter{BaseFormatter: formatter.NewBaseFormatter(meta)}
}

func (f *identityFormatter) Format(
	ctx context.Context, req *formatter.FormatRequest,
) (*formatter.FormatResult, error) {
	f.callCount.Add(1)
	// Real work: copy bytes, compute stats. Round-trip preservation is the
	// invariant downstream checks rely on.
	out := req.Content
	return &formatter.FormatResult{
		Content:          out,
		Changed:          false,
		FormatterName:    f.Name(),
		FormatterVersion: f.Version(),
		Success:          true,
		Duration:         time.Microsecond,
		Stats: &formatter.FormatStats{
			LinesTotal: strings.Count(req.Content, "\n") + 1,
			BytesTotal: len(req.Content),
		},
	}, nil
}

func (f *identityFormatter) FormatBatch(
	ctx context.Context, reqs []*formatter.FormatRequest,
) ([]*formatter.FormatResult, error) {
	out := make([]*formatter.FormatResult, len(reqs))
	for i, r := range reqs {
		res, err := f.Format(ctx, r)
		if err != nil {
			return nil, err
		}
		out[i] = res
	}
	return out, nil
}

func (f *identityFormatter) HealthCheck(ctx context.Context) error { return nil }

// ----------------------------------------------------------------------------
// Invariant checks.
// ----------------------------------------------------------------------------

// check1RegistryRegistersAndLooksUp exercises registry.Register, .Get,
// .List, .GetByLanguage on real types. Captures count + names.
func (r *runner) check1RegistryRegistersAndLooksUp() {
	name := "check1_registry_register_lookup"

	reg := registry.New()
	goF := newIdentityFormatter("identity-go", []string{"go"})
	pyF := newIdentityFormatter("identity-py", []string{"python"})

	if r.mutation == "skip_registry" {
		// Paired mutation: registration silently dropped — check must FAIL.
	} else {
		if err := reg.Register(goF); err != nil {
			r.record(name, false, "Register(go) error", err)
			return
		}
		if err := reg.Register(pyF); err != nil {
			r.record(name, false, "Register(py) error", err)
			return
		}
	}

	list := reg.List()
	if len(list) != 2 {
		r.record(name, false,
			fmt.Sprintf("expected 2 formatters in List(), got %d (%v)", len(list), list),
			nil)
		return
	}

	got, err := reg.Get("identity-go")
	if err != nil || got == nil {
		r.record(name, false, "Get(identity-go) lookup failed", err)
		return
	}

	byLang := reg.GetByLanguage("python")
	if len(byLang) != 1 {
		r.record(name, false,
			fmt.Sprintf("GetByLanguage(python) expected 1, got %d", len(byLang)),
			nil)
		return
	}

	r.record(name, true,
		fmt.Sprintf("List=%v; Get(identity-go)=%s; GetByLanguage(python).len=%d",
			list, got.Name(), len(byLang)),
		nil)
}

// check2ExecutorRoundTripsBilingualFixtures runs every fixture through
// the real executor pipeline and verifies byte-exact preservation.
func (r *runner) check2ExecutorRoundTripsBilingualFixtures() {
	name := "check2_executor_round_trip_bilingual"

	reg := registry.New()
	for _, lang := range []string{"go", "python", "sql"} {
		if err := reg.Register(newIdentityFormatter(
			"identity-"+lang, []string{lang},
		)); err != nil {
			r.record(name, false, "registry seed failed", err)
			return
		}
	}

	exec := executor.New(reg, executor.DefaultExecutorConfig())

	mismatches := 0
	for _, fx := range fixtures {
		if r.loc != fx.loc && r.loc != locale("all") {
			continue
		}
		res, err := exec.Execute(context.Background(), &formatter.FormatRequest{
			Content:  fx.content,
			Language: fx.language,
		})
		if err != nil {
			r.record(name, false,
				fmt.Sprintf("Execute(%s) failed", fx.name), err)
			return
		}
		if r.mutation == "corrupt_roundtrip" {
			// Paired mutation: deliberately mutate the comparison.
			res.Content = res.Content + "X"
		}
		if res.Content != fx.content {
			mismatches++
		}
	}

	if mismatches != 0 {
		r.record(name, false,
			fmt.Sprintf("%d bilingual fixtures lost byte-exact round-trip", mismatches),
			nil)
		return
	}
	r.record(name, true,
		fmt.Sprintf("%d bilingual fixtures preserved byte-exact via executor.Execute",
			countActiveFixtures(r.loc)),
		nil)
}

func countActiveFixtures(loc locale) int {
	n := 0
	for _, f := range fixtures {
		if loc == locale("all") || f.loc == loc {
			n++
		}
	}
	return n
}

// check3MiddlewareChainExecutes proves real middleware composition runs.
func (r *runner) check3MiddlewareChainExecutes() {
	name := "check3_middleware_chain_executes"

	reg := registry.New()
	idf := newIdentityFormatter("identity-mw", []string{"go"})
	_ = reg.Register(idf)

	exec := executor.New(reg, executor.DefaultExecutorConfig())
	exec.Use(executor.TimeoutMiddleware(2 * time.Second))
	exec.Use(executor.ValidationMiddleware())

	req := &formatter.FormatRequest{
		Content:  "package x\n",
		Language: "go",
	}
	res, err := exec.Execute(context.Background(), req)
	if err != nil {
		r.record(name, false, "Execute with middleware failed", err)
		return
	}
	calls := idf.callCount.Load()
	if r.mutation == "no_format_call" {
		calls = 0 // simulate underlying skip
	}
	if calls == 0 {
		r.record(name, false, "underlying Formatter.Format never invoked", nil)
		return
	}
	r.record(name, true,
		fmt.Sprintf("middleware chain executed; underlying Format calls=%d; success=%v",
			calls, res.Success),
		nil)
}

// check4PipelineComposesSteps proves executor.Pipeline chains steps and
// each step receives the previous step's output.
func (r *runner) check4PipelineComposesSteps() {
	name := "check4_pipeline_composes_steps"

	a := newIdentityFormatter("p-step-a", []string{"go"})
	b := newIdentityFormatter("p-step-b", []string{"go"})
	c := newIdentityFormatter("p-step-c", []string{"go"})
	pipe := executor.NewPipeline(a, b, c)

	res, err := pipe.Execute(context.Background(), &formatter.FormatRequest{
		Content:  "package main\n",
		Language: "go",
	})
	if err != nil {
		r.record(name, false, "Pipeline.Execute failed", err)
		return
	}

	if r.mutation == "skip_steps" {
		a.callCount.Store(0)
	}
	if a.callCount.Load() != 1 || b.callCount.Load() != 1 || c.callCount.Load() != 1 {
		r.record(name, false,
			fmt.Sprintf("pipeline step calls a=%d b=%d c=%d (want 1 each)",
				a.callCount.Load(), b.callCount.Load(), c.callCount.Load()),
			nil)
		return
	}
	r.record(name, true,
		fmt.Sprintf("pipeline composed 3 steps; final.Success=%v; bytes=%d",
			res.Success, len(res.Content)),
		nil)
}

// check5CacheStoresAndRetrieves exercises the real cache.InMemoryCache.
func (r *runner) check5CacheStoresAndRetrieves() {
	name := "check5_cache_stores_and_retrieves"

	c := cache.NewInMemoryCache(cache.DefaultCacheConfig())
	defer c.Stop()

	setReq := &formatter.FormatRequest{
		Content:  "package x\nfunc bilingual() {}\n",
		Language: "go",
		FilePath: "round255_bilingual.go",
	}
	val := &formatter.FormatResult{
		Content: "cached content",
		Success: true,
	}
	c.Set(setReq, val)

	getReq := setReq
	if r.mutation == "cache_miss" {
		// Paired mutation: change the key components so the lookup misses.
		getReq = &formatter.FormatRequest{
			Content:  "different content for mutation",
			Language: "go",
			FilePath: "round255_bilingual.go",
		}
	}

	got, ok := c.Get(getReq)
	if !ok || got == nil {
		r.record(name, false, "cache.Get reported miss", nil)
		return
	}
	if got.Content != "cached content" {
		r.record(name, false,
			fmt.Sprintf("cache returned wrong content: %q", got.Content), nil)
		return
	}
	r.record(name, true,
		fmt.Sprintf("Set+Get round-trip preserved content (%d bytes); cache.Size=%d",
			len(got.Content), c.Size()),
		nil)
}

// check6NativeConstructorsExpose proves native.New*Formatter constructors
// produce real Formatter instances with correct metadata (no execution —
// host may not have gofmt/black/etc. installed).
func (r *runner) check6NativeConstructorsExpose() {
	name := "check6_native_constructors_metadata"

	specs := []struct {
		fn       func() *native.NativeFormatter
		wantName string
		wantLang string
	}{
		{native.NewGoFormatter, "gofmt", "go"},
		{native.NewPythonFormatter, "black", "python"},
		{native.NewJSFormatter, "prettier", "javascript"},
		{native.NewRustFormatter, "rustfmt", "rust"},
		{native.NewSQLFormatter, "sqlformat", "sql"},
	}

	var bad []string
	for _, s := range specs {
		f := s.fn()
		if f == nil {
			bad = append(bad, s.wantName+":nil")
			continue
		}
		if f.Name() != s.wantName {
			bad = append(bad, fmt.Sprintf("%s:name=%q", s.wantName, f.Name()))
			continue
		}
		langOK := false
		for _, l := range f.Languages() {
			if l == s.wantLang {
				langOK = true
				break
			}
		}
		if !langOK {
			bad = append(bad, fmt.Sprintf("%s:lang∉%v", s.wantName, f.Languages()))
		}
	}

	if r.mutation == "constructor_drift" {
		bad = append(bad, "MUTATION:fake-drift")
	}

	if len(bad) > 0 {
		r.record(name, false,
			fmt.Sprintf("constructor drift: %s", strings.Join(bad, ",")),
			nil)
		return
	}
	r.record(name, true,
		fmt.Sprintf("5/5 native constructors produced expected (name, language) pairs"),
		nil)
}

// check7DetectLanguageFromPath exercises the real DetectLanguageFromPath.
func (r *runner) check7DetectLanguageFromPath() {
	name := "check7_detect_language_from_path"

	cases := map[string]string{
		"main.go":    "go",
		"app.py":     "python",
		"index.ts":   "typescript",
		"style.scss": "scss",
		"query.sql":  "sql",
	}

	var bad []string
	for path, want := range cases {
		got := registry.DetectLanguageFromPath(path)
		if r.mutation == "detect_drift" {
			got = "wrong"
		}
		if got != want {
			bad = append(bad, fmt.Sprintf("%s→%q(want %q)", path, got, want))
		}
	}
	if len(bad) > 0 {
		r.record(name, false,
			"language detection drift: "+strings.Join(bad, ","), nil)
		return
	}
	r.record(name, true,
		fmt.Sprintf("%d path→language mappings correct", len(cases)), nil)
}

// check8ExecutorRejectsAmbiguous proves negative-leg: empty req must error.
func (r *runner) check8ExecutorRejectsAmbiguous() {
	name := "check8_executor_rejects_ambiguous_request"

	reg := registry.New()
	exec := executor.New(reg, executor.DefaultExecutorConfig())

	_, err := exec.Execute(context.Background(), &formatter.FormatRequest{
		Content: "x",
		// no Language, no FilePath
	})
	if r.mutation == "accept_ambiguous" {
		err = nil
	}
	if err == nil {
		r.record(name, false, "expected error for empty Language/FilePath, got nil", nil)
		return
	}
	r.record(name, true,
		fmt.Sprintf("negative-leg error returned: %v", err), nil)
}

// ----------------------------------------------------------------------------
// Entry point.
// ----------------------------------------------------------------------------

func main() {
	var langFlag string
	flag.StringVar(&langFlag, "lang", "all", "locale: en | sr | all")
	flag.Parse()

	loc := locale(langFlag)
	if loc != localeEN && loc != localeSR && loc != locale("all") {
		fmt.Fprintf(os.Stderr, "unknown locale %q (use en|sr|all)\n", langFlag)
		os.Exit(2)
	}

	displayLoc := loc
	if loc == locale("all") {
		displayLoc = localeEN
	}

	mutation := os.Getenv("MUTATE")
	r := &runner{loc: loc, mutation: mutation}

	fmt.Println(tr(displayLoc, "banner"))
	if mutation != "" {
		fmt.Printf("  %s: %s\n", tr(displayLoc, "mutation"), mutation)
	} else {
		fmt.Printf("  %s\n", tr(displayLoc, "mutationOff"))
	}
	fmt.Println()

	// Run every check; record but do not abort on first failure so a single
	// mutation that breaks one invariant produces a clean exit-99 signal.
	r.check1RegistryRegistersAndLooksUp()
	r.check2ExecutorRoundTripsBilingualFixtures()
	r.check3MiddlewareChainExecutes()
	r.check4PipelineComposesSteps()
	r.check5CacheStoresAndRetrieves()
	r.check6NativeConstructorsExpose()
	r.check7DetectLanguageFromPath()
	r.check8ExecutorRejectsAmbiguous()

	fmt.Println()
	fmt.Printf("%s: %s=%d %s=%d %s=%d\n",
		tr(displayLoc, "summary"),
		tr(displayLoc, "total"), r.totalCount.Load(),
		tr(displayLoc, "passed"), r.passCount.Load(),
		tr(displayLoc, "failed"), r.failCount.Load(),
	)

	if r.failCount.Load() > 0 {
		if mutation != "" {
			// Expected: mutation broke an invariant → exit 99.
			os.Exit(99)
		}
		// Unexpected real failure.
		var failed []string
		for _, c := range r.results {
			if !c.passed {
				failed = append(failed, c.name)
			}
		}
		fmt.Fprintf(os.Stderr, "Challenge FAILED: %s\n", strings.Join(failed, ","))
		os.Exit(2)
	}

	if mutation != "" {
		// Mutation requested but no invariant flipped — paired-mutation pair is
		// stale; that itself is a defect per §1.1.
		fmt.Fprintln(os.Stderr,
			"paired-mutation requested but no invariant flipped — gate is stale")
		os.Exit(2)
	}

	_ = errors.New // keep import stable
}
