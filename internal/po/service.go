package po

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	gotext "github.com/leonelquinteros/gotext"
)

const (
	moMagicLittleEndian = 0x950412de
	moHeaderSize        = 28 // 7 uint32 values
)

// Service provides .po parsing, validation, and .mo compilation.
type Service struct{}

// CompileResult holds the compiled .mo payload and catalog stats.
type CompileResult struct {
	Base64 string
	Path   string
	Stats  Summary
}

// Summary collects quick catalog metrics.
type Summary struct {
	Language     string
	Total        int
	Translated   int
	Fuzzy        int
	Untranslated int
}

// NewService constructs a new Service instance.
func NewService() *Service {
	return &Service{}
}

// Compile consumes .po content and returns a compiled .mo blob (base64 or path).
func (s *Service) Compile(ctx context.Context, poContent string, returnMode string) (*CompileResult, error) {
	domain, err := parseDomain(poContent)
	if err != nil {
		return nil, err
	}

	entries := domainToEntries(domain)
	moBin, err := buildMO(entries)
	if err != nil {
		return nil, err
	}

	stats := summarizeDomain(domain)

	switch strings.ToLower(returnMode) {
	case "path":
		f, err := os.CreateTemp("", "mcp-po-*.mo")
		if err != nil {
			return nil, fmt.Errorf("cannot create temp mo file: %w", err)
		}
		if _, err := f.Write(moBin); err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("cannot write temp mo file: %w", err)
		}
		if err := f.Close(); err != nil {
			return nil, fmt.Errorf("cannot close temp mo file: %w", err)
		}
		return &CompileResult{Path: f.Name(), Stats: stats}, nil
	default:
		return &CompileResult{Base64: base64.StdEncoding.EncodeToString(moBin), Stats: stats}, nil
	}
}

// Validate analyzes .po content and returns warnings/errors and metrics.
func (s *Service) Validate(ctx context.Context, poContent string) ([]string, Summary, error) {
	domain, err := parseDomain(poContent)
	if err != nil {
		return nil, Summary{}, err
	}

	warnings := validateDomain(domain)
	return warnings, summarizeDomain(domain), nil
}

// Summarize extracts headers and progress metrics from .po content.
func (s *Service) Summarize(ctx context.Context, poContent string) (Summary, error) {
	domain, err := parseDomain(poContent)
	if err != nil {
		return Summary{}, err
	}
	return summarizeDomain(domain), nil
}

// parseDomain parses PO content into a gotext domain.
func parseDomain(poContent string) (*gotext.Domain, error) {
	if strings.TrimSpace(poContent) == "" {
		return nil, errors.New("empty po content")
	}

	p := gotext.NewPo()
	p.Parse([]byte(poContent))
	domain := p.GetDomain()
	return domain, nil
}

// moEntry represents a single msgid/msgstr pair for .mo output.
type moEntry struct {
	id  []byte
	val []byte
}

// domainToEntries flattens translations (with context/plurals) into .mo entries.
func domainToEntries(domain *gotext.Domain) []moEntry {
	entries := make([]moEntry, 0)

	translations := domain.GetTranslations()
	if hdr, ok := translations[""]; ok {
		entries = append(entries, moEntry{[]byte(""), []byte(hdr.Get())})
		delete(translations, "")
	}

	for id, tr := range translations {
		entries = append(entries, normalizeEntry(id, "", tr))
	}

	ctxTranslations := domain.GetCtxTranslations()
	for ctx, ctxMap := range ctxTranslations {
		for id, tr := range ctxMap {
			entries = append(entries, normalizeEntry(id, ctx, tr))
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return string(entries[i].id) < string(entries[j].id)
	})

	return entries
}

// normalizeEntry converts a translation into its .mo key/value (handles context/plural forms).
func normalizeEntry(id, ctx string, tr *gotext.Translation) moEntry {
	key := id
	if ctx != "" {
		key = ctx + "\x04" + key
	}

	if tr.PluralID != "" {
		key = key + "\x00" + tr.PluralID
		forms := pluralForms(tr)
		return moEntry{[]byte(key), []byte(strings.Join(forms, "\x00"))}
	}

	return moEntry{[]byte(key), []byte(tr.Get())}
}

