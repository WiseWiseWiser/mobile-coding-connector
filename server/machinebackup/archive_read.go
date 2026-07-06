package machinebackup

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ulikunitz/xz"
)

// ReadArchiveConfig returns .backup/config.json from an archive, or nil if absent.
func ReadArchiveConfig(r io.Reader) (*ExclusionConfig, error) {
	data, err := readArchiveFile(r, backupMetaDir+"/"+metaConfigName)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil
	}
	var cfg ExclusionConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse archive config: %w", err)
	}
	return &cfg, nil
}

// ReadArchiveMeta returns .backup meta files except config.json and *.machine.bak.
func ReadArchiveMeta(r io.Reader) (map[string][]byte, error) {
	xzr, err := xz.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("xz reader: %w", err)
	}
	tr := tar.NewReader(xzr)

	out := make(map[string][]byte)
	prefix := backupMetaDir + "/"
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		name := normalizeRelPath(hdr.Name)
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		base := strings.TrimPrefix(name, prefix)
		if base == "" || base == metaConfigName || strings.HasSuffix(base, machineBakSuffix) {
			continue
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", name, err)
		}
		out[base] = data
	}
	return out, nil
}

func readArchiveFile(r io.Reader, want string) ([]byte, error) {
	xzr, err := xz.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("xz reader: %w", err)
	}
	tr := tar.NewReader(xzr)
	want = normalizeRelPath(want)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if normalizeRelPath(hdr.Name) != want {
			continue
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			return nil, fmt.Errorf("archive entry %s is not a regular file", want)
		}
		return io.ReadAll(tr)
	}
}