package service

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/pkg/password"
	"github.com/booknav/book-nav/apps/server/internal/repository"
	_ "modernc.org/sqlite"
)

type AdminService struct {
	users      *repository.UserRepo
	invites    *repository.InvitationRepo
	websites   *repository.WebsiteRepo
	categories *repository.CategoryRepo
	oplog      *repository.OpLogRepo
	jobs       *repository.JobRepo
	settings   *SettingsService
	dataDir    string
	dbPath     string
}

func NewAdminService(
	users *repository.UserRepo,
	invites *repository.InvitationRepo,
	websites *repository.WebsiteRepo,
	categories *repository.CategoryRepo,
	oplog *repository.OpLogRepo,
	jobs *repository.JobRepo,
	settings *SettingsService,
	dataDir, dbPath string,
) *AdminService {
	return &AdminService{
		users: users, invites: invites, websites: websites, categories: categories,
		oplog: oplog, jobs: jobs, settings: settings, dataDir: dataDir, dbPath: dbPath,
	}
}

func (s *AdminService) DataDir() string { return s.dataDir }

func (s *AdminService) Stats(ctx context.Context) (map[string]any, error) {
	users, _ := s.users.Count(ctx)
	sites, _ := s.websites.Count(ctx)
	cats, err := s.categories.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	running, _ := s.jobs.CountRunning(ctx)
	return map[string]any{
		"users":        users,
		"websites":     sites,
		"categories":   len(cats),
		"jobs_running": running,
	}, nil
}

func (s *AdminService) ListUsers(ctx context.Context) ([]domain.User, error) {
	return s.users.List(ctx)
}

func (s *AdminService) UpdateUser(ctx context.Context, actor *domain.User, id int64, role domain.Role, newPassword string) (*domain.User, error) {
	if actor == nil || !actor.Role.IsSuperAdmin() {
		return nil, apperr.New(apperr.Forbidden, "需要超级管理员")
	}
	u, err := s.users.GetByID(ctx, id)
	if err != nil || u == nil {
		return nil, apperr.New(apperr.NotFound, "用户不存在")
	}
	if role.Valid() {
		if u.ID == actor.ID && role != domain.RoleSuperAdmin {
			return nil, apperr.New(apperr.Validation, "不能取消自己的超管身份")
		}
		u.Role = role
	}
	if newPassword != "" {
		hash, err := password.Hash(newPassword)
		if err != nil {
			return nil, err
		}
		u.PasswordHash = hash
	}
	if err := s.users.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *AdminService) GenerateInvites(ctx context.Context, actor *domain.User, count int) ([]domain.InvitationCode, error) {
	if actor == nil || !actor.Role.IsAdmin() {
		return nil, apperr.New(apperr.Forbidden, "需要管理员权限")
	}
	if count < 1 {
		count = 1
	}
	if count > 50 {
		count = 50
	}
	var out []domain.InvitationCode
	uid := actor.ID
	for i := 0; i < count; i++ {
		code, err := randomInviteCode(8)
		if err != nil {
			return nil, err
		}
		inv, err := s.invites.Create(ctx, code, &uid)
		if err != nil {
			return nil, err
		}
		out = append(out, *inv)
	}
	return out, nil
}

func (s *AdminService) ListInvites(ctx context.Context) ([]domain.InvitationCode, error) {
	return s.invites.List(ctx)
}

func (s *AdminService) DeleteInvite(ctx context.Context, id int64) error {
	return s.invites.Delete(ctx, id)
}

func (s *AdminService) DeleteLog(ctx context.Context, id int64) error {
	return s.oplog.Delete(ctx, id)
}

func (s *AdminService) ClearLogs(ctx context.Context) error {
	// clear all by deleting in batches via empty user filter is not available; use raw through repo helper
	return s.oplog.ClearAll(ctx)
}

func (s *AdminService) ListLogs(ctx context.Context, page, pageSize int, userID *int64) ([]domain.OperationLog, int, error) {
	return s.oplog.List(ctx, page, pageSize, userID)
}

// configNamespaces are settings packs (not navigation content).
var configNamespaces = []string{
	"site", "transition", "announcement", "ai", "vector", "icon", "webdav",
}

