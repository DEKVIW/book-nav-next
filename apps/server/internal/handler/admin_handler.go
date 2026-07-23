package handler

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/booknav/book-nav/apps/server/internal/domain"
	"github.com/booknav/book-nav/apps/server/internal/middleware"
	"github.com/booknav/book-nav/apps/server/internal/pkg/response"
	"github.com/booknav/book-nav/apps/server/internal/service"
	"github.com/go-chi/chi/v5"
)

type AdminHandler struct {
	admin      *service.AdminService
	websites   *service.WebsiteService
	categories *service.CategoryService
	settings   *service.SettingsService
	jobs       *service.JobService
}

func NewAdminHandler(
	admin *service.AdminService,
	websites *service.WebsiteService,
	categories *service.CategoryService,
	settings *service.SettingsService,
	jobs *service.JobService,
) *AdminHandler {
	return &AdminHandler{admin: admin, websites: websites, categories: categories, settings: settings, jobs: jobs}
}

func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	data, err := h.admin.Stats(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, data)
}

func (h *AdminHandler) ListWebsites(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	var catID *int64
	if v := r.URL.Query().Get("category_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			catID = &n
		}
	}
	items, total, err := h.websites.ListAdmin(r.Context(), page, pageSize, catID, r.URL.Query().Get("q"))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, map[string]any{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *AdminHandler) CreateWebsite(w http.ResponseWriter, r *http.Request) {
	var in service.WebsiteInput
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	site, err := h.websites.Create(r.Context(), middleware.UserFrom(r.Context()), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, site)
}

func (h *AdminHandler) UpdateWebsite(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in service.WebsiteInput
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	site, err := h.websites.Update(r.Context(), middleware.UserFrom(r.Context()), id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, site)
}

func (h *AdminHandler) DeleteWebsite(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.websites.Delete(r.Context(), middleware.UserFrom(r.Context()), id); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "deleted")
}

func (h *AdminHandler) BatchDeleteWebsites(w http.ResponseWriter, r *http.Request) {
	var body struct {
		IDs []int64 `json:"ids"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	n, err := h.websites.BatchDelete(r.Context(), middleware.UserFrom(r.Context()), body.IDs)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, map[string]any{"deleted": n})
}

func (h *AdminHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	list, err := h.categories.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, list)
}

func (h *AdminHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	var in service.CategoryInput
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	c, err := h.categories.Create(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, c)
}

func (h *AdminHandler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var in service.CategoryInput
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	c, err := h.categories.Update(r.Context(), id, in)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, c)
}

func (h *AdminHandler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.categories.Delete(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "deleted")
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.ListUsers(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	// hide hashes
	out := make([]map[string]any, 0, len(list))
	for i := range list {
		out = append(out, list[i].Public())
	}
	response.OK(w, out)
}

func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Role        string `json:"role"`
		NewPassword string `json:"new_password"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	u, err := h.admin.UpdateUser(r.Context(), middleware.UserFrom(r.Context()), id, domain.Role(body.Role), body.NewPassword)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, u.Public())
}

func (h *AdminHandler) ListInvites(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.ListInvites(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, list)
}

func (h *AdminHandler) CreateInvites(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Count int `json:"count"`
	}
	_ = decodeJSON(r, &body)
	list, err := h.admin.GenerateInvites(r.Context(), middleware.UserFrom(r.Context()), body.Count)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, list)
}

func (h *AdminHandler) DeleteInvite(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.admin.DeleteInvite(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "deleted")
}

func (h *AdminHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	data, err := h.settings.GetNamespaceForAdmin(r.Context(), ns)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, data)
}

func (h *AdminHandler) PutSettings(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	if err := h.settings.SetNamespace(r.Context(), ns, body); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "saved")
}

func (h *AdminHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	page := queryInt(r, "page", 1)
	pageSize := queryInt(r, "page_size", 20)
	var uid *int64
	if v := r.URL.Query().Get("user_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			uid = &n
		}
	}
	items, total, err := h.admin.ListLogs(r.Context(), page, pageSize, uid)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, map[string]any{"items": items, "total": total, "page": page, "page_size": pageSize})
}

