package kindle

import (
	metadata "Kindria/internal/core/api/books"
	"Kindria/internal/utils"
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	errKindleNotFound = errors.New("kindle mtp mount not found")
	mtpRootRe         = regexp.MustCompile(`default_location=(mtp://Amazon_Kindle_[^/]+/)`)
)

type SyncResult struct {
	DetectedBooks []string
	Inserted      int
	Failed        int
	Duplicated    int
	Refreshed     []*metadata.Package
}

func DetectKindleRootURI(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "gio", "mount", "-li")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gio mount -li failed: %w\n%s", err, string(out))
	}

	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		m := mtpRootRe.FindStringSubmatch(line)
		if len(m) == 2 {
			return m[1], nil
		}
	}
	return "", errKindleNotFound
}

func KindleDocumentsURI(root string) string {
	root = strings.TrimRight(root, "/")
	return root + "/Internal Storage/documents/"
}

func ListKindleFiles(ctx context.Context, docsURI string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "gio", "list", docsURI)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gio list failed: %w\n%s", err, string(out))
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	files := make([]string, 0, len(lines))
	for _, l := range lines {
		name := strings.TrimSpace(l)
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		files = append(files, name)
	}
	return files, nil
}

func CopyFromKindle(ctx context.Context, srcURI, localDstPath string) error {
	absDst, err := filepath.Abs(localDstPath)
	if err != nil {
		return err
	}
	dstURI := (&url.URL{Scheme: "file", Path: absDst}).String()

	cmd := exec.CommandContext(ctx, "gio", "copy", "--", srcURI, dstURI)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gio copy failed: %w\n%s", err, string(out))
	}
	return nil
}

func JoinMTP(baseURI, fileName string) string {
	baseURI = strings.TrimRight(baseURI, "/")
	return baseURI + "/" + url.PathEscape(fileName)
}

func FilterConvertibleBooks(entries []string) []string {
	allowed := map[string]struct{}{
		".epub": {},
		".azw":  {},
		".azw3": {},
		".mobi": {},
		".pdf":  {},
		".txt":  {},
	}
	filtered := make([]string, 0, len(entries))
	for _, e := range entries {
		ext := strings.ToLower(filepath.Ext(e))
		if _, ok := allowed[ext]; ok {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func ScanKindleBooks(ctx context.Context) (docsURI string, books []string, err error) {
	root, err := DetectKindleRootURI(ctx)
	if err != nil {
		return "", nil, err
	}
	docsURI = KindleDocumentsURI(root)
	entries, err := ListKindleFiles(ctx, docsURI)
	if err != nil {
		return "", nil, err
	}
	return docsURI, FilterConvertibleBooks(entries), nil
}

func KindleExtract(ctx context.Context, h *metadata.Handler, docsURI string, selected []string) (SyncResult, error) {
	var result SyncResult
	if docsURI == "" {
		root, err := DetectKindleRootURI(ctx)
		if err != nil {
			return result, err
		}
		docsURI = KindleDocumentsURI(root)
	}

	detected, err := ListKindleFiles(ctx, docsURI)
	if err != nil {
		return result, err
	}
	detected = FilterConvertibleBooks(detected)
	result.DetectedBooks = detected

	target := selected
	if len(target) == 0 {
		target = detected
	}

	booksFolder, err := os.ReadDir("./books")
	if err != nil {
		return result, err
	}
	existingNames := make(map[string]struct{}, len(booksFolder))
	for _, b := range booksFolder {
		existingNames[b.Name()] = struct{}{}
	}

	tmpDir, err := os.MkdirTemp("", "kindria-kindle-sync-*")
	if err != nil {
		return result, err
	}
	defer os.RemoveAll(tmpDir)

	copied := 0
	for _, name := range target {
		srcURI := JoinMTP(docsURI, name)
		localSrc := filepath.Join(tmpDir, name)

		if err := CopyFromKindle(ctx, srcURI, localSrc); err != nil {
			result.Failed++
			continue
		}

		outputName := name
		finalSrc := localSrc
		if strings.ToLower(filepath.Ext(name)) != ".epub" {
			outputName = strings.TrimSuffix(name, filepath.Ext(name)) + ".epub"
			converted := filepath.Join(tmpDir, outputName)
			if err := convertToEPUB(ctx, localSrc, converted); err != nil {
				result.Failed++
				continue
			}
			finalSrc = converted
		}

		exists, err := h.CheckBookExist(outputName)
		if err != nil {
			result.Failed++
			continue
		}
		if exists != 0 {
			result.Duplicated++
			continue
		}
		if _, ok := existingNames[outputName]; ok {
			result.Duplicated++
			continue
		}

		if err := utils.CopyFile(finalSrc, "./books/"+outputName); err != nil {
			result.Failed++
			continue
		}

		existingNames[outputName] = struct{}{}
		copied++
	}

	if copied == 0 {
		return result, nil
	}

	insertedRows, err := h.InsertBooks()
	if err != nil {
		return result, err
	}
	result.Inserted = len(insertedRows)
	refreshed, err := h.SelectBooks()
	if err != nil {
		return result, err
	}
	result.Refreshed = refreshed
	return result, nil
}

func convertToEPUB(ctx context.Context, src, dst string) error {
	cmd := exec.CommandContext(ctx, "ebook-convert", src, dst)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ebook-convert failed: %w\n%s", err, string(out))
	}
	return nil
}
