package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
)

// MappingRecord is the canonical identity, target, desired state and auth for one
// mapping. Tokens live in this 0600 file; API readers must redact them.
type MappingRecord struct {
	ID        string       `json:"id"`
	Target    OriginTarget `json:"target"`
	Hostname  string       `json:"hostname"`
	SavedName *string      `json:"savedName,omitempty"`
	Provider  string       `json:"provider"`
	Desired   bool         `json:"desired"`
	Auth      PortAuth     `json:"auth,omitempty"`
}
type MappingStore struct {
	Version  int                      `json:"version"`
	Mappings map[string]MappingRecord `json:"mappings"`
}

var mappingStoreMu sync.Mutex

func MappingStorePath() string {
	return filepath.Join(filepath.Dir(PortMappingDesiredFile), "port-mappings-v2.json")
}
func NewMappingID() (string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "map_" + hex.EncodeToString(b), nil
}

func LoadMappings() (MappingStore, error) {
	mappingStoreMu.Lock()
	defer mappingStoreMu.Unlock()
	return loadMappings()
}
func loadMappings() (MappingStore, error) {
	var s MappingStore
	b, err := os.ReadFile(MappingStorePath())
	if err == nil {
		if err = json.Unmarshal(b, &s); err != nil {
			return s, err
		}
		if s.Version != 2 || s.Mappings == nil {
			return s, fmt.Errorf("unsupported port mapping store")
		}
		for id, r := range s.Mappings {
			if id != r.ID {
				return s, fmt.Errorf("mapping ID mismatch")
			}
			target, e := r.Target.Normalize(0)
			if e != nil {
				return s, e
			}
			r.Target = target
			s.Mappings[id] = r
		}
		return s, nil
	}
	if !os.IsNotExist(err) {
		return s, err
	}
	s = MappingStore{Version: 2, Mappings: map[string]MappingRecord{}}
	names, err := LoadPortMappingNames()
	if err != nil {
		return s, err
	}
	auths, err := LoadPortMappingAuth()
	if err != nil {
		return s, err
	}
	desired, err := LoadPortMappingDesired()
	if err != nil {
		return s, err
	}
	keys := map[string]bool{}
	for k := range names {
		keys[k] = true
	}
	for k := range auths {
		keys[k] = true
	}
	for k := range desired {
		keys[k] = true
	}
	for key := range keys {
		port, e := strconv.Atoi(key)
		if e != nil || port < 1 || port > 65535 {
			return s, fmt.Errorf("invalid legacy mapping port %q", key)
		}
		r := MappingRecord{ID: LocalMappingID(port), Target: OriginTarget{Scheme: "http", Host: "127.0.0.1", Port: port}, Hostname: names[key], Auth: auths[key]}
		if name, ok := names[key]; ok {
			r.SavedName = &name
		}
		if d, ok := desired[key]; ok {
			r.Desired = true
			r.Hostname = d.Hostname
			r.Provider = d.Provider
		}
		s.Mappings[r.ID] = r
	}
	// Keep exact legacy snapshots for recovery; v2 becomes the source of truth.
	for _, path := range []string{PortMappingNamesFile, PortMappingAuthFile, PortMappingDesiredFile} {
		data, e := os.ReadFile(path)
		if os.IsNotExist(e) {
			continue
		}
		if e != nil {
			return s, e
		}
		f, e := os.OpenFile(path+".pre-target-v2", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if os.IsExist(e) {
			continue
		}
		if e != nil {
			return s, e
		}
		_, e = f.Write(data)
		ce := f.Close()
		if e != nil {
			return s, e
		}
		if ce != nil {
			return s, ce
		}
	}
	if err = saveMappings(s); err != nil {
		return s, err
	}
	return s, nil
}
func saveMappings(s MappingStore) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(MappingStorePath())
	if err = os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".port-mappings-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(append(b, '\n'))
	}
	if err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err != nil {
		return err
	}
	if ce != nil {
		return ce
	}
	return os.Rename(name, MappingStorePath())
}
func UpdateMappings(fn func(*MappingStore) error) error {
	mappingStoreMu.Lock()
	defer mappingStoreMu.Unlock()
	s, err := loadMappings()
	if err != nil {
		return err
	}
	if err = fn(&s); err != nil {
		return err
	}
	return saveMappings(s)
}

// ResolveMapping accepts an ID or an unambiguous legacy port selector. Include
// inactive records so deleting an already-down mapping still stops restoration.
func ResolveMapping(s MappingStore, selector string) (MappingRecord, error) {
	if r, ok := s.Mappings[selector]; ok {
		return r, nil
	}
	port, err := strconv.Atoi(selector)
	if err != nil {
		return MappingRecord{}, fmt.Errorf("mapping %q not found", selector)
	}
	var found []MappingRecord
	for _, r := range s.Mappings {
		if r.Target.Port == port && r.Desired {
			found = append(found, r)
		}
	}
	if len(found) == 0 {
		for _, r := range s.Mappings {
			if r.Target.Port == port {
				found = append(found, r)
			}
		}
	}
	if len(found) > 1 {
		return MappingRecord{}, fmt.Errorf("port %d matches %d mappings; specify a mapping ID", port, len(found))
	}
	if len(found) == 0 {
		return MappingRecord{}, fmt.Errorf("port %d is not being forwarded", port)
	}
	return found[0], nil
}
