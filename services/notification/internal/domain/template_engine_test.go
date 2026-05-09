package domain

import (
	"regexp"
	"testing"

	"pgregory.net/rapid"
)

// variableRe digunakan untuk mendeteksi pola {variable} yang tersisa setelah render.
var variableRe = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}`)

func genVarName(t *rapid.T, label string) string {
	return rapid.StringMatching(`[a-zA-Z_][a-zA-Z0-9_]{0,9}`).Draw(t, label)
}

// genPlainText menghasilkan teks biasa tanpa kurung kurawal agar tidak membentuk pola {var}.
func genPlainText(t *rapid.T, label string) string {
	return rapid.StringMatching(`[a-zA-Z0-9 .,!?:;\-]{0,30}`).Draw(t, label)
}

// **Memvalidasi: Kebutuhan 5.1, 5.3**
//
// Untuk setiap template body yang mengandung placeholder {variable} dan data map
// mengandung pola {variable} - semua placeholder diganti dengan nilai atau string kosong.
func TestProperty_TemplateRenderingCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		engine := NewTemplateEngine()

		// Buat jumlah variabel acak (1-5)
		numVars := rapid.IntRange(1, 5).Draw(t, "numVars")
		varNames := make([]string, numVars)
		for i := 0; i < numVars; i++ {
			varNames[i] = genVarName(t, "varName")
		}

		// Bangun template body dengan placeholder
		var body string
		for i, v := range varNames {
			prefix := genPlainText(t, "prefix")
			body += prefix + "{" + v + "}"
			if i == numVars-1 {
				suffix := genPlainText(t, "suffix")
				body += suffix
			}
		}

		// Buat data map - sebagian variabel mungkin tidak ada di map
		data := make(map[string]string)
		for _, v := range varNames {
			if rapid.Bool().Draw(t, "includeVar_"+v) {
				data[v] = genPlainText(t, "val_"+v)
			}
		}

		// Render template
		result := engine.Render(body, data)

		// Verifikasi: tidak ada pola {variable} yang tersisa
		if variableRe.MatchString(result) {
			remaining := variableRe.FindAllString(result, -1)
			t.Fatalf(
				"Render(%q, %v) = %q, masih mengandung placeholder: %v",
				body, data, result, remaining,
			)
		}
	})
}

// **Memvalidasi: Kebutuhan 5.1, 5.3, 5.4**
//
// render(render(body, data), data) == render(body, data)
func TestProperty_TemplateRenderIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		engine := NewTemplateEngine()

		// Buat jumlah variabel acak (1-5)
		numVars := rapid.IntRange(1, 5).Draw(t, "numVars")
		varNames := make([]string, numVars)
		for i := 0; i < numVars; i++ {
			varNames[i] = genVarName(t, "varName")
		}

		// Bangun template body dengan placeholder
		var body string
		for i, v := range varNames {
			prefix := genPlainText(t, "prefix")
			body += prefix + "{" + v + "}"
			if i == numVars-1 {
				suffix := genPlainText(t, "suffix")
				body += suffix
			}
		}

		// Buat data map - semua variabel memiliki nilai
		data := make(map[string]string)
		for _, v := range varNames {
			data[v] = genPlainText(t, "val_"+v)
		}

		// Render pertama
		firstRender := engine.Render(body, data)

		secondRender := engine.Render(firstRender, data)

		// Verifikasi: hasil render kedua identik dengan render pertama
		if firstRender != secondRender {
			t.Fatalf(
				"Idempotence gagal:\n  body:   %q\n  data:   %v\n  render1: %q\n  render2: %q",
				body, data, firstRender, secondRender,
			)
		}
	})
}