// SnapshotDatabase writes a consistent SQLite copy via VACUUM INTO.
func (s *AdminService) SnapshotDatabase(dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	_ = os.Remove(dst)
	db, err := sql.Open("sqlite", s.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()
	_, _ = db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	if _, err := db.Exec(`VACUUM INTO ?`, dst); err != nil {
		// fallback: raw copy (may be incomplete under heavy write)
		in, rerr := os.ReadFile(s.dbPath)
		if rerr != nil {
			return err
		}
		return os.WriteFile(dst, in, 0o644)
	}
	return nil
}

// ExportSQLiteTemp creates a downloadable .db3 snapshot; caller should remove path after send.
func (s *AdminService) ExportSQLiteTemp() (path string, err error) {
	path = filepath.Join(os.TempDir(), fmt.Sprintf("booknav-export-%d.db3", time.Now().UnixNano()))
	if err := s.SnapshotDatabase(path); err != nil {
		return "", err
	}
	return path, nil
}

// ExportNative JSON export of categories/websites/users (legacy helper; prefer SQLite export).
func (s *AdminService) ExportNative(ctx context.Context) (map[string]any, error) {
	users, _ := s.users.List(ctx)
	cats, _ := s.categories.ListAll(ctx)
	sites, _ := s.websites.ListAll(ctx)
	return map[string]any{
		"format":      "booknav-native",
		"version":     1,
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"users":       users,
		"categories":  cats,
		"websites":    sites,
	}, nil
}

func (s *AdminService) ImportNative(ctx context.Context, payload map[string]any) (map[string]int, error) {
	rawCats, _ := json.Marshal(payload["categories"])
	rawSites, _ := json.Marshal(payload["websites"])
	var cats []domain.Category
	var sites []domain.Website
	_ = json.Unmarshal(rawCats, &cats)
	_ = json.Unmarshal(rawSites, &sites)

	idMap := map[int64]int64{}
	createdCats := 0
	for _, c := range cats {
		oldID := c.ID
		in := CategoryInput{
			Name: c.Name, Description: c.Description, Icon: c.Icon, Color: c.Color,
			SortOrder: &c.SortOrder, DisplayLimit: &c.DisplayLimit,
		}
		created, err := NewCategoryService(s.categories, s.websites).Create(ctx, in)
		if err != nil {
			continue
		}
		idMap[oldID] = created.ID
		createdCats++
	}
	for _, c := range cats {
		if c.ParentID != nil {
			if newID, ok := idMap[c.ID]; ok {
				if newParent, ok2 := idMap[*c.ParentID]; ok2 {
					_ = s.categories.Update(ctx, &domain.Category{
						ID: newID, Name: c.Name, Description: c.Description, Icon: c.Icon, Color: c.Color,
						SortOrder: c.SortOrder, DisplayLimit: c.DisplayLimit, ParentID: &newParent,
					})
				}
			}
		}
	}

	users, _ := s.users.List(ctx)
	var owner int64 = 1
	if len(users) > 0 {
		owner = users[0].ID
	}
	createdSites := 0
	for _, w := range sites {
		var newCat *int64
		if w.CategoryID != nil {
			if nid, ok := idMap[*w.CategoryID]; ok {
				newCat = &nid
			}
		}
		w.ID = 0
		w.CategoryID = newCat
		w.CreatedBy = owner
		if err := s.websites.Create(ctx, &w); err == nil {
			createdSites++
		}
	}
	return map[string]int{"categories": createdCats, "websites": createdSites}, nil
}

// ExportConfigPack exports site / AI / icon / webdav settings as JSON (no link rows).
func (s *AdminService) ExportConfigPack(ctx context.Context) (map[string]any, error) {
	namespaces := map[string]any{}
	for _, ns := range configNamespaces {
		raw, err := s.settings.GetNamespace(ctx, ns)
		if err != nil {
			return nil, err
		}
		decoded := map[string]any{}
		for k, v := range raw {
			var val any
			if err := json.Unmarshal(v, &val); err != nil {
				decoded[k] = json.RawMessage(v)
			} else {
				decoded[k] = val
			}
		}
		namespaces[ns] = decoded
	}
	return map[string]any{
		"format":      "booknav-config",
		"version":     1,
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"namespaces":  namespaces,
	}, nil
}

// ImportConfigPack merges config namespaces from a booknav-config JSON.
func (s *AdminService) ImportConfigPack(ctx context.Context, payload map[string]any) (map[string]int, error) {
	nsRaw, ok := payload["namespaces"].(map[string]any)
	if !ok {
		// allow flat namespace maps at top level
		nsRaw = map[string]any{}
		for _, ns := range configNamespaces {
			if v, ok := payload[ns]; ok {
				nsRaw[ns] = v
			}
		}
	}
	if len(nsRaw) == 0 {
		return nil, apperr.New(apperr.Validation, "无效的配置包")
	}
	applied := 0
	for _, ns := range configNamespaces {
		v, ok := nsRaw[ns]
		if !ok {
			continue
		}
		m, ok := v.(map[string]any)
		if !ok {
			// re-marshal through json
			b, _ := json.Marshal(v)
			_ = json.Unmarshal(b, &m)
		}
		if len(m) == 0 {
			continue
		}
		if err := s.settings.SetNamespace(ctx, ns, m); err != nil {
			return nil, err
		}
		applied++
	}
	return map[string]int{"namespaces": applied}, nil
}

// BackupLocal creates a full data zip: database + uploads (icons, media).
func (s *AdminService) BackupLocal(ctx context.Context) (string, error) {
	_ = ctx
	dir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("booknav-data-%s.zip", time.Now().UTC().Format("20060102-150405"))
	dst := filepath.Join(dir, name)

	tmpDB := filepath.Join(os.TempDir(), fmt.Sprintf("booknav-bak-%d.db", time.Now().UnixNano()))
	if err := s.SnapshotDatabase(tmpDB); err != nil {
		return "", err
	}
	defer os.Remove(tmpDB)

	f, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer f.Close()
	zw := zip.NewWriter(f)

	meta := map[string]any{
		"format":      "booknav-data-backup",
		"version":     1,
		"exported_at": time.Now().UTC().Format(time.RFC3339),
	}
	mb, _ := json.MarshalIndent(meta, "", "  ")
	if w, err := zw.Create("manifest.json"); err == nil {
		_, _ = w.Write(mb)
	}
	if err := zipWriteFile(zw, "booknav.db", tmpDB); err != nil {
		_ = zw.Close()
		_ = os.Remove(dst)
		return "", err
	}
	uploads := filepath.Join(s.dataDir, "uploads")
	if err := zipAddDir(zw, uploads, "uploads"); err != nil {
		_ = zw.Close()
		_ = os.Remove(dst)
		return "", err
	}
	if err := zw.Close(); err != nil {
		_ = os.Remove(dst)
		return "", err
	}
	return name, nil
}

// BackupConfigLocal writes a config-only JSON file under backups/.
// Filename ends with .config.json so it is obvious this is settings, not data.
// Contents (namespaces): site, transition, announcement, ai, vector, icon, webdav
// — includes AI providers/keys, vector/Qdrant, icon sources, WebDAV, etc.
// Restore via ImportConfigPack merges into settings_kv (secrets: empty/mask keeps previous).
func (s *AdminService) BackupConfigLocal(ctx context.Context) (string, error) {
	dir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	pack, err := s.ExportConfigPack(ctx)
	if err != nil {
		return "", err
	}
	name := fmt.Sprintf("booknav-%s.config.json", time.Now().UTC().Format("20060102-150405"))
	dst := filepath.Join(dir, name)
	b, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dst, b, 0o644); err != nil {
		return "", err
	}
	return name, nil
}

