package gomaxpages

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/max-messenger/max-bot-api-client-go/v2/model"
	template "github.com/max-messenger/max-message-template-go"
)

//nolint:cyclop
func files(ctx context.Context, uploads UploadAPI, contextDir string) (template.Values, error) {
	contentRoot := filepath.Join(contextDir, "content")

	// индекс всегда в корне content/
	indexPath := filepath.Join(contentRoot, ".index")
	values, err := readFileIndex(indexPath)
	if err != nil {
		return nil, err
	}

	var changed bool
	err = filepath.WalkDir(contentRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == ".index" {
			return nil
		}

		// относительный путь от content/ — он же ключ в индексе
		rel, relErr := filepath.Rel(contentRoot, path)
		if relErr != nil {
			return relErr
		}
		// нормализуем разделители на случай Windows
		rel = filepath.ToSlash(rel)

		if _, ok := values.Get(rel); ok {
			return nil
		}

		var token string
		for retry := 3; retry > 0; retry-- {
			token, err = fileUpload(ctx, uploads, path)
			if err == nil {
				values[rel] = token
				changed = true

				break
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// save index
	if changed {
		var file *os.File
		file, err = os.Create(indexPath)
		if err != nil {
			return values, err
		}
		defer func() { _ = file.Close() }()
		for k, v := range values {
			_, _ = file.WriteString(fmt.Sprintf("%s:%s\n", k, v)) //nolint:all
		}
	}

	return values, nil
}

func readFileIndex(path string) (template.Values, error) {
	err := createIfNotExists(path)
	if err != nil {
		return nil, err
	}

	values := template.Values{}
	data, err := os.ReadFile(path)
	if err != nil {
		return values, err
	}

	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			values[parts[0]] = parts[1]
		}
	}

	return values, nil
}

func createIfNotExists(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		f, fErr := os.Create(path)
		if fErr != nil {
			return err
		}

		_ = f.Close()
	}

	return nil
}

func fileUpload(ctx context.Context, uploads UploadAPI, path string) (string, error) {
	typ := model.UploadFile
	ext := filepath.Ext(path)

	switch ext {
	case ".jpg", ".jpeg", ".png":
		typ = model.UploadImage
	case ".mp4":
		typ = model.UploadVideo
	case ".mp3":
		typ = model.UploadAudio
	}

	file, err := os.Open(path)
	if err != nil {
		return "", err
	}

	defer func() { _ = file.Close() }()
	stat, err := file.Stat()
	if err != nil {
		return "", err
	}

	return uploads.Upload(ctx, typ, file, file.Name(), stat.Size())
}
