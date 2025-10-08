/*
 * File: 07-multipart-file-upload.go
 * Date: 2025/9/9 23:44
 * Author : 你是否看到一只鸡
 * Description: God help those who stand in our way!
 */

package simple_gin

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

/*
	文件上传
		浏览器/客户端用 Content-Type: multipart/form-data; boundary=... 发送二进制文件 + 表单字段
		服务器解析 multipart，把每个 part（字段/文件）暴露为 *multipart.FileHeader
	类型与安全校验原则
		来自客户端的 Content-Type 与扩展名可伪造
		正确做法：读取文件头 magic bytes → http.DetectContentType() → 自己做白名单匹配
*/

const (
	// MaxMultipartMemory 内存缓冲上限, 小于1MB的上传文件会直接放到内存里,超过2MB的部分会被自动存到系统临时目录（Linux 下通常 /tmp），而不是一下子全塞内存
	MaxMultipartMemory = 1 << 20 // 1MB
	// MaxRequestBytes 整次请求（包含文件+表单字段）的硬性上限，超过就直接拒绝
	MaxRequestBytes = 2 << 20     // 5MB
	MaxFileSize     = 1 << 20     // 单个文件的最大允许大小。超过就报错
	MaxFileCount    = 2           // 一次请求允许上传的文件数量上限
	UploadDir       = "./uploads" // 当前程序运行目录下的 uploads 文件夹
	// 左移二进制位 1 << 10 = 1024  1 << 20 = 1048576（1MB）
)

// 允许的 MIME
var allowedMIMEs = map[string]struct{}{
	"image/jpeg":      {},
	"image/png":       {},
	"image/gif":       {},
	"image/webp":      {},
	"application/pdf": {},
}

// 受控扩展名（与 MIME 双重白名单）
var allowedExt = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".gif": {}, ".webp": {}, ".pdf": {},
}

type UploadMeta struct {
	Biz     string `form:"biz" binding:"omitempty,max=32"`
	OwnerID string `form:"owner_id" binding:"omitempty,max=64"`
}

func uploadSingle(ctx *gin.Context) {
	var meta UploadMeta
	if err := ctx.ShouldBind(&meta); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid form fields", "detail": err.Error()})
		return
	}
	// 取文件（字段名 file）
	fh, err := ctx.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "file not provided"})
		return
	}
	info, err := processOneFile(ctx, fh)
	if err != nil {
		status := http.StatusBadRequest
		if isRequestTooLarge(err) {
			status = http.StatusRequestEntityTooLarge
		}
		ctx.JSON(status, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"biz":   meta.Biz,
		"owner": meta.OwnerID,
		"file":  info,
	})
}

func uploadMultiple(ctx *gin.Context) {
	var meta UploadMeta
	if err := ctx.ShouldBind(&meta); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid form fields", "detail": err.Error()})
		return
	}
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart form", "detail": err.Error()})
		return
	}
	files := form.File["files"]
	if len(files) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "files not provided"})
		return
	}
	if len(files) > MaxFileCount {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("too many files (max %d)", MaxFileCount)})
		return
	}
	result := make([]FileInfo, 0, len(files))
	for _, fh := range files {
		info, err := processOneFile(ctx, fh)
		if err != nil {
			status := http.StatusBadRequest
			if isRequestTooLarge(err) {
				status = http.StatusRequestEntityTooLarge
			}
			ctx.JSON(status, gin.H{"error": err.Error()})
			return
		}
		result = append(result, info)
	}
	ctx.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"biz":   meta.Biz,
		"owner": meta.OwnerID,
		"files": result,
	})
}

func isRequestTooLarge(err error) bool {
	if err == nil {
		return false
	}
	var mbe *http.MaxBytesError
	if errors.As(err, &mbe) {
		return true
	}
	if errors.Is(err, multipart.ErrMessageTooLarge) {
		return true
	}
	return false
}

// FileInfo 核心：逐文件校验与安全保存
type FileInfo struct {
	OrigName string `json:"orig_name"`
	Size     int64  `json:"size"`
	MIME     string `json:"mime"`
	SavedAs  string `json:"saved_as"`
	URL      string `json:"url"`
}

