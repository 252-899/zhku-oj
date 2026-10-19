/*
@Author: omenkk7
@Date: 2026/03/17
@Description: 课程文件（分片上传/断点续传/下载）服务实现
*/

package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	courseFilesCollection = "course_files"
	defaultUploadDir      = "./data/uploads"
)

// 获取上传根目录（优先环境变量，否则用默认值）
func (s *CourseService) uploadBaseDir() string {
	if v := strings.TrimSpace(os.Getenv("UPLOAD_DIR")); v != "" {
		return v
	}
	return defaultUploadDir
}

// 拼接分片存储目录：./data/uploads/course/{课程ID}/{文件SHA256}/chunks
func (s *CourseService) courseFileChunksDir(courseID, fileSHA256 string) string {
	return filepath.Join(s.uploadBaseDir(), "course", courseID, fileSHA256, "chunks")
}

// 拼接最终文件存储目录：./data/uploads/course/{课程ID}/{文件SHA256}/final
func (s *CourseService) courseFileFinalDir(courseID, fileSHA256 string) string {
	return filepath.Join(s.uploadBaseDir(), "course", courseID, fileSHA256, "final")
}

// 安全处理文件名（防止路径穿越、非法字符）
func safeFileName(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "file"
	}
	return name
}

// CheckCourseFileUpload 上传前检查：秒传/断点续传

// isSubPath reports whether target stays within base after path cleaning.
func isSubPath(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// CheckCourseFileUpload validates instant upload / resume-upload state before chunks are sent.
func (s *CourseService) CheckCourseFileUpload(ctx context.Context, courseID, userID string, req *models.CourseFileUploadCheckRequest) (*models.CourseFileUploadCheckResponse, error) {
	courseObjID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		return nil, errors.New("无效的课程ID")
	}
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	// 课程成员可检查（方便学生端断点续传场景）
	if ok, err := courseMemberAuthorized(ctx, utils.GetCollection("courses"), courseObjID, userObjID); err != nil || !ok {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("权限不足")
	}

	// 秒传：已存在同 sha256 的课程文件记录
	var existing models.CourseFile
	err = utils.GetCollection(courseFilesCollection).FindOne(ctx, bson.M{
		"course_id": courseObjID,
		"sha256":    req.FileSHA256,
	}).Decode(&existing)
	if err == nil {
		return &models.CourseFileUploadCheckResponse{Instant: true, Uploaded: []int{}}, nil
	}
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return nil, err
	}

	//断点续传检查 → 扫描磁盘分片目录，返回已上传的分片索引
	chunksDir := s.courseFileChunksDir(courseID, req.FileSHA256)
	// 读取分片目录下的所有文件/文件夹
	entries, err := os.ReadDir(chunksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &models.CourseFileUploadCheckResponse{Instant: false, Uploaded: []int{}}, nil
		}
		return nil, err
	}

	// 遍历分片目录，提取已上传的分片索引
	uploaded := make([]int, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".part") { // 只处理分片文件（后缀是.part）
			continue
		}
		// 去掉.part后缀，得到分片索引（比如 0.part → 0）
		idxStr := strings.TrimSuffix(name, ".part")
		idx, err := strconv.Atoi(idxStr)
		if err != nil || idx < 0 {
			continue
		}
		uploaded = append(uploaded, idx)
	}
	sort.Ints(uploaded) // 排序（比如 [2,0,1] → [0,1,2]），方便前端处理

	return &models.CourseFileUploadCheckResponse{Instant: false, Uploaded: uploaded}, nil
}

