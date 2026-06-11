package services

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

const (
	maxEntryBytes = 50 * 1024 * 1024
	maxTotalBytes = 200 * 1024 * 1024
	maxFiles      = 10000
	maxMetaBytes  = 512 * 1024
	maxTextBytes  = 512 * 1024
)

type BundleMetadata struct {
	Name              string
	PublicDescription string
	Copyright         string
	License           string
	Ctf01dTraining    json.RawMessage
}

func safeRelPath(rel string) string {
	s := strings.ReplaceAll(rel, "\\", "/")
	s = strings.TrimPrefix(s, "/")
	s = strings.TrimSuffix(s, "/")
	if s == "" {
		return ""
	}
	if strings.Contains(s, "\x00") {
		return ""
	}
	segments := strings.Split(s, "/")
	for _, seg := range segments {
		if seg == "" || seg == "." || seg == ".." {
			return ""
		}
	}
	return s
}

func detectRootPrefix(zipReader *zip.Reader) string {
	seen := make(map[string]bool)
	for _, f := range zipReader.File {
		n := strings.TrimPrefix(f.Name, "/")
		if n == "" {
			continue
		}
		parts := strings.SplitN(n, "/", 2)
		seg := parts[0]
		if seg == "" {
			continue
		}
		seen[seg] = true
	}
	if len(seen) != 1 {
		return ""
	}
	for seg := range seen {
		if seg == "service" || seg == "checker" {
			return ""
		}
		return seg + "/"
	}
	return ""
}

var (
	readmeCandidates  = []string{"README.md", "readme.md", "README", "readme"}
	licenseCandidates = []string{
		"LICENSE", "LICENSE.txt", "LICENSE.md",
		"LICENCE", "LICENCE.txt",
		"COPYING", "COPYING.txt",
	}
)

func readFirstFromZip(zr *zip.Reader, rootPrefix string, candidates []string) (string, []byte) {
	for _, name := range candidates {
		fullPath := rootPrefix + name
		for _, f := range zr.File {
			if f.Name == fullPath && !f.FileInfo().IsDir() {
				data, err := readSmallZipEntry(f, maxMetaBytes)
				if err == nil {
					return name, data
				}
			}
		}
	}
	return "", nil
}

func readSmallZipEntry(f *zip.File, maxBytes int64) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, maxBytes))
}

func readEntryFromZip(zr *zip.Reader, entryName string) []byte {
	for _, f := range zr.File {
		if f.Name == entryName && !f.FileInfo().IsDir() {
			data, err := readSmallZipEntry(f, maxTextBytes)
			if err == nil {
				return data
			}
		}
	}
	return nil
}

func readLicenseFromZip(zr *zip.Reader) []byte {
	for _, name := range licenseCandidates {
		fullPath := "service/" + name
		if data := readEntryFromZip(zr, fullPath); data != nil {
			return data
		}
	}
	return nil
}

func readTrainingJSONFromZip(zr *zip.Reader) []byte {
	if data := readEntryFromZip(zr, "service/ctf01d-training.json"); data != nil {
		return data
	}
	var fallback *zip.File
	for _, f := range zr.File {
		if strings.HasSuffix(f.Name, "ctf01d-training.json") && !strings.HasPrefix(f.Name, "checker/") {
			if fallback == nil || len(f.Name) < len(fallback.Name) {
				fallback = f
			}
		}
	}
	if fallback != nil {
		data, err := readSmallZipEntry(fallback, maxTextBytes)
		if err == nil {
			return data
		}
	}
	return nil
}

func readReadmeFromZip(zr *zip.Reader) []byte {
	for _, name := range readmeCandidates {
		if data := readEntryFromZip(zr, "service/"+name); data != nil {
			return data
		}
	}
	return nil
}