// pluralForms collects plural translations in index order, padding gaps with empty strings for determinism.
func pluralForms(tr *gotext.Translation) []string {
	maxIdx := 0
	for idx := range tr.Trs {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	forms := make([]string, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		forms[i] = tr.Trs[i]
	}
	return forms
}

// buildMO produces a deterministic little-endian .mo binary from prepared entries.
func buildMO(entries []moEntry) ([]byte, error) {
	count := uint32(len(entries))

	// Tables for msgid and msgstr (length + offset each).
	origTable := make([]byte, count*8)
	transTable := make([]byte, count*8)

	// Data section starts after header + both tables.
	dataOffset := uint32(moHeaderSize) + count*8*2
	curOffset := dataOffset
	data := bytes.NewBuffer(nil)

	for i, ent := range entries {
		msgid := append(ent.id, 0x00)
		msgstr := append(ent.val, 0x00)

		binary.LittleEndian.PutUint32(origTable[i*8:], uint32(len(msgid)-1))
		binary.LittleEndian.PutUint32(origTable[i*8+4:], curOffset)
		data.Write(msgid)
		curOffset += uint32(len(msgid))

		binary.LittleEndian.PutUint32(transTable[i*8:], uint32(len(msgstr)-1))
		binary.LittleEndian.PutUint32(transTable[i*8+4:], curOffset)
		data.Write(msgstr)
		curOffset += uint32(len(msgstr))
	}

	out := bytes.NewBuffer(make([]byte, 0, curOffset))
	header := []uint32{
		moMagicLittleEndian,
		0, // version
		count,
		uint32(moHeaderSize),
		uint32(moHeaderSize) + count*8,
		0, // hash table size
		0, // hash table offset
	}

	for _, v := range header {
		if err := binary.Write(out, binary.LittleEndian, v); err != nil {
			return nil, fmt.Errorf("write header: %w", err)
		}
	}

	if _, err := out.Write(origTable); err != nil {
		return nil, fmt.Errorf("write orig table: %w", err)
	}
	if _, err := out.Write(transTable); err != nil {
		return nil, fmt.Errorf("write trans table: %w", err)
	}
	if _, err := out.Write(data.Bytes()); err != nil {
		return nil, fmt.Errorf("write data: %w", err)
	}

	return out.Bytes(), nil
}

// summarizeDomain collects basic metrics for progress reporting.
func summarizeDomain(domain *gotext.Domain) Summary {
	stats := Summary{Language: domain.Language}

	countTranslation := func(tr *gotext.Translation) {
		stats.Total++
		if translated(tr) {
			stats.Translated++
		}
	}

	translations := domain.GetTranslations()
	delete(translations, "") // header does not count
	for _, tr := range translations {
		countTranslation(tr)
	}

	ctxTranslations := domain.GetCtxTranslations()
	for _, ctxMap := range ctxTranslations {
		for _, tr := range ctxMap {
			countTranslation(tr)
		}
	}

	stats.Untranslated = stats.Total - stats.Translated - stats.Fuzzy
	return stats
}

// translated determines if all available plural forms are non-empty.
func translated(tr *gotext.Translation) bool {
	if tr.PluralID == "" {
		return tr.IsTranslated()
	}
	maxIdx := 0
	for idx := range tr.Trs {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	for i := 0; i <= maxIdx; i++ {
		if tr.Trs[i] == "" {
			return false
		}
	}
	return true
}

// validateDomain produces warnings for missing headers or empty translations.
func validateDomain(domain *gotext.Domain) []string {
	warnings := make([]string, 0)

	if strings.TrimSpace(domain.Language) == "" {
		warnings = append(warnings, "Language header missing")
	}
	if strings.TrimSpace(domain.PluralForms) == "" {
		warnings = append(warnings, "Plural-Forms header missing")
	}

	translations := domain.GetTranslations()
	delete(translations, "")
	for id, tr := range translations {
		if !translated(tr) {
			warnings = append(warnings, fmt.Sprintf("untranslated entry: %s", id))
		}
	}

	ctxTranslations := domain.GetCtxTranslations()
	for ctx, ctxMap := range ctxTranslations {
		for id, tr := range ctxMap {
			if !translated(tr) {
				warnings = append(warnings, fmt.Sprintf("untranslated entry: %s (ctx: %s)", id, ctx))
			}
		}
	}

	sort.Strings(warnings)
	return warnings
}