// UploadCourseFileChunk 上传单个分片（幂等：已存在则直接成功）
func (s *CourseService) UploadCourseFileChunk(
	ctx context.Context,
	courseID string,
	userID string,
	fileSHA256 string,
	_ string, // fileName: 仅用于前端显示/调试，服务端以 sha256 为主
	chunkIndex int,
	totalChunks int,
	header *multipart.FileHeader,
	src multipart.File,
) error {
	courseObjID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		return errors.New("无效的课程ID")
	}
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("无效的用户ID")
	}

	// 默认：仅课程创建者/教师允许上传教学资源
	if ok, err := authorized(ctx, utils.GetCollection("courses"), courseObjID, userObjID); err != nil || !ok {
		if err != nil {
			return err
		}
		return errors.New("权限不足")
	}

	if strings.TrimSpace(fileSHA256) == "" {
		return errors.New("file_sha256 不能为空")
	}
	if totalChunks <= 0 || chunkIndex < 0 || chunkIndex >= totalChunks {
		return errors.New("分片参数不合法")
	}

	// 拼接分片存储目录：./data/uploads/course/{课程ID}/{文件SHA256}/chunks
	chunksDir := s.courseFileChunksDir(courseID, fileSHA256)
	// 递归创建目录（不存在则创建，已存在则无操作），权限0o755（所有者可读可写可执行，其他只读可执行）
	if err := os.MkdirAll(chunksDir, 0o755); err != nil {
		return err
	}

	// 最终分片文件路径：比如 chunks/0.part
	finalPath := filepath.Join(chunksDir, fmt.Sprintf("%d.part", chunkIndex))
	tmpPath := finalPath + ".tmp"

	// 幂等：已存在则直接成功
	if st, err := os.Stat(finalPath); err == nil && st.Size() > 0 {
		return nil
	}

	// 创建/打开临时文件：O_CREATE（不存在则创建）+ O_TRUNC（存在则清空）+ O_WRONLY（只写）
	dst, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	// 把前端上传的分片内容复制到临时文件//临时文件机制，保证分片传送的时候不会出现网络 0.part 一定是完整的。
	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	// Windows 下对已打开文件执行 Rename 会失败，必须先关闭句柄（Linux 同样兼容）
	if err := dst.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("关闭分片临时文件失败: %w", err)
	}

	//校验分片大小（关闭后再 Stat，两系统行为一致）
	if header != nil && header.Size > 0 {
		if st, err := os.Stat(tmpPath); err != nil || st.Size() != header.Size {
			_ = os.Remove(tmpPath)
			return errors.New("分片大小不一致")
		}
	}

	// 把临时文件重命名为正式分片文件（比如 0.part.tmp → 0.part）
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}