func ExtractMetadata(bundleZipBytes []byte) (*BundleMetadata, error) {
	r, err := zip.NewReader(bytes.NewReader(bundleZipBytes), int64(len(bundleZipBytes)))
	if err != nil {
		return nil, fmt.Errorf("reading bundle zip: %w", err)
	}

	readme := readReadmeFromZip(r)
	licenseText := readLicenseFromZip(r)
	trainingJSON := readTrainingJSONFromZip(r)

	var training map[string]interface{}
	if trainingJSON != nil {
		_ = json.Unmarshal(trainingJSON, &training)
	}

	meta := &BundleMetadata{}

	if training != nil {
		if dn, ok := training["display_name"].(string); ok && strings.TrimSpace(dn) != "" {
			meta.Name = strings.TrimSpace(dn)
		}
		if desc, ok := training["description"].(string); ok && strings.TrimSpace(desc) != "" {
			meta.PublicDescription = strings.TrimSpace(desc)
		}
		meta.Ctf01dTraining = trainingJSON
	}

	if meta.Name == "" && readme != nil {
		meta.Name = extractTitle(readme)
	}
	if meta.PublicDescription == "" && readme != nil {
		meta.PublicDescription = summarizeMarkdown(readme)
	}
	if licenseText != nil {
		meta.License = detectLicense(string(licenseText))
		meta.Copyright = extractCopyright(string(licenseText))
	}

	return meta, nil
}