func (s *AdminService) ListBackups() ([]map[string]any, error) {
	dir := filepath.Join(s.dataDir, "backups")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []map[string]any{}, nil
		}
		return nil, err
	}
	var out []map[string]any
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, _ := e.Info()
		name := e.Name()
		kind := "other"
		lower := strings.ToLower(name)
		switch {
		case strings.HasSuffix(lower, ".config.json"),
			strings.HasPrefix(name, "booknav-config-") && strings.HasSuffix(lower, ".json"):
			kind = "config"
		case strings.HasPrefix(name, "booknav-data-") && strings.HasSuffix(lower, ".zip"):
			kind = "data"
		case strings.HasSuffix(lower, ".db"), strings.HasSuffix(lower, ".db3"):
			kind = "legacy-db"
		case strings.HasSuffix(lower, ".zip"):
			kind = "data"
		case strings.HasSuffix(lower, ".json") && strings.Contains(lower, "config"):
			kind = "config"
		}
		item := map[string]any{"name": name, "kind": kind}
		if info != nil {
			item["size"] = info.Size()
			item["mod_time"] = info.ModTime().UTC().Format(time.RFC3339)
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *AdminService) BackupPath(name string) string {
	return filepath.Join(s.dataDir, "backups", filepath.Base(name))
}

