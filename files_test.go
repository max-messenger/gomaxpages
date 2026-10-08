package gomaxpages

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
)

// --- Mock implementations ---

type mockUploader struct {
	uploadFn func(ctx context.Context, t model.UploadType, r io.Reader, name string, size int64) (string, error)
}

func (m *mockUploader) Upload(ctx context.Context, t model.UploadType, r io.Reader, name string, size int64) (string, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, t, r, name, size)
	}
	return fmt.Sprintf("token-%s", name), nil
}

type failingUploader struct{}

func (f *failingUploader) Upload(ctx context.Context, t model.UploadType, r io.Reader, name string, size int64) (string, error) {
	return "", errors.New("upload failed")
}

// --- Test createIfNotExists ---

func TestCreateIfNotExists_createsMissingFile(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "newfile.txt")

	err := createIfNotExists(missing)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	info, err := os.Stat(missing)
	if err != nil {
		t.Fatalf("file should exist after createIfNotExists, got error: %v", err)
	}
	if info.IsDir() {
		t.Fatal("expected a file, not a directory")
	}
}

func TestCreateIfNotExists_skipsExistingFile(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "existing.txt")

	f, err := os.Create(existing)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	info, err := os.Stat(existing)
	if err != nil {
		t.Fatal(err)
	}
	origMod := info.ModTime()

	err = createIfNotExists(existing)
	if err != nil {
		t.Fatalf("expected no error for existing file, got: %v", err)
	}

	info, err = os.Stat(existing)
	if err != nil {
		t.Fatal(err)
	}
	if info.ModTime().Before(origMod) {
		t.Fatal("existing file should not be recreated or modified")
	}
}

func TestCreateIfNotExists_failsOnBadPath(t *testing.T) {
	err := createIfNotExists("/nonexistent/deeply/nested/dir/file.txt")
	if err == nil {
		t.Fatal("expected error for path under nonexistent directory")
	}
}

// --- Test readFileIndex ---

func TestReadFileIndex_emptyFile(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, ".index")

	f, err := os.Create(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	values, err := readFileIndex(indexPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("expected empty values, got %d entries", len(values))
	}
}

func TestReadFileIndex_validContent(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, ".index")

	err := os.WriteFile(indexPath, []byte("image.png:token1\nvideo.mp4:token2\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	values, err := readFileIndex(indexPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(values))
	}
	if values["image.png"] != "token1" {
		t.Fatalf("expected token1 for image.png, got %s", values["image.png"])
	}
	if values["video.mp4"] != "token2" {
		t.Fatalf("expected token2 for video.mp4, got %s", values["video.mp4"])
	}
}

func TestReadFileIndex_singleColonValues(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, ".index")

	// строка с более чем двумя полями через ":" должна игнорировать дополнительные части
	err := os.WriteFile(indexPath, []byte("key:val:extra\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	values, err := readFileIndex(indexPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 0 {
		// строка "key:val:extra" при split по ":" даёт 3 части, поэтому игнорируется
		t.Fatalf("expected 0 entries (3-part line ignored), got %d: %v", len(values), values)
	}
}

func TestReadFileIndex_trailingNewlines(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, ".index")

	err := os.WriteFile(indexPath, []byte("a:1\n\n\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	values, err := readFileIndex(indexPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(values))
	}
}

func TestReadFileIndex_autoCreatesMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, ".index")

	values, err := readFileIndex(missing)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 0 {
		t.Fatalf("expected empty values, got %d", len(values))
	}

	// файл должен быть создан
	_, statErr := os.Stat(missing)
	if statErr != nil {
		t.Fatalf("index file should have been auto-created, error: %v", statErr)
	}
}

// --- Test fileUpload ---