func BuildBundle(zipBytes []byte) ([]byte, error) {
	if len(zipBytes) == 0 {
		return nil, fmt.Errorf("empty zip")
	}

	srcReader, err := zip.NewReader(bytes.NewReader(zipBytes), int64(len(zipBytes)))
	if err != nil {
		return nil, fmt.Errorf("reading zip: %w", err)
	}

	rootPrefix := detectRootPrefix(srcReader)
	servicePrefix := rootPrefix + "service/"
	checkerPrefix := rootPrefix + "checker/"

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	var totalBytes int64
	fileCount := 0
	serviceFound := false

	hasService := false
	hasChecker := false
	for _, f := range srcReader.File {
		if strings.HasPrefix(f.Name, servicePrefix) {
			hasService = true
		}
		if strings.HasPrefix(f.Name, checkerPrefix) {
			hasChecker = true
		}
	}

	_, rootReadme := readFirstFromZip(srcReader, rootPrefix, readmeCandidates)
	_, rootLicense := readFirstFromZip(srcReader, rootPrefix, licenseCandidates)
	_, rootTrainingJSON := readFirstFromZip(srcReader, rootPrefix, []string{"ctf01d-training.json"})

	serviceHasReadme := false
	serviceHasLicense := false
	serviceHasTrainingJSON := false

	cb := func(rel string) {
		if !serviceHasReadme {
			for _, n := range readmeCandidates {
				if strings.EqualFold(rel, n) {
					serviceHasReadme = true
					break
				}
			}
		}
		if !serviceHasLicense {
			for _, n := range licenseCandidates {
				if strings.EqualFold(rel, n) {
					serviceHasLicense = true
					break
				}
			}
		}
		if !serviceHasTrainingJSON {
			if strings.EqualFold(rel, "ctf01d-training.json") {
				serviceHasTrainingJSON = true
			}
		}
	}

	if hasService {
		serviceFound, err = copyTree(srcReader, w, servicePrefix, "service/", nil, &totalBytes, &fileCount, cb)
		if err != nil {
			return nil, err
		}
	} else {
		var excludes []string
		if hasChecker {
			excludes = []string{"checker/"}
		}
		serviceFound, err = copyTree(srcReader, w, rootPrefix, "service/", excludes, &totalBytes, &fileCount, cb)
		if err != nil {
			return nil, err
		}
	}

	if hasChecker {
		if _, err = copyTree(srcReader, w, checkerPrefix, "checker/", nil, &totalBytes, &fileCount, nil); err != nil {
			return nil, err
		}
	}

	if rootReadme != nil && !serviceHasReadme {
		f, err := w.Create("service/README.md")
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(rootReadme); err != nil {
			return nil, err
		}
	}

	if rootLicense != nil && !serviceHasLicense {
		f, err := w.Create("service/LICENSE")
		if err != nil {
			return nil, err
		}
		if _, err := f.Write(rootLicense); err != nil {
			return nil, err
		}
	}

	if rootTrainingJSON != nil && !serviceHasTrainingJSON {
		f, err := w.Create("service/ctf01d-training.json")
		if err != nil {
			return nil, err
		}
		limit := rootTrainingJSON
		if len(limit) > maxMetaBytes {
			limit = limit[:maxMetaBytes]
		}
		if _, err := f.Write(limit); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	if !serviceFound {
		return nil, fmt.Errorf("no service content found in archive")
	}

	return buf.Bytes(), nil
}

func copyTree(src *zip.Reader, dst *zip.Writer, fromPrefix, toPrefix string, excludeRelPrefixes []string, totalBytes *int64, fileCount *int, onRel func(string)) (bool, error) {
	found := false
	for _, entry := range src.File {
		if !strings.HasPrefix(entry.Name, fromPrefix) {
			continue
		}
		rel := strings.TrimPrefix(entry.Name, fromPrefix)
		rel = safeRelPath(rel)
		if rel == "" {
			continue
		}
		skip := false
		for _, p := range excludeRelPrefixes {
			cleanP := strings.TrimSuffix(p, "/")
			if rel == cleanP || strings.HasPrefix(rel, cleanP+"/") {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		if strings.HasPrefix(rel, ".git/") {
			continue
		}
		if onRel != nil {
			onRel(rel)
		}
		if entry.FileInfo().IsDir() {
			dirRel := rel
			if !strings.HasSuffix(dirRel, "/") {
				dirRel += "/"
			}
			if _, err := dst.Create(toPrefix + dirRel); err != nil {
				return false, err
			}
			continue
		}

		*fileCount++
		if *fileCount > maxFiles {
			return false, fmt.Errorf("too many files in archive")
		}

		uncompressedSize := int64(entry.UncompressedSize64)
		if uncompressedSize > maxEntryBytes {
			return false, fmt.Errorf("file in archive too large (%d bytes)", uncompressedSize)
		}
		*totalBytes += uncompressedSize
		if *totalBytes > maxTotalBytes {
			return false, fmt.Errorf("archive too large (total %d bytes)", *totalBytes)
		}

		rc, err := entry.Open()
		if err != nil {
			return false, fmt.Errorf("opening entry %s: %w", entry.Name, err)
		}
		data, err := io.ReadAll(io.LimitReader(rc, maxEntryBytes+1))
		rc.Close()
		if err != nil {
			return false, fmt.Errorf("reading entry %s: %w", entry.Name, err)
		}

		out, err := dst.Create(toPrefix + rel)
		if err != nil {
			return false, err
		}
		if _, err := out.Write(data); err != nil {
			return false, err
		}
		found = true
	}
	return found, nil
}

func extractCheckerFromBundle(bundleBytes []byte) []byte {
	r, err := zip.NewReader(bytes.NewReader(bundleBytes), int64(len(bundleBytes)))
	if err != nil {
		return nil
	}

	hasChecker := false
	for _, f := range r.File {
		if strings.HasPrefix(f.Name, "checker/") && !f.FileInfo().IsDir() {
			hasChecker = true
			break
		}
	}
	if !hasChecker {
		return nil
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "checker/") {
			continue
		}
		if f.FileInfo().IsDir() {
			if _, err := w.Create(f.Name); err != nil {
				return nil
			}
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil
		}
		data, err := io.ReadAll(io.LimitReader(rc, maxCheckerEntryBytes))
		rc.Close()
		if err != nil {
			return nil
		}
		out, err := w.Create(f.Name)
		if err != nil {
			return nil
		}
		if _, err := out.Write(data); err != nil {
			return nil
		}
	}
	if err := w.Close(); err != nil {
		return nil
	}
	return buf.Bytes()
}

var (
	linkRe = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
	codeRe = regexp.MustCompile("`+([^`]+)`+")
)

func extractTitle(md []byte) string {
	text := string(md)
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			title := strings.TrimPrefix(trimmed, "# ")
			title = linkRe.ReplaceAllString(title, "$1")
			title = codeRe.ReplaceAllString(title, "$1")
			return strings.TrimSpace(title)
		}
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			title := strings.TrimPrefix(trimmed, "## ")
			title = linkRe.ReplaceAllString(title, "$1")
			title = codeRe.ReplaceAllString(title, "$1")
			return strings.TrimSpace(title)
		}
	}
	return ""
}

