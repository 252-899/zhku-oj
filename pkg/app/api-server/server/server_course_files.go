/*
@Author: omenkk7
@Date: 2026/03/17
@Description: 课程文件（分片上传/断点续传/下载）路由处理
*/

package server

import (
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"github.com/gin-gonic/gin"
)

// CourseFileUploadCheck godoc
// @Summary      课程文件上传检查（秒传/断点续传）
// @Description  根据 file_sha256 返回文件是否已存在，以及已上传的分片列表
// @Tags         课程文件
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        request body models.CourseFileUploadCheckRequest true "检查请求"
// @Success      200 {object} utils.Response{data=models.CourseFileUploadCheckResponse} "检查结果"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId}/files/upload/check [post]
func (s *Server) CourseFileUploadCheck(c *gin.Context) {
	courseID := c.Param("courseId")
	if courseID == "" {
		utils.BadRequestResponse(c, "课程ID不能为空")
		return
	}

	var req models.CourseFileUploadCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	res, err := s.svc.CourseService.CheckCourseFileUpload(c.Request.Context(), courseID, userID.(string), &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, res)
}

// CourseFileUploadChunk godoc
// @Summary      上传课程文件分片
// @Description  上传单个 chunk（multipart/form-data），支持重复上传（幂等）
// @Tags         课程文件
// @Accept       multipart/form-data
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        file_sha256 formData string true "文件SHA256"
// @Param        file_name formData string true "文件名"
// @Param        chunk_index formData int true "分片序号（从0开始）"
// @Param        total_chunks formData int true "总分片数"
// @Param        chunk formData file true "分片文件"
// @Success      200 {object} utils.Response "上传成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId}/files/upload/chunk [post]
func (s *Server) CourseFileUploadChunk(c *gin.Context) {
	courseID := c.Param("courseId")
	if courseID == "" {
		utils.BadRequestResponse(c, "课程ID不能为空")
		return
	}

	fileSHA256 := c.PostForm("file_sha256")
	fileName := c.PostForm("file_name")
	chunkIndexStr := c.PostForm("chunk_index")
	totalChunksStr := c.PostForm("total_chunks")

	if fileSHA256 == "" || fileName == "" || chunkIndexStr == "" || totalChunksStr == "" {
		utils.BadRequestResponse(c, "缺少必要参数")
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIndexStr)
	if err != nil || chunkIndex < 0 {
		utils.BadRequestResponse(c, "chunk_index 不合法")
		return
	}
	totalChunks, err := strconv.Atoi(totalChunksStr)
	if err != nil || totalChunks <= 0 {
		utils.BadRequestResponse(c, "total_chunks 不合法")
		return
	}

	file, header, err := c.Request.FormFile("chunk")
	if err != nil {
		utils.BadRequestResponse(c, "读取分片失败: "+err.Error())
		return
	}
	defer file.Close()

	userID, _ := c.Get("user_id")
	if err := s.svc.CourseService.UploadCourseFileChunk(
		c.Request.Context(),
		courseID,
		userID.(string),
		fileSHA256,
		fileName,
		chunkIndex,
		totalChunks,
		header,
		file,
	); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, nil)
}

// CourseFileUploadMerge godoc
// @Summary      合并课程文件分片
// @Description  所有 chunk 上传完成后调用，服务端流式合并并校验 SHA256
// @Tags         课程文件
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        request body models.CourseFileUploadMergeRequest true "合并请求"
// @Success      200 {object} utils.Response{data=models.CourseFileUploadMergeResponse} "合并成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId}/files/upload/merge [post]
func (s *Server) CourseFileUploadMerge(c *gin.Context) {
	courseID := c.Param("courseId")
	if courseID == "" {
		utils.BadRequestResponse(c, "课程ID不能为空")
		return
	}

	var req models.CourseFileUploadMergeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	fileID, err := s.svc.CourseService.MergeCourseFileChunks(c.Request.Context(), courseID, userID.(string), &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, models.CourseFileUploadMergeResponse{FileID: fileID})
}

// ListCourseFiles godoc
// @Summary      查询课程文件列表
// @Tags         课程文件
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Success      200 {object} utils.Response{data=[]models.CourseFile} "文件列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId}/files [get]
func (s *Server) ListCourseFiles(c *gin.Context) {
	courseID := c.Param("courseId")
	if courseID == "" {
		utils.BadRequestResponse(c, "课程ID不能为空")
		return
	}

	userID, _ := c.Get("user_id")
	files, err := s.svc.CourseService.ListCourseFiles(c.Request.Context(), courseID, userID.(string))
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, files)
}

// DeleteCourseFile godoc
// @Summary      删除课程文件
// @Tags         课程文件
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        fileId path string true "文件ID"
// @Success      200 {object} utils.Response "删除成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId}/files/{fileId} [delete]
func (s *Server) DeleteCourseFile(c *gin.Context) {
	courseID := c.Param("courseId")
	fileID := c.Param("fileId")
	if courseID == "" || fileID == "" {
		utils.BadRequestResponse(c, "请求参数不能为空")
		return
	}

	userID, _ := c.Get("user_id")
	if err := s.svc.CourseService.DeleteCourseFile(c.Request.Context(), courseID, userID.(string), fileID); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, nil)
}

// DownloadCourseFile godoc
// @Summary      下载课程文件
// @Tags         课程文件
// @Produce      application/octet-stream
// @Param        courseId path string true "课程ID"
// @Param        fileId path string true "文件ID"
// @Success      200 {file} file "文件流"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId}/files/{fileId}/download [get]
func (s *Server) DownloadCourseFile(c *gin.Context) {
	courseID := c.Param("courseId")
	fileID := c.Param("fileId")
	if courseID == "" || fileID == "" {
		utils.BadRequestResponse(c, "参数不能为空")
		return
	}

	userID, _ := c.Get("user_id")
	meta, absPath, err := s.svc.CourseService.GetCourseFileForDownload(c.Request.Context(), courseID, userID.(string), fileID)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	// 兼容中文文件名：让浏览器用 filename*=UTF-8'' 解析
	c.Header("Content-Type", meta.ContentType)
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(meta.FileName))

	// 可选：由 nginx internal + sendfile 直出文件（需与 nginx 中 location 前缀、alias 物理目录一致）
	if prefix := strings.TrimSpace(os.Getenv("COURSE_FILE_X_ACCEL_PREFIX")); prefix != "" {
		rel, err := s.svc.CourseService.RelPathUnderUploadDir(absPath)
		if err != nil {
			utils.BadRequestResponse(c, err.Error())
			return
		}
		internalURI := strings.TrimSuffix(prefix, "/") + "/" + rel
		c.Header("X-Accel-Redirect", internalURI)
		c.Status(http.StatusOK)
		return
	}

	c.File(absPath)
}