func processOneFile(ctx *gin.Context, fh *multipart.FileHeader) (FileInfo, error) {
	// 单文件大小上限（Header 中 size 不一定可靠，但可先做一次快筛）
	if fh.Size > MaxFileSize {
		return FileInfo{}, fmt.Errorf("file too large: %s exceeds %d bytes", fh.Filename, MaxFileSize)
	}
	// 打开流（需要手动关闭）
	file, err := fh.Open()
	if err != nil {
		return FileInfo{}, fmt.Errorf("open file failed: %w", err)
	}
	defer func(file multipart.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)
	// 读取前 512 字节做 MIME 嗅探（不会改变 read pointer 对 Gin 的 SaveUploadedFile 没影响；如果后面自己写盘，要把已读部分写回）
	// http.DetectContentType 的规定 该函数最多检查输入的前 512 字节来判断内容类型；给它更长也只看前 512
	// head := make([]byte, 512)
	// n, _ := io.ReadFull(file, head) // 如果小于 512 也没关系
	// mime := http.DetectContentType(head[:n])
	mime, _, n, _ := sniffMIME(file)
	if _, ok := allowedMIMEs[mime]; !ok {
		return FileInfo{}, fmt.Errorf("mime not allowed: %s detected as %s", fh.Filename, mime)
	}
	// 扩展名白名单（可选，双重保险） .webp
	ext := strings.ToLower(filepath.Ext(fh.Filename))
	if ext == "" {
		// 为无扩展名文件根据 MIME自行映射：如 image/jpeg -> .jpg
		ext = suggestExt(mime)
	}
	if _, ok := allowedExt[ext]; !ok {
		return FileInfo{}, fmt.Errorf("extension not allowed: %s", ext)
	}
	// 再做一次严格的实际大小检查（避免 Header 欺骗）。使用 io.LimitedReader
	if err := fSizeWithinLimit(file, int64(MaxFileSize), n); err != nil {
		return FileInfo{}, err
	}
	// 生成安全文件名与目标路径
	safeName := randomName() + ext
	dst := filepath.Join(UploadDir, filepath.Base(safeName))
	if err := saveByReopen(fh, dst); err != nil {
		return FileInfo{}, err
	}
	// 用于 ctx.SaveUploadedFile 简化落盘
	// if err = ctx.SaveUploadedFile(fh, dst); err != nil {
	//	return FileInfo{}, fmt.Errorf("save failed: %w", err)
	// }
	return FileInfo{
		OrigName: fh.Filename,
		Size:     fh.Size, // 注意：这是 header 提供的可能与实际略有偏差（已用 fSizeWithinLimit 做实测）
		MIME:     mime,
		SavedAs:  safeName,
		URL:      "/static/" + safeName, // 实际可用对象存储 URL
	}, nil
}

func sniffMIME(src io.Reader) (mime string, head []byte, n int, br *bufio.Reader) {
	// 更稳的嗅探写法
	// http.DetectContentType 的规定 该函数最多检查输入的前 512 字节来判断内容类型；给它更长也只看前 512
	br = bufio.NewReader(src)
	// Peek 最多返回 512（或更少，如果流一开始就没那么多）
	peek, _ := br.Peek(512)
	n = len(peek)
	mime = http.DetectContentType(peek[:n]) // text/plain; charset=utf-8
	// 为了兼容想要把头部单独拿走，再复制一份
	head = make([]byte, n)
	copy(head, peek[:n])
	return
}

func saveByReopen(fh *multipart.FileHeader, dst string) error {
	// 直接用 Gin 提供的方法：内部会处理从临时文件/内存到目标文件的复制
	return saveFileStream(fh, dst)
}

// saveFileStream 在没有 gin.Context 的情况下安全落盘：
// 1) 打开 multipart 文件；2) 写入同目录临时文件；3) Sync；4) Rename 原子替换。
// 可选：传入已嗅探的 head 数据以避免重复读取（见注释）。
func saveFileStream(fh *multipart.FileHeader, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("mkdir %s failed: %w", filepath.Dir(dst), err)
	}

	src, err := fh.Open()
	if err != nil {
		return fmt.Errorf("open upload failed: %w", err)
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {
			return
		}
	}(src)

	// 在目标目录创建临时文件，确保 rename 原子
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".upload-*")
	if err != nil {
		return fmt.Errorf("create temp failed: %w", err)
	}
	defer func() {
		err := tmp.Close()
		if err != nil {
			return
		}
		_ = os.Remove(tmp.Name()) // 最终会被 rename 覆盖；失败时清理
	}()

	// 流式复制剩余数据（或全部数据）
	if _, err := io.Copy(tmp, src); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	// 刷盘，降低断电风险
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("fsync failed: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp failed: %w", err)
	}

	// 原子替换到最终路径
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return fmt.Errorf("rename failed: %w", err)
	}

	// 设定权限（可选，umask 会影响最终权限）
	if err := os.Chmod(dst, 0o644); err != nil {
		return fmt.Errorf("chmod failed: %w", err)
	}
	return nil
}

func randomName() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	ts := time.Now().UTC().Format("20060102T150405Z")
	return ts + "_" + hex.EncodeToString(b)
}

func fSizeWithinLimit(f multipart.File, limit int64, alreadyRead int) error {
	// 将已读取的 head 长度计入；余下可读字节 <= limit - alreadyRead(512)
	remain := limit - int64(alreadyRead)
	if remain < 0 {
		return fmt.Errorf("file too large (head)")
	}
	lr := io.LimitedReader{R: f, N: remain + 1} // +1 用于检测是否超限
	written, _ := io.Copy(io.Discard, &lr)
	if written > remain {
		return fmt.Errorf("file too large")
	}
	return nil
}

func suggestExt(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "application/pdf":
		return ".pdf"
	default:
		return ""
	}
}

func RegisterUploadRoutes(r gin.IRouter) {
	api := r.Group("/api/v1/upload")
	{
		api.POST("/single", uploadSingle)
		api.POST("/multiple", uploadMultiple)
	}
}