func summarizeMarkdown(md []byte) string {
	text := string(md)
	text = linkRe.ReplaceAllString(text, "$1")
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		lines = append(lines, l)
	}
	result := strings.TrimSpace(strings.Join(lines, "\n"))
	if len(result) > 400 {
		result = result[:400]
	}
	return result
}

func extractCopyright(text string) string {
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)
		if !strings.Contains(lower, "copyright") && !strings.Contains(lower, "(c)") && !strings.Contains(trimmed, "©") {
			continue
		}
		v := trimmed
		re1 := regexp.MustCompile(`(?i)^(copyright\s+)?\(c\)\s*`)
		v = re1.ReplaceAllString(v, "")
		re2 := regexp.MustCompile(`^(Copyright\s+)?©\s*`)
		v = re2.ReplaceAllString(v, "")
		re3 := regexp.MustCompile(`(?i)^copyright\s+`)
		v = re3.ReplaceAllString(v, "")
		v = strings.Join(strings.Fields(v), " ")
		if len(v) > 200 {
			v = v[:200]
		}
		return v
	}
	return ""
}

func detectLicense(text string) string {
	t := strings.ToLower(text)
	if t == "" {
		return ""
	}
	if strings.Contains(t, "mit license") || strings.Contains(t, "permission is hereby granted, free of charge") {
		return "MIT"
	}
	if (strings.Contains(t, "apache license") && strings.Contains(t, "version 2.0")) || strings.Contains(t, "apache-2.0") {
		return "Apache-2.0"
	}
	if strings.Contains(t, "redistribution and use in source and binary forms") {
		if strings.Contains(t, "neither the name of the") || strings.Contains(t, "neither the name nor the names") {
			return "BSD-3-Clause"
		}
		return "BSD-2-Clause"
	}
	if strings.Contains(t, "gnu general public license") {
		if strings.Contains(t, "version 3") || strings.Contains(t, "gpl version 3") || strings.Contains(t, "gplv3") {
			return "GPL-3.0"
		}
		if strings.Contains(t, "version 2") || strings.Contains(t, "gpl version 2") || strings.Contains(t, "gplv2") {
			return "GPL-2.0"
		}
		return "GPL"
	}
	if strings.Contains(t, "gnu lesser general public license") || strings.Contains(t, "lgpl") {
		if strings.Contains(t, "version 3") || strings.Contains(t, "lgplv3") {
			return "LGPL-3.0"
		}
		return "LGPL"
	}
	if strings.Contains(t, "mozilla public license") {
		return "MPL-2.0"
	}
	if strings.Contains(t, "isc license") || (strings.Contains(t, "permission to use, copy, modify") && strings.Contains(t, "the author and contributors")) {
		return "ISC"
	}
	if strings.Contains(t, "this is free and unencumbered software released into the public domain") || strings.Contains(t, "unlicense") {
		return "Unlicense"
	}
	return ""
}

func computeSHA256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func validateZipBytes(data []byte) error {
	if len(data) < 4 {
		return fmt.Errorf("data too short to be a valid zip")
	}
	if !bytes.Equal(data[:4], zipMagic) {
		return fmt.Errorf("not a valid ZIP archive")
	}
	return nil
}