func (h *AdminHandler) Export(w http.ResponseWriter, r *http.Request) {
	data, err := h.admin.ExportNative(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, data)
}

func (h *AdminHandler) Import(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	stats, err := h.admin.ImportNative(r.Context(), body)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, stats)
}

// ImportLegacyDB3 imports old Flask export placed under data dir.
// body: { "filename": "booknav_export_xxx.db3", "mode": "merge"|"replace" }
func (h *AdminHandler) ImportLegacyDB3(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Filename string `json:"filename"`
		Mode     string `json:"mode"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	path, err := h.admin.ResolveImportPath(h.admin.DataDir(), body.Filename)
	if err != nil {
		writeErr(w, err)
		return
	}
	stats, err := h.admin.ImportLegacyDB3(r.Context(), path, body.Mode, middleware.UserFrom(r.Context()))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, stats, "导入完成")
}

func (h *AdminHandler) CreateBackup(w http.ResponseWriter, r *http.Request) {
	name, err := h.admin.BackupLocal(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, map[string]any{"name": name})
}

func (h *AdminHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	list, err := h.admin.ListBackups()
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, list)
}

func (h *AdminHandler) DownloadBackup(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	path := h.admin.BackupPath(name)
	if _, err := os.Stat(path); err != nil {
		response.NotFound(w, "backup not found")
		return
	}
	w.Header().Set("Content-Disposition", "attachment; filename="+name)
	http.ServeFile(w, r, path)
}

func (h *AdminHandler) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.admin.DeleteBackup(name); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "deleted")
}

func (h *AdminHandler) RestoreBackup(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.admin.RestoreBackup(name); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "已回滚，建议重启服务")
}

// —— WebDAV cloud backup ——

func (h *AdminHandler) ListWebDAV(w http.ResponseWriter, r *http.Request) {
	list, err := h.settings.ListWebDAV(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, list)
}

func (h *AdminHandler) SaveWebDAV(w http.ResponseWriter, r *http.Request) {
	var in service.WebDAVConfig
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	list, err := h.settings.SaveWebDAV(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, list, "已保存")
}

func (h *AdminHandler) DeleteWebDAV(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	list, err := h.settings.DeleteWebDAV(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, list, "已删除")
}

func (h *AdminHandler) TestWebDAV(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	data, err := h.settings.TestWebDAV(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, data)
}

func (h *AdminHandler) UploadBackupWebDAV(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		Filename string `json:"filename"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.Filename) == "" {
		response.BadRequest(w, "filename required")
		return
	}
	local := h.admin.BackupPath(body.Filename)
	if err := h.settings.UploadBackupToWebDAV(r.Context(), id, local, body.Filename); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "已上传到云端")
}

func (h *AdminHandler) ListRemoteWebDAV(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	list, err := h.settings.ListRemoteWebDAV(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, list)
}

func (h *AdminHandler) ClearWebsites(w http.ResponseWriter, r *http.Request) {
	if err := h.admin.ClearWebsites(r.Context()); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "cleared")
}

func (h *AdminHandler) StartDeadlink(w http.ResponseWriter, r *http.Request) {
	j, err := h.jobs.StartDeadlinkCheck(r.Context(), middleware.UserFrom(r.Context()))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, j)
}

func (h *AdminHandler) StartIconJob(w http.ResponseWriter, r *http.Request) {
	j, err := h.jobs.StartIconFetch(r.Context(), middleware.UserFrom(r.Context()))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, j)
}

func (h *AdminHandler) StartVectorJob(w http.ResponseWriter, r *http.Request) {
	j, err := h.jobs.StartVectorIndex(r.Context(), middleware.UserFrom(r.Context()))
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, j)
}

func (h *AdminHandler) TestVector(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	_ = decodeJSON(r, &body)
	if body == nil {
		body = map[string]any{}
	}
	data, err := h.jobs.TestVectorConfig(r.Context(), body)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, data)
}

// —— AI management (multi-provider + tasks) ——

