package po

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	gotext "github.com/leonelquinteros/gotext"
)

const samplePO = `
msgid ""
msgstr ""
"Project-Id-Version: test\n"
"Content-Type: text/plain; charset=UTF-8\n"
"Plural-Forms: nplurals=2; plural=(n != 1);\n"
"Language: es\n"

msgid "Hello"
msgstr "Hola"

msgid "File"
msgid_plural "Files"
msgstr[0] "Archivo"
msgstr[1] "Archivos"

msgctxt "menu"
msgid "Open"
msgstr "Abrir"
`

func TestCompileProducesReadableMo(t *testing.T) {
	svc := NewService()

	res, err := svc.Compile(context.Background(), samplePO, "base64")
	if err != nil {
		t.Fatalf("compile returned error: %v", err)
	}
	if res.Base64 == "" {
		t.Fatalf("expected base64 output")
	}

	moBytes, err := base64.StdEncoding.DecodeString(res.Base64)
	if err != nil {
		t.Fatalf("cannot decode base64: %v", err)
	}

	mo := gotext.NewMo()
	mo.Parse(moBytes)

	if got := mo.Get("Hello"); got != "Hola" {
		t.Fatalf("expected 'Hola', got '%s'", got)
	}
	if got := mo.GetN("File", "Files", 1); got != "Archivo" {
		t.Fatalf("expected singular plural form, got '%s'", got)
	}
	if got := mo.GetN("File", "Files", 3); got != "Archivos" {
		t.Fatalf("expected plural form, got '%s'", got)
	}
	if got := mo.GetC("Open", "menu"); got != "Abrir" {
		t.Fatalf("expected context translation, got '%s'", got)
	}

	if res.Stats.Total != 3 || res.Stats.Translated != 3 {
		t.Fatalf("unexpected stats: %+v", res.Stats)
	}
	if res.Stats.Language != "es" {
		t.Fatalf("expected language 'es', got '%s'", res.Stats.Language)
	}
}

func TestCompileToPath(t *testing.T) {
	svc := NewService()
	res, err := svc.Compile(context.Background(), samplePO, "path")
	if err != nil {
		t.Fatalf("compile returned error: %v", err)
	}
	if res.Path == "" {
		t.Fatalf("expected temp path")
	}
	if _, err := os.Stat(res.Path); err != nil {
		t.Fatalf("temp mo file missing: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(res.Path) })
}

func TestValidateWarnings(t *testing.T) {
	svc := NewService()
	raw := "msgid \"\"\nmsgstr \"\"\n\nmsgid \"Hello\"\nmsgstr \"\"\n"

	warnings, summary, err := svc.Validate(context.Background(), raw)
	if err != nil {
		t.Fatalf("validate returned error: %v", err)
	}
	if len(warnings) == 0 {
		t.Fatalf("expected warnings for missing headers and untranslated entry")
	}
	if summary.Total != 1 || summary.Translated != 0 || summary.Untranslated != 1 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}