// MergeCourseFileChunks 合并分片：流式写入 + SHA256 校验 + 写入 Mongo 元数据
func (s *CourseService) MergeCourseFileChunks(ctx context.Context, courseID, userID string, req *models.CourseFileUploadMergeRequest) (string, error) {
	courseObjID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		return "", errors.New("无效的课程ID")
	}
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return "", errors.New("无效的用户ID")
	}

	// 默认：仅课程创建者/教师允许合并（上传最终确认）
	if ok, err := authorized(ctx, utils.GetCollection("courses"), courseObjID, userObjID); err != nil || !ok {
		if err != nil {
			return "", err
		}
		return "", errors.New("权限不足")
	}

	if req.TotalChunks <= 0 {
		return "", errors.New("total_chunks 不合法")
	}
	if strings.TrimSpace(req.FileSHA256) == "" {
		return "", errors.New("file_sha256 不能为空")
	}

	// 已存在则直接返回（避免重复合并）
	var existing models.CourseFile
	err = utils.GetCollection(courseFilesCollection).FindOne(ctx, bson.M{
		"course_id": courseObjID,
		"sha256":    req.FileSHA256,
	}).Decode(&existing)
	if err == nil {
		return existing.ID.Hex(), nil
	}
	if err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
		return "", err
	}

	// 校验每个分片存在
	chunksDir := s.courseFileChunksDir(courseID, req.FileSHA256)
	// 遍历所有分片索引，检查每个分片文件是否存在
	for i := 0; i < req.TotalChunks; i++ {
		p := filepath.Join(chunksDir, fmt.Sprintf("%d.part", i))
		if _, err := os.Stat(p); err != nil {
			return "", fmt.Errorf("缺少分片: %d", i)
		}
	}

	// 拼接最终文件存储目录：./data/uploads/course/{课程ID}/{SHA256}/final
	finalDir := s.courseFileFinalDir(courseID, req.FileSHA256)
	if err := os.MkdirAll(finalDir, 0o755); err != nil {
		return "", err
	}

	// 安全处理文件名（防止路径穿越、非法字符）
	fileName := safeFileName(req.FileName)
	// 最终文件路径：final/test.txt
	finalPath := filepath.Join(finalDir, fileName)
	// 临时文件路径：test.txt.tmp（先写临时文件，避免合并中断导致文件损坏）
	tmpFinal := finalPath + ".tmp"

	// 打开/创建临时文件（只写、清空已有内容）
	out, err := os.OpenFile(tmpFinal, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}

	hasher := sha256.New()
	// 多写入器：同时把内容写入临时文件 + SHA256 哈希器（一次读写，既存文件又算 SHA256）
	mw := io.MultiWriter(out, hasher)

	var written int64              // 记录合并后的总文件大小
	buf := make([]byte, 1024*1024) // 1MB buffer  // 1MB缓冲区（减少IO次数，提升合并效率）
	// 按顺序遍历所有分片，逐个写入临时文件
	for i := 0; i < req.TotalChunks; i++ {
		p := filepath.Join(chunksDir, fmt.Sprintf("%d.part", i))
		in, err := os.Open(p)
		if err != nil {
			_ = out.Close()
			_ = os.Remove(tmpFinal)
			return "", err
		}
		n, err := io.CopyBuffer(mw, in, buf)
		_ = in.Close()
		if err != nil {
			_ = out.Close()
			_ = os.Remove(tmpFinal)
			return "", err
		}
		written += n
	}

	// 校验1：合并后总大小是否和前端传的一致
	if req.FileSize > 0 && written != req.FileSize {
		_ = out.Close()
		_ = os.Remove(tmpFinal)
		return "", errors.New("合并后文件大小不一致")
	}

	// 校验2：合并后文件的 SHA256 是否和前端传的一致//自己根据内容重新计算一遍
	sum := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(sum, req.FileSHA256) {
		_ = out.Close()
		_ = os.Remove(tmpFinal)
		return "", errors.New("文件校验失败（SHA256不一致）")
	}

	if err := out.Close(); err != nil {
		_ = os.Remove(tmpFinal)
		return "", fmt.Errorf("关闭合并临时文件失败: %w", err)
	}

	// 把临时文件重命名为正式文件（test.txt.tmp → test.txt）
	if err := os.Rename(tmpFinal, finalPath); err != nil {
		_ = os.Remove(tmpFinal)
		return "", err
	}

	// 计算文件的相对路径（用于数据库存储，避免硬编码绝对路径）
	base := filepath.Clean(s.uploadBaseDir())
	absFinal := filepath.Clean(finalPath)
	rel, err := filepath.Rel(base, absFinal)
	if err != nil {
		rel = filepath.Base(absFinal)
	}
	relPath := filepath.ToSlash(rel)

	// 构造文件元数据//数据库存的是文件的id，但是实际的文件在服务器的后端文件夹
	now := time.Now()
	doc := &models.CourseFile{
		ID:          primitive.NewObjectID(),
		CourseID:    courseObjID,
		UploaderID:  userObjID,
		FileName:    fileName,
		ContentType: strings.TrimSpace(req.ContentType),
		Size:        written,
		SHA256:      req.FileSHA256,
		StoragePath: relPath,
		CTime:       now,
		MTime:       now,
	}

	// upsert 防止并发 merge 重复写入
	filter := bson.M{"course_id": courseObjID, "sha256": req.FileSHA256}
	update := bson.M{"$setOnInsert": doc}
	upsert := true
	res, err := utils.GetCollection(courseFilesCollection).UpdateOne(ctx, filter, update, &options.UpdateOptions{Upsert: &upsert})
	if err != nil {
		return "", err
	}

	// 若并发导致记录已存在，查回 id
	if res.UpsertedID == nil {
		var again models.CourseFile
		if err := utils.GetCollection(courseFilesCollection).FindOne(ctx, filter).Decode(&again); err == nil {
			doc.ID = again.ID
		}
	}

	// 清理分片（失败不影响主流程）
	_ = os.RemoveAll(chunksDir)

	return doc.ID.Hex(), nil
}