func (h *AdminHandler) GetAIState(w http.ResponseWriter, r *http.Request) {
	state, err := h.settings.GetAIState(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, state)
}

func (h *AdminHandler) SaveAIProvider(w http.ResponseWriter, r *http.Request) {
	var in service.AIProvider
	if err := decodeJSON(r, &in); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	state, err := h.settings.SaveAIProvider(r.Context(), in)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, state, "提供方已保存")
}

func (h *AdminHandler) DeleteAIProvider(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	state, err := h.settings.DeleteAIProvider(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, state, "提供方已删除")
}

func (h *AdminHandler) DetectAIProvider(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	state, err := h.settings.DetectProviderModels(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, state, "模型检测完成")
}

func (h *AdminHandler) DetectAllAIProviders(w http.ResponseWriter, r *http.Request) {
	state, okNames, errs, err := h.settings.DetectAllProviders(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	msg := "未执行任何检测"
	if len(okNames) > 0 {
		msg = "已检测 " + strconv.Itoa(len(okNames)) + " 个提供方"
	}
	if len(errs) > 0 {
		msg += "，失败 " + strconv.Itoa(len(errs)) + " 个"
	}
	response.OK(w, map[string]any{
		"state":    state,
		"detected": okNames,
		"errors":   errs,
		"message":  msg,
		"ok":       len(okNames) > 0 && len(errs) == 0,
	})
}

func (h *AdminHandler) TestAIProvider(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var body struct {
		ModelName string `json:"model_name"`
	}
	_ = decodeJSON(r, &body)
	data, err := h.settings.TestProvider(r.Context(), id, body.ModelName)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, data)
}

func (h *AdminHandler) SaveAITaskBindings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TaskBindings map[string]service.AITaskBinding `json:"task_bindings"`
	}
	if err := decodeJSON(r, &body); err != nil {
		response.BadRequest(w, "invalid json")
		return
	}
	state, err := h.settings.SaveTaskBindings(r.Context(), body.TaskBindings)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, state, "任务模型已保存")
}

func (h *AdminHandler) TestAITasks(w http.ResponseWriter, r *http.Request) {
	// optional draft bindings
	var body struct {
		TaskBindings map[string]service.AITaskBinding `json:"task_bindings"`
	}
	_ = decodeJSON(r, &body)
	if body.TaskBindings != nil {
		if _, err := h.settings.SaveTaskBindings(r.Context(), body.TaskBindings); err != nil {
			writeErr(w, err)
			return
		}
	}
	state, err := h.settings.TestAllTasks(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	success := 0
	for _, t := range state.TaskTests {
		if t.Status == "success" {
			success++
		}
	}
	response.OK(w, map[string]any{
		"state":   state,
		"message": "已完成 4 项测试，成功 " + strconv.Itoa(success) + " 项",
		"ok":      success == 4,
	})
}

func (h *AdminHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	j, err := h.jobs.Get(r.Context(), id)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, j)
}

func (h *AdminHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	list, err := h.jobs.List(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, list)
}

func (h *AdminHandler) DeleteJob(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.jobs.Delete(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "已删除")
}

func (h *AdminHandler) ClearJobs(w http.ResponseWriter, r *http.Request) {
	n, err := h.jobs.ClearFinished(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, map[string]any{"deleted": n})
}

func (h *AdminHandler) DeleteLog(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.admin.DeleteLog(r.Context(), id); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "已删除")
}

func (h *AdminHandler) ClearLogs(w http.ResponseWriter, r *http.Request) {
	if err := h.admin.ClearLogs(r.Context()); err != nil {
		writeErr(w, err)
		return
	}
	response.OKMessage(w, nil, "已清空")
}

func (h *AdminHandler) ListDeadlinks(w http.ResponseWriter, r *http.Request) {
	batch := r.URL.Query().Get("batch_id")
	invalidOnly := r.URL.Query().Get("invalid_only") == "1"
	list, err := h.jobs.ListDeadlinks(r.Context(), batch, invalidOnly)
	if err != nil {
		writeErr(w, err)
		return
	}
	response.OK(w, list)
}

