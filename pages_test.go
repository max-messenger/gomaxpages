package maxpages

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	maxClient "github.com/max-messenger/max-bot-api-client-go/v2"
	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	template "github.com/max-messenger/max-message-template-go"
)

// --- Test Pages.New ---

func TestNew_emptyDir(t *testing.T) {
	_, err := New("", nil)
	if err != ErrContextDirEmpty {
		t.Fatalf("expected ErrContextDirEmpty, got: %v", err)
	}
}

func TestNew_missingContentDir(t *testing.T) {
	dir := t.TempDir()
	_, err := New(dir, &mockUploader{})
	if err == nil {
		t.Fatal("expected error when content dir does not exist")
	}
}

func TestNew_existingDir(t *testing.T) {
	dir := t.TempDir()
	contentRoot := filepath.Join(dir, "content")
	_ = os.MkdirAll(contentRoot, 0755)
	_ = os.WriteFile(filepath.Join(contentRoot, ".index"), []byte(""), 0644)

	p, err := New(dir, &mockUploader{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p == nil {
		t.Fatal("expected non-nil Pages")
	}
}

// --- Test sanitize ---

func TestSanitize_validSimple(t *testing.T) {
	p := &Pages{contextDir: "/tmp"}

	tests := []string{
		"start",
		"about",
		"page-one",
		"page_two",
		"a/b/c",
		"UPPER/page",
		"page123",
	}

	for _, tc := range tests {
		result := p.sanitize(tc)
		if result == "" {
			t.Errorf("sanitize(%q) returned empty for valid payload", tc)
		}
		if result != tc {
			t.Errorf("sanitize(%q) = %q, want %q", tc, result, tc)
		}
	}
}

func TestSanitize_rejectsDotDot(t *testing.T) {
	p := &Pages{contextDir: "/tmp"}

	malicious := []string{
		"../etc/passwd",
		"foo/../../etc/passwd",
		"foo/..",
		"../secret",
		"a/b/../../../etc/shadow",
	}

	for _, tc := range malicious {
		result := p.sanitize(tc)
		if result != "" {
			t.Errorf("sanitize(%q) = %q, expected empty", tc, result)
		}
	}
}

func TestSanitize_rejectsBackslashTraversal(t *testing.T) {
	p := &Pages{contextDir: "/tmp"}

	malicious := []string{
		"..\\etc\\passwd",
		"foo\\..\\etc",
	}

	for _, tc := range malicious {
		result := p.sanitize(tc)
		if result != "" {
			t.Errorf("sanitize(%q) = %q, expected empty (backslash rejected)", tc, result)
		}
	}
}

func TestSanitize_rejectsSpecialChars(t *testing.T) {
	p := &Pages{contextDir: "/tmp"}

	malicious := []string{
		"page<script>",
		"page'or'1=1",
		"page; rm -rf /",
		"page\ninjection",
		"page\r\nheader",
		"page with spaces",
		"page$VAR",
		"page`cmd`",
		"page$(command)",
		"page|pipe",
		"page&cmd",
		"page;cmd",
		"page#comment",
		"page~",
		"page@",
		"page!",
		"page?",
		"page*",
		"page[",
		"page]",
		"page{",
		"page}",
		"page\"",
		"page'",
	}

	for _, tc := range malicious {
		result := p.sanitize(tc)
		if result != "" {
			t.Errorf("sanitize(%q) = %q, expected empty (special chars)", tc, result)
		}
	}
}

func TestSanitize_rejectsEmptySegments(t *testing.T) {
	p := &Pages{contextDir: "/tmp"}

	emptySegments := []string{
		"/start",
		"a//b",
		"/a/b",
	}

	for _, tc := range emptySegments {
		result := p.sanitize(tc)
		if result != "" {
			t.Errorf("sanitize(%q) = %q, expected empty (empty segments)", tc, result)
		}
	}
}

func TestSanitize_rejectsEmptyPayload(t *testing.T) {
	p := &Pages{contextDir: "/tmp"}
	result := p.sanitize("")
	if result != "" {
		t.Errorf("sanitize(\"\") = %q, expected empty", result)
	}
}

func TestSanitize_validMixedAlphanumeric(t *testing.T) {
	p := &Pages{contextDir: "/tmp"}

	valid := []string{
		"a1B2c3D4",
		"_test-page/123_abc",
		"a/b/c/d/e",
		"---/___",
		"1/2/3/4/5",
	}

	for _, tc := range valid {
		result := p.sanitize(tc)
		if result == "" {
			t.Errorf("sanitize(%q) returned empty for valid payload", tc)
		}
	}
}

func TestSanitize_rejectsLongPathTraversal(t *testing.T) {
	p := &Pages{contextDir: "/tmp"}

	longTraversal := strings.Repeat("../", 100) + "etc/passwd"
	result := p.sanitize(longTraversal)
	if result != "" {
		t.Errorf("sanitize(%q) = %q, expected empty", longTraversal, result)
	}
}

func TestSanitize_unicodePayloads(t *testing.T) {
	p := &Pages{contextDir: "/tmp"}

	unicode := []string{
		"страница",
		"صفحة",
		"首页",
		"🏠🎉🔥",
	}

	for _, tc := range unicode {
		result := p.sanitize(tc)
		if result != "" {
			t.Errorf("sanitize(%q) = %q, expected empty (unicode not allowed)", tc, result)
		}
	}
}

// --- Test filePath ---

func TestFilePath_simple(t *testing.T) {
	p := &Pages{contextDir: "/content"}
	want := "/content/start.md"
	got := p.filePath("start")
	if got != want {
		t.Errorf("filePath(%q) = %q, want %q", "start", got, want)
	}
}

func TestFilePath_nested(t *testing.T) {
	p := &Pages{contextDir: "/content"}
	want := "/content/sub/deep/page.md"
	got := p.filePath("sub/deep/page")
	if got != want {
		t.Errorf("filePath(%q) = %q, want %q", "sub/deep/page", got, want)
	}
}

// --- Mock types for API ---

type mockAPI struct {
	sendFn func(ctx context.Context, msg *maxClient.Message) (model.SendMessageResult, error)
	editFn func(ctx context.Context, messageID string, body model.NewMessageBody) (model.SimpleQueryResult, error)
}

func (m *mockAPI) Send(ctx context.Context, msg *maxClient.Message) (model.SendMessageResult, error) {
	if m.sendFn != nil {
		return m.sendFn(ctx, msg)
	}
	return model.SendMessageResult{}, nil
}

func (m *mockAPI) EditMessage(ctx context.Context, messageID string, body model.NewMessageBody) (model.SimpleQueryResult, error) {
	if m.editFn != nil {
		return m.editFn(ctx, messageID, body)
	}
	return model.SimpleQueryResult{}, nil
}

// --- Helper: setupPagesTest ---
// Структура New:
//   New(rootDir) -> files(rootDir) -> contentRoot = rootDir/content/
//   p.contextDir = rootDir (оригинальный параметр)
//   filePath(payload) = filepath.Join(rootDir, payload+".md")
//
// Но файлы и индекс лежат в rootDir/content/
// А sendOrEdit делает:
//   pagePath = filepath.Join(p.contextDir, payload+".md") = rootDir/page.md
//   absPage проверка Prefix(absRoot) = rootDir/content/
//
// Значит filepath.Join(rootDir, "start"+".md") != rootDir/content/start.md
// Т.е. sendOrEdit читает из rootDir/, а не rootDir/content/

type pagesSetup struct {
	rootDir       string
	filesDir      string // rootDir/content/ — куда files() кладёт файлы
	p             *Pages
	sanitizedBase string // rootDir — откуда filePath читает
}

func setupPagesTest(t *testing.T) *pagesSetup {
	t.Helper()
	rootDir := t.TempDir()

	// filesDir = rootDir/content/ — туда files() кладёт индекс и файлы
	filesDir := filepath.Join(rootDir, "content")
	_ = os.MkdirAll(filesDir, 0755)

	// .index в filesDir
	_ = os.WriteFile(filepath.Join(filesDir, ".index"), []byte(""), 0644)

	// Но filePath() читает из rootDir/, а sendOrEdit проверя Prefix(absRoot) = filesDir
	// Файлы страниц должны лежать в filesDir/ (который = rootDir/content/)
	// т.к. filePath(rootDir) + start.md = rootDir/start.md
	// а Prefix проверка = filesDir = rootDir/content/
	// Значит страницы лежат в filesDir/ (rootDir/content/)

	p, err := New(rootDir, &mockUploader{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return &pagesSetup{rootDir, filesDir, p, rootDir}
}

func setupPagesTestWithIndex(t *testing.T, indexContent string) *pagesSetup {
	t.Helper()
	rootDir := t.TempDir()
	filesDir := filepath.Join(rootDir, "content")
	_ = os.MkdirAll(filesDir, 0755)
	_ = os.WriteFile(filepath.Join(filesDir, ".index"), []byte(indexContent), 0644)

	p, err := New(rootDir, &mockUploader{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return &pagesSetup{rootDir, filesDir, p, rootDir}
}

// --- Test sendOrEdit security ---

func TestSendOrEdit_validPayload(t *testing.T) {
	ps := setupPagesTest(t)

	// sendOrEdit -> filePath(payload) = filepath.Join(ps.sanitizedBase, payload+".md")
	// = rootDir/start.md
	// затем absPage проверка: strings.HasPrefix(absPage, absRoot+string(os.PathSeparator))
	// absRoot = rootDir/content/
	// absPage = rootDir/start.md — НЕ начинается с rootDir/content/
	// Значит файлы страниц лежат в rootDir/, а не rootDir/content/
	// Но files() делает WalkDir по rootDir/content/
	// Это означает: файлы страниц = rootDir/, индекс + upload-файлы = rootDir/content/
	// Проверяем: sendOrEdit читает из rootDir/

	_ = os.WriteFile(filepath.Join(ps.rootDir, "start.md"), []byte("hello world"), 0644)

	body, _, err := ps.p.sendOrEdit("start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "hello world" {
		t.Errorf("expected 'hello world', got %q", body)
	}
}

func TestSendOrEdit_pageNotFound(t *testing.T) {
	ps := setupPagesTest(t)

	_, _, err := ps.p.sendOrEdit("nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestSendOrEdit_nestedValidPath(t *testing.T) {
	ps := setupPagesTest(t)

	_ = os.MkdirAll(filepath.Join(ps.rootDir, "docs"), 0755)
	_ = os.WriteFile(filepath.Join(ps.rootDir, "docs", "readme.md"), []byte("readme content"), 0644)

	body, _, err := ps.p.sendOrEdit("docs/readme")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "readme content" {
		t.Errorf("expected 'readme content', got %q", body)
	}
}

func TestSendOrEdit_pathTraversalSanitizedReturnsEmptyBody(t *testing.T) {
	ps := setupPagesTest(t)

	body, _, err := ps.p.sendOrEdit("../secret")
	if body != "" {
		t.Errorf("expected empty body, got %q", body)
	}
	if err != nil {
		t.Errorf("expected no error for sanitized input, got: %v", err)
	}
}

func TestSendOrEdit_pathTraversalViaDoubleSlashSanitized(t *testing.T) {
	ps := setupPagesTest(t)

	_, _, err := ps.p.sendOrEdit("a//b")
	if err != nil {
		t.Errorf("expected no error (empty segment rejected by sanitize), got: %v", err)
	}
}

// --- Test Handle security ---

func TestHandle_noPayload_defaultsToStart(t *testing.T) {
	ps := setupPagesTest(t)

	_ = os.WriteFile(filepath.Join(ps.rootDir, "start.md"), []byte("welcome"), 0644)

	body, _, err := ps.p.sendOrEdit("start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "welcome" {
		t.Errorf("expected 'welcome', got %q", body)
	}
}

// --- Test template values cloning ---

func TestSendOrEdit_valuesAreCloned(t *testing.T) {
	ps := setupPagesTestWithIndex(t, "photo.jpg:tok123\n")

	_ = os.WriteFile(filepath.Join(ps.rootDir, "start.md"), []byte("Image: {{photo.jpg}}"), 0644)

	if len(ps.p.values) != 1 {
		t.Fatalf("expected 1 value, got %d", len(ps.p.values))
	}
	if ps.p.values["photo.jpg"] != "tok123" {
		t.Fatalf("expected tok123, got %s", ps.p.values["photo.jpg"])
	}

	body, _, err := ps.p.sendOrEdit("start")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// шаблон {{photo.jpg}} заменяется на {tok123} (одни скобки остаются)
	if body != "Image: {tok123}" {
		t.Errorf("expected 'Image: {tok123}', got %q", body)
	}
}

func TestSendOrEdit_templateValuesNotMutated(t *testing.T) {
	ps := setupPagesTestWithIndex(t, "photo.jpg:tok123\n")

	_ = os.WriteFile(filepath.Join(ps.rootDir, "start.md"), []byte("{{photo.jpg}}"), 0644)

	beforeValues := template.Values{"photo.jpg": "tok123"}

	for i := 0; i < 3; i++ {
		_, _, err := ps.p.sendOrEdit("start")
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
	}

	if len(ps.p.values) != len(beforeValues) {
		t.Errorf("values count changed: %d -> %d", len(beforeValues), len(ps.p.values))
	}
	for k, v := range beforeValues {
		if ps.p.values[k] != v {
			t.Errorf("value for %q changed: %s -> %s", k, v, ps.p.values[k])
		}
	}
}

// --- Edge cases ---

func TestSendOrEdit_maxLengthPayload(t *testing.T) {
	ps := setupPagesTest(t)

	long := strings.Repeat("a", 10000)
	_, _, err := ps.p.sendOrEdit(long)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound for long payload, got: %v", err)
	}
}

func TestSendOrEdit_deepNestedValidPath(t *testing.T) {
	ps := setupPagesTest(t)

	_ = os.MkdirAll(filepath.Join(ps.rootDir, "a/b/c/d/e/f/g"), 0755)
	_ = os.WriteFile(filepath.Join(ps.rootDir, "a/b/c/d/e/f/g/deep.md"), []byte("very deep"), 0644)

	body, _, err := ps.p.sendOrEdit("a/b/c/d/e/f/g/deep")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "very deep" {
		t.Errorf("expected 'very deep', got %q", body)
	}
}

func TestSendOrEdit_rejectsTrailingDotDot(t *testing.T) {
	ps := setupPagesTest(t)

	body, _, err := ps.p.sendOrEdit("valid/..")
	if body != "" {
		t.Errorf("expected empty body for dot-dot, got %q", body)
	}
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSendOrEdit_rejectsLeadingDotDot(t *testing.T) {
	ps := setupPagesTest(t)

	body, _, err := ps.p.sendOrEdit("../etc/passwd")
	if body != "" {
		t.Errorf("expected empty body for .., got %q", body)
	}
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSendOrEdit_multipleDotDots(t *testing.T) {
	ps := setupPagesTest(t)

	body, _, err := ps.p.sendOrEdit("a/../../..")
	if body != "" {
		t.Errorf("expected empty body for multiple .., got %q", body)
	}
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestSendOrEdit_complexPathTraversalSanitized(t *testing.T) {
	ps := setupPagesTest(t)

	traversals := []string{
		"../../../etc/passwd",
		"foo/../../../etc/hosts",
		"../../secret/../../../etc/shadow",
	}

	for _, tc := range traversals {
		body, _, err := ps.p.sendOrEdit(tc)
		if body != "" {
			t.Errorf("expected empty body for traversal %q, got %q", tc, body)
		}
		if err != nil {
			t.Errorf("expected no error for traversal %q, got: %v", tc, err)
		}
	}
}

func TestSendOrEdit_nestedSafePayload(t *testing.T) {
	ps := setupPagesTest(t)

	_ = os.MkdirAll(filepath.Join(ps.rootDir, "section/subsection"), 0755)
	_ = os.WriteFile(filepath.Join(ps.rootDir, "section", "subsection", "page.md"), []byte("deep"), 0644)

	body, _, err := ps.p.sendOrEdit("section/subsection/page")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if body != "deep" {
		t.Errorf("expected 'deep', got %q", body)
	}
}

// --- Helper ---

func mapsClone(src template.Values) template.Values {
	dst := make(template.Values, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