func (s *AdminService) DeleteBackup(name string) error {
	return os.Remove(s.BackupPath(name))
}

// RestoreBackup restores a data zip or legacy .db/.db3 snapshot.
func (s *AdminService) RestoreBackup(name string) error {
	src := s.BackupPath(name)
	if _, err := os.Stat(src); err != nil {
		return apperr.New(apperr.NotFound, "备份不存在")
	}
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".json"):
		return apperr.New(apperr.Validation, "配置备份请使用「恢复配置」")
	case strings.HasSuffix(lower, ".zip"):
		return s.restoreDataZip(src)
	case strings.HasSuffix(lower, ".db"), strings.HasSuffix(lower, ".db3"):
		return s.replaceDatabaseFile(src)
	default:
		return apperr.New(apperr.Validation, "无法识别的备份格式")
	}
}

// RestoreConfigBackup applies a config JSON backup file from backups/.
func (s *AdminService) RestoreConfigBackup(ctx context.Context, name string) error {
	src := s.BackupPath(name)
	b, err := os.ReadFile(src)
	if err != nil {
		return apperr.New(apperr.NotFound, "备份不存在")
	}
	var payload map[string]any
	if err := json.Unmarshal(b, &payload); err != nil {
		return apperr.New(apperr.Validation, "无效的配置备份")
	}
	_, err = s.ImportConfigPack(ctx, payload)
	return err
}

func (s *AdminService) replaceDatabaseFile(src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	tmp := s.dbPath + ".restore-tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	// drop WAL sidecars so restored main file is authoritative
	_ = os.Remove(s.dbPath + "-wal")
	_ = os.Remove(s.dbPath + "-shm")
	_ = os.Remove(s.dbPath)
	if err := os.Rename(tmp, s.dbPath); err != nil {
		if err2 := os.WriteFile(s.dbPath, data, 0o644); err2 != nil {
			return err2
		}
		_ = os.Remove(tmp)
	}
	return nil
}

func (s *AdminService) restoreDataZip(zipPath string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	tmpDir, err := os.MkdirTemp("", "booknav-restore-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	var dbMember string
	for _, f := range r.File {
		name := filepath.ToSlash(f.Name)
		if strings.Contains(name, "..") {
			continue
		}
		target := filepath.Join(tmpDir, filepath.FromSlash(name))
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := extractZipFile(f, target); err != nil {
			return err
		}
		base := strings.ToLower(filepath.Base(name))
		if base == "booknav.db" || strings.HasSuffix(base, ".db3") || base == "booknav.db3" {
			dbMember = target
		}
	}
	if dbMember == "" {
		// search any .db
		_ = filepath.Walk(tmpDir, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if strings.HasSuffix(strings.ToLower(info.Name()), ".db") || strings.HasSuffix(strings.ToLower(info.Name()), ".db3") {
				dbMember = p
			}
			return nil
		})
	}
	if dbMember == "" {
		return apperr.New(apperr.Validation, "备份中未找到数据库文件")
	}
	if err := s.replaceDatabaseFile(dbMember); err != nil {
		return err
	}
	// restore uploads if present
	upSrc := filepath.Join(tmpDir, "uploads")
	if st, err := os.Stat(upSrc); err == nil && st.IsDir() {
		upDst := filepath.Join(s.dataDir, "uploads")
		_ = os.RemoveAll(upDst)
		if err := copyDir(upSrc, upDst); err != nil {
			return err
		}
	}
	return nil
}

func zipWriteFile(zw *zip.Writer, name, src string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, in)
	return err
}

func zipAddDir(zw *zip.Writer, dir, prefix string) error {
	st, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !st.IsDir() {
		return nil
	}
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		name := filepath.ToSlash(filepath.Join(prefix, rel))
		if info.IsDir() {
			_, err := zw.Create(name + "/")
			return err
		}
		return zipWriteFile(zw, name, path)
	})
}

func extractZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func (s *AdminService) ClearWebsites(ctx context.Context) error {
	return s.websites.ClearAll(ctx)
}

func randomInviteCode(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b)[:n], nil
}