// ListCourseFiles 查询课程文件列表
func (s *CourseService) ListCourseFiles(ctx context.Context, courseID, userID string) ([]models.CourseFile, error) {
	courseObjID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		return nil, errors.New("无效的课程ID")
	}
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, errors.New("无效的用户ID")
	}

	// 课程成员可见
	if ok, err := courseMemberAuthorized(ctx, utils.GetCollection("courses"), courseObjID, userObjID); err != nil || !ok {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("权限不足")
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "ctime", Value: -1}}).
		SetLimit(2000)
	cur, err := utils.GetCollection(courseFilesCollection).Find(ctx, bson.M{"course_id": courseObjID}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var out []models.CourseFile
	if err := cur.All(ctx, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// GetCourseFileForDownload 下载前获取元数据与绝对路径
// DeleteCourseFile removes course file metadata and best-effort cleans storage.
func (s *CourseService) DeleteCourseFile(ctx context.Context, courseID, userID, fileID string) error {
	courseObjID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		return errors.New("课程ID非法")
	}
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return errors.New("用户ID非法")
	}

	if ok, err := authorized(ctx, utils.GetCollection("courses"), courseObjID, userObjID); err != nil || !ok {
		if err != nil {
			return err
		}
		return errors.New("无权限操作")
	}

	fileObjID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return errors.New("文件ID非法")
	}

	coll := utils.GetCollection(courseFilesCollection)
	var meta models.CourseFile
	if err := coll.FindOne(ctx, bson.M{
		"_id":       fileObjID,
		"course_id": courseObjID,
	}).Decode(&meta); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("文件不存在")
		}
		return err
	}

	if _, err := coll.DeleteOne(ctx, bson.M{
		"_id":       fileObjID,
		"course_id": courseObjID,
	}); err != nil {
		return err
	}

	baseDir := filepath.Clean(s.uploadBaseDir())
	if strings.TrimSpace(meta.StoragePath) != "" {
		absFilePath := filepath.Clean(filepath.Join(baseDir, filepath.FromSlash(meta.StoragePath)))
		if isSubPath(baseDir, absFilePath) {
			if err := os.Remove(absFilePath); err != nil && !os.IsNotExist(err) {
				utils.Logger.Warnf("DeleteCourseFile: remove file failed, course_id=%s, file_id=%s, path=%s, error=%v", courseID, fileID, absFilePath, err)
			}
		} else {
			utils.Logger.Warnf("DeleteCourseFile: skip unsafe file path cleanup, course_id=%s, file_id=%s, path=%s", courseID, fileID, absFilePath)
		}
	}

	shaRootDir := filepath.Clean(filepath.Join(baseDir, "course", courseID, meta.SHA256))
	if isSubPath(baseDir, shaRootDir) {
		if err := os.RemoveAll(shaRootDir); err != nil {
			utils.Logger.Warnf("DeleteCourseFile: cleanup hash directory failed, course_id=%s, file_id=%s, dir=%s, error=%v", courseID, fileID, shaRootDir, err)
		}
	} else {
		utils.Logger.Warnf("DeleteCourseFile: skip unsafe hash directory cleanup, course_id=%s, file_id=%s, dir=%s", courseID, fileID, shaRootDir)
	}

	return nil
}

// GetCourseFileForDownload returns metadata and absolute path for downloading a course file.
func (s *CourseService) GetCourseFileForDownload(ctx context.Context, courseID, userID, fileID string) (*models.CourseFile, string, error) {
	courseObjID, err := primitive.ObjectIDFromHex(courseID)
	if err != nil {
		return nil, "", errors.New("无效的课程ID")
	}
	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		return nil, "", errors.New("无效的用户ID")
	}

	// 课程成员可下载
	if ok, err := courseMemberAuthorized(ctx, utils.GetCollection("courses"), courseObjID, userObjID); err != nil || !ok {
		if err != nil {
			return nil, "", err
		}
		return nil, "", errors.New("权限不足")
	}

	fileObjID, err := primitive.ObjectIDFromHex(fileID)
	if err != nil {
		return nil, "", errors.New("无效的文件ID")
	}

	var meta models.CourseFile
	if err := utils.GetCollection(courseFilesCollection).FindOne(ctx, bson.M{
		"_id":       fileObjID,
		"course_id": courseObjID,
	}).Decode(&meta); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, "", errors.New("文件不存在")
		}
		return nil, "", err
	}

	abs := filepath.Join(s.uploadBaseDir(), filepath.FromSlash(meta.StoragePath))
	if _, err := os.Stat(abs); err != nil {
		return nil, "", errors.New("文件已丢失或不可用")
	}
	return &meta, abs, nil
}

// RelPathUnderUploadDir 将绝对路径转为相对 UPLOAD_DIR 的 POSIX 路径，供 X-Accel-Redirect 与 nginx internal 对齐。
func (s *CourseService) RelPathUnderUploadDir(absPath string) (string, error) {
	base := filepath.Clean(s.uploadBaseDir())
	abs := filepath.Clean(absPath)
	if !isSubPath(base, abs) {
		return "", errors.New("路径非法")
	}
	rel, err := filepath.Rel(base, abs)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", errors.New("路径非法")
	}
	return filepath.ToSlash(rel), nil
}
