package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/pkg/apperr"
	"github.com/booknav/book-nav/apps/server/internal/pkg/password"
	"github.com/booknav/book-nav/apps/server/internal/repository"
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

func (s *AdminService) ExportNative(ctx context.Context) (map[string]any, error) {
	users, _ := s.users.List(ctx)
	cats, _ := s.categories.ListAll(ctx)
	sites, _ := s.websites.ListAll(ctx)
	// strip password hashes from export? keep for restore
	return map[string]any{
		"format":     "booknav-native",
		"version":    1,
		"exported_at": time.Now().UTC().Format(time.RFC3339),
		"users":      users,
		"categories": cats,
		"websites":   sites,
	}, nil
}

func (s *AdminService) ImportNative(ctx context.Context, payload map[string]any) (map[string]int, error) {
	// simplified: import websites + categories only if empty-ish
	rawCats, _ := json.Marshal(payload["categories"])
	rawSites, _ := json.Marshal(payload["websites"])
	var cats []domain.Category
	var sites []domain.Website
	_ = json.Unmarshal(rawCats, &cats)
	_ = json.Unmarshal(rawSites, &sites)

	// map oldID -> newID for categories
	idMap := map[int64]int64{}
	createdCats := 0
	for _, c := range cats {
		oldID := c.ID
		c.ID = 0
		// remap parent later
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
	// second pass parents
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

	// need a user for created_by
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

func (s *AdminService) BackupLocal(ctx context.Context) (string, error) {
	dir := filepath.Join(s.dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("booknav-%s.db", time.Now().UTC().Format("20060102-150405"))
	dst := filepath.Join(dir, name)
	in, err := os.ReadFile(s.dbPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(dst, in, 0o644); err != nil {
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
		item := map[string]any{"name": e.Name()}
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

// RestoreBackup replaces live DB with a local backup snapshot.
// Caller should restart process after restore for best consistency; we copy file in place.
func (s *AdminService) RestoreBackup(name string) error {
	src := s.BackupPath(name)
	if _, err := os.Stat(src); err != nil {
		return apperr.New(apperr.NotFound, "备份不存在")
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	// write to temp then rename
	tmp := s.dbPath + ".restore-tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	// best-effort replace (Windows may lock; still write)
	_ = os.Remove(s.dbPath)
	if err := os.Rename(tmp, s.dbPath); err != nil {
		// fallback copy
		if err2 := os.WriteFile(s.dbPath, data, 0o644); err2 != nil {
			return err2
		}
		_ = os.Remove(tmp)
	}
	return nil
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
