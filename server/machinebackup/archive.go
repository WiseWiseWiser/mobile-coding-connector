package machinebackup

import (
	"archive/tar"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ulikunitz/xz"
)

type archiveEntry struct {
	Header *tar.Header
	Data   []byte
}

// ReadArchive decompresses tar.xz and returns manifest plus file entries.
func ReadArchive(r io.Reader) (*Manifest, []archiveEntry, error) {
	xzr, err := xz.NewReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("xz reader: %w", err)
	}
	tr := tar.NewReader(xzr)

	var manifest *Manifest
	var entries []archiveEntry
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("read tar: %w", err)
		}
		name := normalizeRelPath(hdr.Name)
		if name == "" {
			continue
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg, tar.TypeRegA, tar.TypeSymlink, tar.TypeLink:
			if name == "manifest.json" {
				data, err := io.ReadAll(tr)
				if err != nil {
					return nil, nil, fmt.Errorf("read manifest: %w", err)
				}
				manifest, err = parseManifest(data)
				if err != nil {
					return nil, nil, err
				}
				continue
			}
			var data []byte
			if hdr.Typeflag == tar.TypeReg || hdr.Typeflag == tar.TypeRegA {
				data, err = io.ReadAll(tr)
				if err != nil {
					return nil, nil, fmt.Errorf("read %s: %w", name, err)
				}
			}
			entries = append(entries, archiveEntry{Header: hdr, Data: data})
		default:
			return nil, nil, fmt.Errorf("unsupported tar entry %s type %c", name, hdr.Typeflag)
		}
	}
	if manifest == nil {
		return nil, nil, fmt.Errorf("archive missing manifest.json")
	}
	return manifest, entries, nil
}

func parseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}