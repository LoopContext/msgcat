package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/loopcontext/msgcat"
	"gopkg.in/yaml.v2"
)

// mergeConfig holds flags for the merge command.
type mergeConfig struct {
	source          string
	targetLangs     string
	targetDir       string
	outdir          string
	translatePrefix string
}

func usageMerge() {
	fmt.Fprintf(os.Stderr, `usage: msgcat merge [options]

Merge produces per-language translate files from a source message file. For each target
language, writes translate.<lang>.yaml with every key from the source; keys missing or
empty in the target use source short/long as placeholder. Copies source 'group' and
'default' into each output file.

Flags:
`)
	flag.CommandLine.PrintDefaults()
}

func parseMergeFlags(args []string) (*mergeConfig, error) {
	fs := flag.NewFlagSet("merge", flag.ExitOnError)
	fs.Usage = usageMerge
	var cfg mergeConfig
	fs.StringVar(&cfg.source, "source", "", "Source message file (e.g. resources/messages/en.yaml). Required.")
	fs.StringVar(&cfg.targetLangs, "targetLangs", "", "Comma-separated target language tags (e.g. es,fr).")
	fs.StringVar(&cfg.targetDir, "targetDir", "", "Directory containing target YAMLs; language inferred from filenames (e.g. es.yaml -> es).")
	fs.StringVar(&cfg.outdir, "outdir", "", "Where to write translate.<lang>.yaml (default: same dir as source).")
	fs.StringVar(&cfg.translatePrefix, "translatePrefix", "translate.", "Filename prefix for output files.")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func runMerge(cfg *mergeConfig) error {
	if cfg.source == "" {
		return fmt.Errorf("merge: -source is required")
	}
	srcContent, err := os.ReadFile(cfg.source)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}
	var source msgcat.Messages
	if err := yaml.Unmarshal(srcContent, &source); err != nil {
		return fmt.Errorf("parse source YAML: %w", err)
	}
	if source.Set == nil {
		source.Set = make(map[string]msgcat.RawMessage)
	}
	// Ensure default has at least placeholders for valid msgcat YAML
	if source.Default.ShortTpl == "" {
		source.Default.ShortTpl = "Unexpected error"
	}
	if source.Default.LongTpl == "" {
		source.Default.LongTpl = "Message not found in catalog"
	}

	targets := cfg.targetLangsList()
	if len(targets) == 0 && cfg.targetDir != "" {
		targets, err = readTargetLangsFromDir(cfg.targetDir, cfg.source)
		if err != nil {
			return err
		}
	}
	if len(targets) == 0 {
		return fmt.Errorf("merge: specify -targetLangs or -targetDir")
	}

	outdir := cfg.outdir
	if outdir == "" {
		outdir = filepath.Dir(cfg.source)
	}
	prefix := cfg.translatePrefix
	if prefix == "" {
		prefix = "translate."
	}

	for _, lang := range targets {
		outPath := filepath.Join(outdir, prefix+lang+".yaml")
		var target msgcat.Messages
		targetPath := filepath.Join(filepath.Dir(cfg.source), lang+".yaml")
		if cfg.targetDir != "" {
			targetPath = filepath.Join(cfg.targetDir, lang+".yaml")
		}
		if tb, err := os.ReadFile(targetPath); err == nil {
			_ = yaml.Unmarshal(tb, &target)
		}
		if target.Set == nil {
			target.Set = make(map[string]msgcat.RawMessage)
		}
		// Build merged: for each key in source, use target if non-empty short and long, else source
		merged := msgcat.Messages{
			Group:   source.Group,
			Default: source.Default,
			Set:     make(map[string]msgcat.RawMessage),
		}
		for key, srcEntry := range source.Set {
			dstEntry := target.Set[key]
			if dstEntry.ShortTpl != "" && dstEntry.LongTpl != "" {
				merged.Set[key] = dstEntry
			} else {
				entry := msgcat.RawMessage{
					ShortTpl:    srcEntry.ShortTpl,
					LongTpl:     srcEntry.LongTpl,
					Code:        srcEntry.Code,
					ShortForms:  srcEntry.ShortForms,
					LongForms:   srcEntry.LongForms,
					PluralParam: srcEntry.PluralParam,
				}
				merged.Set[key] = entry
			}
		}
		out, err := yaml.Marshal(&merged)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", lang, err)
		}
		if err := os.WriteFile(outPath, out, 0644); err != nil {
			return fmt.Errorf("write %s: %w", outPath, err)
		}
		fmt.Fprintf(os.Stderr, "msgcat: wrote %s\n", outPath)
	}
	return nil
}

func (c *mergeConfig) targetLangsList() []string {
	if c.targetLangs == "" {
		return nil
	}
	var out []string
	for _, s := range strings.Split(c.targetLangs, ",") {
		s = strings.TrimSpace(strings.ToLower(s))
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func readTargetLangsFromDir(dir, sourcePath string) ([]string, error) {
	sourceBase := filepath.Base(sourcePath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var langs []string
	seen := make(map[string]struct{})
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".yaml") {
			continue
		}
		if name == sourceBase || strings.HasPrefix(name, "translate.") {
			continue
		}
		lang := strings.TrimSuffix(name, ".yaml")
		lang = strings.TrimSpace(strings.ToLower(lang))
		if lang == "" {
			continue
		}
		if _, ok := seen[lang]; ok {
			continue
		}
		seen[lang] = struct{}{}
		langs = append(langs, lang)
	}
	return langs, nil
}
