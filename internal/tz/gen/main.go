// SPDX-License-Identifier: MIT

//go:build ignore

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	pinnedRelease  = "2025b"
	releaseURLFmt  = "https://data.iana.org/time-zones/releases/tzdata%s.tar.gz"
	fetchTimeout   = 2 * time.Minute
	compileTimeout = time.Minute
	outFilePerm    = 0o644
	sourceFilePerm = 0o644
)

var primaryZoneFiles = []string{
	"africa", "antarctica", "asia", "australasia",
	"etcetera", "europe", "northamerica", "southamerica",
}

var (
	labelPattern = regexp.MustCompile(`^[A-Za-z_+-]+(?:/[A-Za-z0-9_+-]+)+$`)
	posixPattern = regexp.MustCompile(`^` +
		`(?:[A-Za-z]+|<[+\-0-9A-Za-z]+>)[+-]?\d{1,3}(?::\d{2}){0,2}` +
		`(?:(?:[A-Za-z]+|<[+\-0-9A-Za-z]+>)(?:[+-]?\d{1,3}(?::\d{2}){0,2})?` +
		`(?:,(?:J\d{1,3}|\d{1,3}|M\d{1,2}\.\d\.\d)(?:/[+-]?\d{1,3}(?::\d{2}){0,2})?` +
		`,(?:J\d{1,3}|\d{1,3}|M\d{1,2}\.\d\.\d)(?:/[+-]?\d{1,3}(?::\d{2}){0,2})?)?)?$`)
)

var errNoFooter = errors.New("no POSIX TZ footer")

func main() {
	version := flag.String("version", pinnedRelease, "IANA tzdata release, e.g. 2025b")
	out := flag.String("out", "timezones.json", "output path for the generated table")
	flag.Parse()

	if err := run(*version, *out); err != nil {
		fmt.Fprintf(os.Stderr, "gen: %v\n", err)
		os.Exit(1)
	}
}

func run(version, out string) error {
	if _, err := exec.LookPath("zic"); err != nil {
		return fmt.Errorf("zic not found on PATH, install the tzdata/tzcode package: %w", err)
	}

	fetchCtx, cancelFetch := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancelFetch()
	srcDir, err := fetchSource(fetchCtx, version)
	if err != nil {
		return err
	}
	defer os.RemoveAll(srcDir)

	compileCtx, cancelCompile := context.WithTimeout(context.Background(), compileTimeout)
	defer cancelCompile()
	zoneDir, err := compileZones(compileCtx, srcDir)
	if err != nil {
		return err
	}
	defer os.RemoveAll(zoneDir)

	table, err := harvest(zoneDir)
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(table, "", "\t")
	if err != nil {
		return fmt.Errorf("encode table: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(out, data, outFilePerm); err != nil {
		return fmt.Errorf("write %s: %w", out, err)
	}

	fmt.Printf("gen: wrote %d zones from tzdata %s to %s\n", len(table), version, out)
	return nil
}

func fetchSource(ctx context.Context, version string) (string, error) {
	url := fmt.Sprintf(releaseURLFmt, version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request for %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch %s: status %s", url, resp.Status)
	}

	dir, err := os.MkdirTemp("", "tzdata-src-")
	if err != nil {
		return "", fmt.Errorf("create source dir: %w", err)
	}

	if err := extractTarGz(resp.Body, dir); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("extract %s: %w", url, err)
	}
	return dir, nil
}

func extractTarGz(r io.Reader, dir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("open gzip stream: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read tar entry: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}

		name := filepath.Base(filepath.Clean(hdr.Name))
		path := filepath.Join(dir, name)
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, sourceFilePerm)
		if err != nil {
			return fmt.Errorf("create %s: %w", path, err)
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return fmt.Errorf("write %s: %w", path, err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("close %s: %w", path, err)
		}
	}
}

func compileZones(ctx context.Context, srcDir string) (string, error) {
	zoneDir, err := os.MkdirTemp("", "tzdata-zoneinfo-")
	if err != nil {
		return "", fmt.Errorf("create zoneinfo dir: %w", err)
	}

	var sources []string
	for _, name := range primaryZoneFiles {
		if _, err := os.Stat(filepath.Join(srcDir, name)); err == nil {
			sources = append(sources, name)
		}
	}
	if len(sources) == 0 {
		os.RemoveAll(zoneDir)
		return "", fmt.Errorf("no zone source files found in %s", srcDir)
	}

	args := append([]string{"-d", zoneDir}, sources...)
	cmd := exec.CommandContext(ctx, "zic", args...)
	cmd.Dir = srcDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		os.RemoveAll(zoneDir)
		return "", fmt.Errorf("zic %s: %w: %s", strings.Join(args, " "), err, stderr.String())
	}
	return zoneDir, nil
}

func harvest(zoneDir string) (map[string]string, error) {
	table := map[string]string{}
	err := filepath.WalkDir(zoneDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(zoneDir, path)
		if err != nil {
			return err
		}
		label := filepath.ToSlash(rel)
		if !labelPattern.MatchString(label) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		posix, err := tzFooter(data)
		if err != nil {
			return fmt.Errorf("%s: %w", label, err)
		}
		if !posixPattern.MatchString(posix) {
			return fmt.Errorf("%s: implausible POSIX TZ string %q", label, posix)
		}
		table[label] = posix
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(table) == 0 {
		return nil, fmt.Errorf("no zones harvested from %s", zoneDir)
	}
	return table, nil
}

func tzFooter(data []byte) (string, error) {
	if !bytes.HasPrefix(data, []byte("TZif")) {
		return "", fmt.Errorf("not a TZif file")
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		return "", errNoFooter
	}
	body := data[:len(data)-1]
	i := bytes.LastIndexByte(body, '\n')
	if i < 0 {
		return "", errNoFooter
	}
	return string(body[i+1:]), nil
}
