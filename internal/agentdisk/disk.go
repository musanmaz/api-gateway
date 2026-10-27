package agentdisk

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gateway-control/internal/model"
	"gateway-control/internal/sign"
)

type Disk struct {
	dir    string
	public ed25519.PublicKey
}

func Open(dir string, public ed25519.PublicKey) (*Disk, error) {
	if err := os.MkdirAll(filepath.Join(dir, "accepted"), 0o755); err != nil {
		return nil, err
	}
	return &Disk{dir: dir, public: public}, nil
}

func (d *Disk) Save(env model.Envelope) error {
	body, err := json.Marshal(env)
	if err != nil {
		return err
	}
	name := filepath.Join(d.dir, "accepted", strconv.FormatInt(env.Revision, 10)+".json")
	if err := atomicWrite(name, body); err != nil {
		return err
	}
	return atomicWrite(filepath.Join(d.dir, "current"), []byte(strconv.FormatInt(env.Revision, 10)+"\n"))
}

func (d *Disk) Load() (model.Envelope, model.Package, error) {
	current, err := os.ReadFile(filepath.Join(d.dir, "current"))
	if err == nil {
		revision, convErr := strconv.ParseInt(strings.TrimSpace(string(current)), 10, 64)
		if convErr == nil {
			if env, pkg, loadErr := d.loadRevision(revision); loadErr == nil {
				return env, pkg, nil
			}
		}
	}
	matches, _ := filepath.Glob(filepath.Join(d.dir, "accepted", "*.json"))
	sort.Sort(sort.Reverse(sort.StringSlice(matches)))
	var best *struct {
		env model.Envelope
		pkg model.Package
		rev int64
	}
	for _, path := range matches {
		base := strings.TrimSuffix(filepath.Base(path), ".json")
		revision, convErr := strconv.ParseInt(base, 10, 64)
		if convErr != nil {
			continue
		}
		env, pkg, loadErr := d.loadRevision(revision)
		if loadErr != nil {
			continue
		}
		if best == nil || revision > best.rev {
			best = &struct {
				env model.Envelope
				pkg model.Package
				rev int64
			}{env: env, pkg: pkg, rev: revision}
		}
	}
	if best == nil {
		return model.Envelope{}, model.Package{}, fmt.Errorf("güvenilir yerel paket yok")
	}
	if err := atomicWrite(filepath.Join(d.dir, "current"), []byte(strconv.FormatInt(best.rev, 10)+"\n")); err != nil {
		return model.Envelope{}, model.Package{}, err
	}
	return best.env, best.pkg, nil
}

func (d *Disk) loadRevision(revision int64) (model.Envelope, model.Package, error) {
	raw, err := os.ReadFile(filepath.Join(d.dir, "accepted", strconv.FormatInt(revision, 10)+".json"))
	if err != nil {
		return model.Envelope{}, model.Package{}, err
	}
	var env model.Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return model.Envelope{}, model.Package{}, err
	}
	if err := sign.Verify(d.public, env); err != nil {
		return model.Envelope{}, model.Package{}, err
	}
	var pkg model.Package
	if err := json.Unmarshal(env.Payload, &pkg); err != nil {
		return model.Envelope{}, model.Package{}, err
	}
	if pkg.Revision != revision || pkg.Revision != env.Revision {
		return model.Envelope{}, model.Package{}, fmt.Errorf("sürüm uyuşmuyor")
	}
	return env, pkg, nil
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tempName, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}
