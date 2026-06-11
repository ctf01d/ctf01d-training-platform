package ctf01d

import (
	"fmt"
	"strings"
	"time"
)

type GameParams struct {
	ID                  string
	Name                string
	StartUTC            time.Time
	EndUTC              time.Time
	CoffeeBreakStartUTC *time.Time
	CoffeeBreakEndUTC   *time.Time
	FlagTTLMin          int
	BasicAttackCost     int
	DefenceCost         float64
}

type ScoreboardParams struct {
	Port       int
	HtmlFolder string
	Random     bool
}

type TeamParams struct {
	ID          string
	Name        string
	Active      bool
	IPAddress   string
	LogoRel     string
	LogoSrc     string
	LogoURL     string
	Ctf01dExtra map[string]string
}

type CheckerFile struct {
	Src string
	Rel string
}

type CheckerParams struct {
	ID              string
	Name            string
	Enabled         bool
	ScriptWait      int
	RoundSleep      int
	ScriptRel       string
	Files           []CheckerFile
	BundlePath      string
	CheckerFromBundle bool
}

type Options struct {
	Prefix          string
	IncludeHTML     bool
	HtmlSourcePath  string
	IncludeCompose  bool
	ComposeProject  string
	Warnings        []string
}

type ExportError struct {
	Errors []string
}

func (e *ExportError) Error() string {
	return fmt.Sprintf("export errors: %s", strings.Join(e.Errors, "; "))
}

func NewExportError(msgs ...string) *ExportError {
	return &ExportError{Errors: msgs}
}
