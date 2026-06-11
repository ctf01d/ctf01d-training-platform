package services

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ctf01d/ctf01d-training-platform/internal/repository/db"
)

var requiredCodes = []string{"101", "102", "103", "104"}

const maxCheckerEntryBytes = 2 * 1024 * 1024

type CheckerQuerier interface {
	GetServiceByID(ctx context.Context, id int64) (db.Service, error)
	SetCheckStatus(ctx context.Context, arg db.SetCheckStatusParams) (db.Service, error)
}

type CheckerService struct {
	q CheckerQuerier
}

func NewCheckerService(q CheckerQuerier) *CheckerService {
	return &CheckerService{q: q}
}

func (cs *CheckerService) CheckChecker(ctx context.Context, id int64, isAdmin bool) (*ServiceModel, error) {
	svc, err := cs.q.GetServiceByID(ctx, id)
	if err != nil {
		return nil, mapNotFound(err, "service")
	}

	if svc.CheckerLocalPath == nil || *svc.CheckerLocalPath == "" {
		status := "unknown"
		svc, err = cs.q.SetCheckStatus(ctx, db.SetCheckStatusParams{
			ID:          id,
			CheckStatus: status,
			CheckedAt:   pgtypeTz(time.Now()),
		})
		if err != nil {
			return nil, err
		}
		result := fromDB(svc, isAdmin)
		return &result, nil
	}

	result := inspectCheckerArchive(svc.CheckerLocalPath)
	status := result.Status

	svc, err = cs.q.SetCheckStatus(ctx, db.SetCheckStatusParams{
		ID:          id,
		CheckStatus: status,
		CheckedAt:   pgtypeTz(time.Now()),
	})
	if err != nil {
		return nil, err
	}

	model := fromDB(svc, isAdmin)
	return &model, nil
}

func inspectCheckerArchive(checkerPath *string) CheckerInspectionResult {
	return CheckerInspectionResult{
		Status:     "unknown",
		FoundCodes: nil,
	}
}

func InspectCheckerFromBytes(data []byte) CheckerInspectionResult {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return CheckerInspectionResult{Status: "unknown"}
	}

	hasChecker := false
	found := make(map[string]bool)

	for _, f := range r.File {
		name := f.Name
		if !hasCheckerDir(name) {
			continue
		}
		hasChecker = true
		if f.FileInfo().IsDir() {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			continue
		}
		var buf bytes.Buffer
		lr := &limitedReader{r: rc, n: maxCheckerEntryBytes}
		_, _ = buf.ReadFrom(lr)
		rc.Close()

		content := buf.String()
		for _, code := range requiredCodes {
			if !found[code] && containsToken(content, code) {
				found[code] = true
			}
		}
		if len(found) == len(requiredCodes) {
			break
		}
	}

	if !hasChecker {
		return CheckerInspectionResult{Status: "missing", FoundCodes: nil}
	}

	var foundCodes []string
	for _, code := range requiredCodes {
		if found[code] {
			foundCodes = append(foundCodes, code)
		}
	}

	if len(found) == len(requiredCodes) {
		return CheckerInspectionResult{Status: "codes", FoundCodes: requiredCodes}
	}
	return CheckerInspectionResult{Status: "present", FoundCodes: foundCodes}
}

func hasCheckerDir(name string) bool {
	return strings.HasPrefix(name, "checker/") || strings.Contains(name, "/checker/")
}

func containsToken(data string, token string) bool {
	if data == "" || token == "" {
		return false
	}
	i := 0
	for {
		pos := strings.Index(data[i:], token)
		if pos < 0 {
			return false
		}
		pos += i

		beforeOK := pos == 0 || !isDigit(data[pos-1])
		afterIdx := pos + len(token)
		afterOK := afterIdx >= len(data) || !isDigit(data[afterIdx])

		if beforeOK && afterOK {
			return true
		}
		i = pos + 1
	}
}

func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

type limitedReader struct {
	r interface{ Read([]byte) (int, error) }
	n int64
}

func (l *limitedReader) Read(p []byte) (int, error) {
	if l.n <= 0 {
		return 0, fmt.Errorf("limit exceeded")
	}
	if int64(len(p)) > l.n {
		p = p[:l.n]
	}
	n, err := l.r.Read(p)
	l.n -= int64(n)
	return n, err
}

type CheckerInspectionResult struct {
	Status     string
	FoundCodes []string
}
