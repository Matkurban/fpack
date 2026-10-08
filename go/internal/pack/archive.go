package pack

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Zip writes the directory src into dest. When prefix is non-empty, entries
// are stored under that top-level folder. Symlinks are stored as links
// (Unix) and file modes are preserved.
func Zip(src, dest, prefix string) error {
	tmp := dest + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	zw := zip.NewWriter(f)
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if rel == "." {
			return nil
		}
		name := filepath.ToSlash(rel)
		if prefix != "" {
			name = prefix + "/" + name
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = name
		if d.IsDir() {
			h.Name += "/"
			_, err = zw.CreateHeader(h)
			return err
		}
		h.Method = zip.Deflate
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			h.Method = zip.Store
			w, err := zw.CreateHeader(h)
			if err != nil {
				return err
			}
			_, err = w.Write([]byte(target))
			return err
		}
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(w, in)
		return err
	})
	if cerr := zw.Close(); err == nil {
		err = cerr
	}
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

// TarGz writes src into dest (.tar.gz) under the top-level folder prefix.
func TarGz(src, dest, prefix string) error {
	tmp := dest + ".partial"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	err = filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		name := filepath.ToSlash(rel)
		if prefix != "" {
			if rel == "." {
				name = prefix
			} else {
				name = prefix + "/" + name
			}
		} else if rel == "." {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		link := ""
		if info.Mode()&os.ModeSymlink != 0 {
			if link, err = os.Readlink(path); err != nil {
				return err
			}
		}
		h, err := tar.FileInfoHeader(info, link)
		if err != nil {
			return err
		}
		h.Name = name
		if d.IsDir() {
			h.Name += "/"
		}
		h.Uid, h.Gid, h.Uname, h.Gname = 0, 0, "", ""
		if err := tw.WriteHeader(h); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			if _, err := io.Copy(tw, in); err != nil {
				return err
			}
		}
		return nil
	})
	for _, c := range []io.Closer{tw, gz, f} {
		if cerr := c.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

// SHA256File hashes a file.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ChecksumsFile is the default checksum list written to the output dir.
const ChecksumsFile = "SHA256SUMS"

// ChecksumFiles maps algorithms to their list file names.
var ChecksumFiles = map[string]string{"sha256": "SHA256SUMS", "sha512": "SHA512SUMS"}

// HashFile returns the hex digest of a file (sha256 or sha512).
func HashFile(path, alg string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var h hash.Hash = sha256.New()
	if alg == "sha512" {
		h = sha512.New()
	}
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// WriteChecksums (re)writes dir/SHA256SUMS (see WriteChecksumsAlg).
func WriteChecksums(dir string) (string, map[string]string, error) {
	return WriteChecksumsAlg(dir, "sha256")
}

// WriteChecksumsAlg (re)writes dir/SHA256SUMS or dir/SHA512SUMS for every
// regular file in dir, in `sha256sum -c` / `shasum -a 512 -c` format.
func WriteChecksumsAlg(dir, alg string) (string, map[string]string, error) {
	file, ok := ChecksumFiles[alg]
	if !ok {
		alg, file = "sha256", ChecksumsFile
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", nil, err
	}
	sums := map[string]string{}
	var names []string
	for _, e := range entries {
		n := e.Name()
		if !e.Type().IsRegular() || n == "SHA256SUMS" || n == "SHA512SUMS" || strings.HasPrefix(n, ".") || strings.HasSuffix(n, ".partial") {
			continue
		}
		s, err := HashFile(filepath.Join(dir, n), alg)
		if err != nil {
			return "", nil, err
		}
		sums[n] = s
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		fmt.Fprintf(&b, "%s  %s\n", sums[n], n)
	}
	p := filepath.Join(dir, file)
	return p, sums, os.WriteFile(p, []byte(b.String()), 0o644)
}

// CopyFile copies a regular file preserving its mode.
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	st, err := in.Stat()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".partial"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, st.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

// MoveFile renames, falling back to copy+remove across devices.
func MoveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	if err := CopyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

// CopyDir copies a directory tree preserving modes and symlinks.
func CopyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		case d.IsDir():
			return os.MkdirAll(target, info.Mode().Perm()|0o700)
		default:
			return CopyFile(path, target)
		}
	})
}