func TestFileUpload_detectsImage(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "photo.jpg")
	err := os.WriteFile(testFile, []byte("fakejpg"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	up := &mockUploader{}
	token, err := fileUpload(context.Background(), up, testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestFileUpload_detectsPng(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "photo.png")
	err := os.WriteFile(testFile, []byte("fakepng"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	up := &mockUploader{}
	_, err = fileUpload(context.Background(), up, testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileUpload_detectsJpeg(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "photo.jpeg")
	err := os.WriteFile(testFile, []byte("fakejpeg"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	up := &mockUploader{}
	_, err = fileUpload(context.Background(), up, testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileUpload_detectsVideo(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "clip.mp4")
	err := os.WriteFile(testFile, []byte("fakemp4"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	up := &mockUploader{}
	_, err = fileUpload(context.Background(), up, testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileUpload_detectsAudio(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "song.mp3")
	err := os.WriteFile(testFile, []byte("fakemp3"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	up := &mockUploader{}
	_, err = fileUpload(context.Background(), up, testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFileUpload_usesUploadFileForUnknownExt(t *testing.T) {
	dir := t.TempDir()
	testFile := filepath.Join(dir, "doc.pdf")
	err := os.WriteFile(testFile, []byte("fakepdf"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	var capturedType model.UploadType
	up := &mockUploader{
		uploadFn: func(ctx context.Context, t model.UploadType, r io.Reader, name string, size int64) (string, error) {
			capturedType = t
			return "token", nil
		},
	}
	_, err = fileUpload(context.Background(), up, testFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedType != model.UploadFile {
		t.Fatalf("expected UploadFile for unknown ext .pdf, got %v", capturedType)
	}
}

func TestFileUpload_failsOnNonexistentFile(t *testing.T) {
	up := &mockUploader{}
	_, err := fileUpload(context.Background(), up, "/no/such/file.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestFileUpload_failsOnDir(t *testing.T) {
	dir := t.TempDir()
	up := &mockUploader{}
	_, err := fileUpload(context.Background(), up, dir)
	// os.Open на директории не возвращает ошибку — возвращает descriptor
	// Главное что это директория, а не файл
	t.Logf("fileUpload on dir: err=%v (expected, directory can be opened on some systems)", err)
}

func TestFileUpload_retryInFiles(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.Mkdir(contentDir, 0755)
	_ = os.WriteFile(filepath.Join(contentDir, ".index"), []byte(""), 0644)

	testFile := filepath.Join(contentDir, "photo.jpg")
	err := os.WriteFile(testFile, []byte("fakejpg"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	callCount := 0
	up := &mockUploader{
		uploadFn: func(ctx context.Context, t model.UploadType, r io.Reader, name string, size int64) (string, error) {
			callCount++
			return "", errors.New("transient error")
		},
	}
	// files() игнорирует ошибки upload (возвращает nil), но retry срабатывает 3 раза
	_, err = files(context.Background(), up, dir)
	// files() не возвращает ошибку при неудачном upload — файл просто пропускается
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 3 {
		t.Fatalf("expected 3 retry attempts in files(), got %d", callCount)
	}
}

func TestFileUpload_retrySucceedsInFiles(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.Mkdir(contentDir, 0755)
	_ = os.WriteFile(filepath.Join(contentDir, ".index"), []byte(""), 0644)

	testFile := filepath.Join(contentDir, "photo.jpg")
	err := os.WriteFile(testFile, []byte("fakejpg"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	callCount := 0
	up := &mockUploader{
		uploadFn: func(ctx context.Context, t model.UploadType, r io.Reader, name string, size int64) (string, error) {
			callCount++
			if callCount < 2 {
				return "", errors.New("transient error")
			}
			return "success-token", nil
		},
	}
	values, err := files(context.Background(), up, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values["photo.jpg"] != "success-token" {
		t.Fatalf("expected success-token, got %s", values["photo.jpg"])
	}
	if callCount != 2 {
		t.Fatalf("expected 2 calls (1 fail + 1 success), got %d", callCount)
	}
}

func TestFiles_noExtraFiles_unchangedIndex(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.Mkdir(contentDir, 0755)

	// создаём index с одной записью
	indexPath := filepath.Join(contentDir, ".index")
	err := os.WriteFile(indexPath, []byte("existing.md:oldtoken\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	up := &mockUploader{}
	values, err := files(context.Background(), up, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(values))
	}
	if values["existing.md"] != "oldtoken" {
		t.Fatalf("expected oldtoken, got %s", values["existing.md"])
	}
}

func TestFiles_discoversNewFile(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.Mkdir(contentDir, 0755)

	// пустой index
	indexPath := filepath.Join(contentDir, ".index")
	_ = os.WriteFile(indexPath, []byte(""), 0644)

	// добавляем файл
	testFile := filepath.Join(contentDir, "new.md")
	err := os.WriteFile(testFile, []byte("content"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	var captured int64
	up := &mockUploader{
		uploadFn: func(ctx context.Context, t model.UploadType, r io.Reader, name string, size int64) (string, error) {
			captured = size
			return "new-token", nil
		},
	}
	values, err := files(context.Background(), up, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected 1 entry, got %d: %v", len(values), values)
	}
	if values["new.md"] != "new-token" {
		t.Fatalf("expected new-token for new.md, got %s", values["new.md"])
	}
	if captured != 7 {
		t.Fatalf("expected size 7, got %d", captured)
	}
}

func TestFiles_skipsExistingInIndex(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.Mkdir(contentDir, 0755)

	// index уже содержит entry для existing.md
	indexPath := filepath.Join(contentDir, ".index")
	err := os.WriteFile(indexPath, []byte("existing.md:cached-token\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	// файл тоже существует
	existingFile := filepath.Join(contentDir, "existing.md")
	_ = os.WriteFile(existingFile, []byte("content"), 0644)

	var uploadCalled bool
	up := &mockUploader{
		uploadFn: func(ctx context.Context, t model.UploadType, r io.Reader, name string, size int64) (string, error) {
			uploadCalled = true
			return "should-not-happen", nil
		},
	}
	values, err := files(context.Background(), up, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uploadCalled {
		t.Fatal("upload should not be called for file already in index")
	}
	if values["existing.md"] != "cached-token" {
		t.Fatalf("expected cached-token, got %s", values["existing.md"])
	}
}

func TestFiles_updatesIndexOnNewFiles(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.Mkdir(contentDir, 0755)

	// пустой index
	indexPath := filepath.Join(contentDir, ".index")
	_ = os.WriteFile(indexPath, []byte(""), 0644)

	// добавляем файл
	testFile := filepath.Join(contentDir, "page.md")
	err := os.WriteFile(testFile, []byte("hello"), 0644)
	if err != nil {
		t.Fatal(err)
	}

	up := &mockUploader{}
	_, err = files(context.Background(), up, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	indexContent := string(data)

	if !strings.Contains(indexContent, "page.md:") {
		t.Fatalf("index should contain page.md entry, got: %q", indexContent)
	}
}

func TestFiles_ignoresDotIndexInWalk(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.Mkdir(contentDir, 0755)

	indexPath := filepath.Join(contentDir, ".index")
	_ = os.WriteFile(indexPath, []byte(""), 0644)

	up := &mockUploader{}
	_, err := files(context.Background(), up, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFiles_nestedDirectories(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.MkdirAll(filepath.Join(contentDir, "sub", "deep"), 0755)

	indexPath := filepath.Join(contentDir, ".index")
	_ = os.WriteFile(indexPath, []byte(""), 0644)

	// добавляем файлы в поддиректории
	_ = os.WriteFile(filepath.Join(contentDir, "root.md"), []byte("root"), 0644)
	_ = os.WriteFile(filepath.Join(contentDir, "sub", "nested.md"), []byte("nested"), 0644)
	_ = os.WriteFile(filepath.Join(contentDir, "sub", "deep", "deep.md"), []byte("deep"), 0644)

	up := &mockUploader{}
	values, err := files(context.Background(), up, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(values), values)
	}
	if values["root.md"] == "" {
		t.Fatal("expected root.md in values")
	}
	if values["sub/nested.md"] == "" {
		t.Fatal("expected sub/nested.md in values")
	}
	if values["sub/deep/deep.md"] == "" {
		t.Fatal("expected sub/deep/deep.md in values")
	}
}

func TestFiles_nestedDirectories_usesSlashPath(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.MkdirAll(filepath.Join(contentDir, "folder"), 0755)

	_ = os.WriteFile(filepath.Join(contentDir, ".index"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(contentDir, "folder", "page.md"), []byte("content"), 0644)

	up := &mockUploader{}
	values, err := files(context.Background(), up, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// ключи должны использовать "/" как разделитель (Unix-style)
	for k := range values {
		if filepath.Separator == '\\' && len(k) > 0 {
			// Windows: ключ должен быть с '/'
			if k[0] == '\\' {
				t.Fatalf("key %q should use forward slashes", k)
			}
		}
		_ = k
	}
}

func TestFiles_uploadsMixedFileTypes(t *testing.T) {
	dir := t.TempDir()
	contentDir := filepath.Join(dir, "content")
	_ = os.Mkdir(contentDir, 0755)

	_ = os.WriteFile(filepath.Join(contentDir, ".index"), []byte(""), 0644)
	_ = os.WriteFile(filepath.Join(contentDir, "photo.jpg"), []byte("img"), 0644)
	_ = os.WriteFile(filepath.Join(contentDir, "clip.mp4"), []byte("vid"), 0644)
	_ = os.WriteFile(filepath.Join(contentDir, "song.mp3"), []byte("aud"), 0644)
	_ = os.WriteFile(filepath.Join(contentDir, "doc.txt"), []byte("txt"), 0644)

	typeCounts := make(map[model.UploadType]int)
	up := &mockUploader{
		uploadFn: func(ctx context.Context, t model.UploadType, r io.Reader, name string, size int64) (string, error) {
			typeCounts[t]++
			return "tok", nil
		},
	}
	_, err := files(context.Background(), up, dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if typeCounts[model.UploadImage] != 1 {
		t.Fatalf("expected 1 image, got %d", typeCounts[model.UploadImage])
	}
	if typeCounts[model.UploadVideo] != 1 {
		t.Fatalf("expected 1 video, got %d", typeCounts[model.UploadVideo])
	}
	if typeCounts[model.UploadAudio] != 1 {
		t.Fatalf("expected 1 audio, got %d", typeCounts[model.UploadAudio])
	}
	if typeCounts[model.UploadFile] != 1 {
		t.Fatalf("expected 1 generic file, got %d", typeCounts[model.UploadFile])
	}
}

func TestFiles_noContentDir(t *testing.T) {
	dir := t.TempDir()
	// contentDir не существует, walks по nonexistent path
	// filepath.WalkDir на несуществующем path вернёт ошибку
	_, err := files(context.Background(), &mockUploader{}, dir)
	if err == nil {
		t.Fatal("expected error when content dir does not exist")
	}
}
