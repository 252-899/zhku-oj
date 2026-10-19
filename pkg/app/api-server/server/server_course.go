/*
@Author:sir
@Date: 2025/10/25
@Name: server_course.go
@Description: 课程服务器路由处理
*/

package server

import (
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"

	"github.com/gin-gonic/gin"
)

func (s *Server) RegisterCourse(g *gin.RouterGroup) {
	courseGroup := g.Group("/courses")
	{
		// 需要认证的路由
		jwtGroup := courseGroup.Use(middleware.JWTMiddleware())
		{
			// 课程 CRUD
			jwtGroup.POST("", s.CreateCourse)
			jwtGroup.GET("", s.PageQueryCourse)                  // 查询创建的课程
			jwtGroup.GET("/teaching", s.PageQueryTeacherCourses) // 查询教学的课程（作为教师加入的）
			jwtGroup.GET("/:courseId", s.GetCourseById)
			jwtGroup.PUT("/:courseId", s.UpdateCourse)
			jwtGroup.DELETE("/:courseId", s.RemoveCourse)
			jwtGroup.PUT("/:courseId/avatar", s.UpdateCourseAvatar)
			jwtGroup.GET("/:courseId/problems", s.GetCourseProblems)
			jwtGroup.GET("/:courseId/contents", s.GetCourseContents)

			// 课程成员管理
			jwtGroup.GET("/:courseId/students", s.GetCourseStudents)
			jwtGroup.GET("/:courseId/teachers", s.GetCourseTeachers)
			jwtGroup.POST("/:courseId/teachers", s.addCourseTeacher)
			jwtGroup.DELETE("/:courseId/teachers/:teacherId", s.removeCourseTeacher)

			// 课程文件（分片上传 / 断点续传 / 下载）
			jwtGroup.POST("/:courseId/files/upload/check", s.CourseFileUploadCheck)
			jwtGroup.POST("/:courseId/files/upload/chunk", s.CourseFileUploadChunk)
			jwtGroup.POST("/:courseId/files/upload/merge", s.CourseFileUploadMerge)
			jwtGroup.GET("/:courseId/files", s.ListCourseFiles)
			jwtGroup.DELETE("/:courseId/files/:fileId", s.DeleteCourseFile)
			jwtGroup.GET("/:courseId/files/:fileId/download", s.DownloadCourseFile)
		}
	}
}

// PageQueryCourse godoc
// @Summary      分页查询创建的课程
// @Description  分页查询当前用户创建的课程列表
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        page_num query int false "页码"
// @Param        page_size query int false "每页数量"
// @Param        status query int false "课程状态"
// @Param        name query string false "课程名称"
// @Success      200 {object} models.CourseListResponse "课程列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses [get]
func (s *Server) PageQueryCourse(c *gin.Context) {
	var PageQueryCourseRequest models.PageQueryCourseRequest
	if err := c.ShouldBindQuery(&PageQueryCourseRequest); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	res, err := s.svc.CourseService.PageQueryCourse(c.Request.Context(), &PageQueryCourseRequest, userID.(string))
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, res)
}

// @Summary      课程创建者添加老师
// @Description  课程创建者添加老师（支持批量添加，只有课程创建者可以操作）
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        request body models.AddCourseTeachersRequest true "添加教师请求"
// @Success      200 {object} utils.Response "添加成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId}/teachers [post]
func (s *Server) addCourseTeacher(c *gin.Context) {
	courseId := c.Param("courseId")
	if courseId == "" {
		utils.BadRequestResponse(c, "课程ID不能为空")
		return
	}

	var req models.AddCourseTeachersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	if len(req.TeacherIds) == 0 {
		utils.BadRequestResponse(c, "教师ID列表不能为空")
		return
	}

	userID, _ := c.Get("user_id")
	if err := s.svc.CourseService.AddCourseTeachers(c.Request.Context(), courseId, req.TeacherIds, userID.(string)); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, nil)
}

// @Summary      课程创建者删除课程老师
// @Description  课程创建者删除课程老师（只有课程创建者可以操作）
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        teacherId path string true "教师ID"
// @Success      200 {object} utils.Response "删除成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId}/teachers/{teacherId} [delete]
func (s *Server) removeCourseTeacher(c *gin.Context) {
	courseId := c.Param("courseId")
	teacherId := c.Param("teacherId")

	if courseId == "" || teacherId == "" {
		utils.BadRequestResponse(c, "课程ID或教师ID不能为空")
		return
	}

	userID, _ := c.Get("user_id")
	if err := s.svc.CourseService.RemoveCourseTeachers(c.Request.Context(), courseId, []string{teacherId}, userID.(string)); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, nil)
}

// CreateCourse godoc
// @Summary      创建课程（教师）
// @Description  教师创建新课程
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        request body models.CreateCourseRequest true "课程信息"
// @Success      200 {object} utils.Response{data=string} "创建成功，返回课程ID"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses [post]
func (s *Server) CreateCourse(c *gin.Context) {
	// 检查权限：只有教师和管理员可以创建课程
	role, exists := c.Get("role")
	if !exists {
		utils.UnauthorizedResponse(c, "需要登录")
		return
	}
	userRole := role.(string)
	if userRole != string(models.RoleAdmin) && userRole != string(models.RoleTeacher) {
		utils.ForbiddenResponse(c, "权限不足，只有教师和管理员可以创建课程")
		return
	}

	var req models.CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	courseID, err := s.svc.CourseService.CreateCourse(c.Request.Context(), &req, userID.(string))
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, courseID)
}

// GetCourseById godoc
// @Summary      获取课程详情
// @Description  根据课程ID获取课程详细信息
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Success      200 {object} utils.Response{data=models.Course} "课程详情"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId} [get]
func (s *Server) GetCourseById(c *gin.Context) {
	courseId := c.Param("courseId")
	if courseId == "" {
		utils.BadRequestResponse(c, "课程id不存在")
		return
	}
	userID, _ := c.Get("user_id")
	response, err := s.svc.CourseService.GetCourseByID(c.Request.Context(), courseId, userID.(string))
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, response)
}

// UpdateCourse godoc
// @Summary      更新课程信息
// @Description  更新课程的基本信息
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        request body models.UpdateCourseRequest true "更新的课程信息"
// @Success      200 {object} utils.Response "更新成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId} [put]
func (s *Server) UpdateCourse(c *gin.Context) {
	courseId := c.Param("courseId")
	if courseId == "" {
		utils.BadRequestResponse(c, "课程ID不能为空")
		return
	}

	var req models.UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	userID, _ := c.Get("user_id")

	if err := s.svc.CourseService.UpdateCourseInfo(c.Request.Context(), userID.(string), courseId, &req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, nil)
}

// UpdateCourseAvatar godoc
// @Summary      更新课程头像
// @Description  上传并更新课程头像图片
// @Tags         课程
// @Accept       multipart/form-data
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        file formData file true "头像文件"
// @Success      200 {object} utils.Response "更新成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId}/avatar [put]
func (s *Server) UpdateCourseAvatar(c *gin.Context) {
	// 解析 multipart 请求
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 限制10MB
		utils.BadRequestResponse(c, "图片内存过大")
		return
	}
	courseId := c.Param("courseId")
	if courseId == "" {
		utils.BadRequestResponse(c, "课程id为空")
		return
	}
	// 获取文件头
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	id, _ := c.Get("user_id")
	if err = s.svc.CourseService.UpdateCourseAvatar(c.Request.Context(), file, header, id.(string), courseId); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, nil)
}

// RemoveCourse godoc
// @Summary      删除课程
// @Description  删除指定课程
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Success      200 {object} utils.Response "删除成功"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/{courseId} [delete]
func (s *Server) RemoveCourse(c *gin.Context) {
	courseId := c.Param("courseId")
	if courseId == "" {
		utils.BadRequestResponse(c, "课程id为空")
		return
	}
	userID, _ := c.Get("user_id")
	if err := s.svc.CourseService.RemoveCourse(c.Request.Context(), userID.(string), courseId); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, nil)
}

// GetCourseStudents godoc
// @Summary      分页查询课程下的学生
// @Description  分页查询指定课程下的学生信息，支持按真实姓名搜索（课程成员可访问）
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        page_num query int false "页码"
// @Param        page_size query int false "每页数量"
// @Param        real_name query string false "真实姓名"
// @Param        student_id query string false "学号"
// @Success      200 {object} models.PageQueryCourseStudentsResponse "学生列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/students [get]
func (s *Server) GetCourseStudents(c *gin.Context) {
	courseId := c.Param("courseId")
	if courseId == "" {
		utils.BadRequestResponse(c, "课程ID不能为空")
		return
	}

	var req models.PageQueryCourseStudentsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	students, err := s.svc.CourseService.GetCourseStudents(c.Request.Context(), userID.(string), courseId, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, students)
}

// GetCourseTeachers godoc
// @Summary      分页查询课程下的教师
// @Description  分页查询指定课程下的教师信息，支持按真实姓名和教师ID搜索（课程成员可访问）
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        courseId path string true "课程ID"
// @Param        page_num query int false "页码"
// @Param        page_size query int false "每页数量"
// @Param        real_name query string false "真实姓名"
// @Param        teacher_id query string false "教师ID"
// @Success      200 {object} models.PageQueryCourseTeachersResponse "教师列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Failure      403 {object} models.ErrorResponse "权限不足"
// @Security     BearerAuth
// @Router       /courses/{courseId}/teachers [get]
func (s *Server) GetCourseTeachers(c *gin.Context) {
	courseId := c.Param("courseId")
	if courseId == "" {
		utils.BadRequestResponse(c, "课程ID不能为空")
		return
	}

	var req models.PageQueryCourseTeachersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	userID, _ := c.Get("user_id")
	teachers, err := s.svc.CourseService.GetCourseTeachers(c.Request.Context(), userID.(string), courseId, &req)
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}

	utils.SuccessResponse(c, teachers)
}

// PageQueryTeacherCourses godoc
// @Summary      分页查询教学的课程
// @Description  分页查询当前用户教学的所有课程（包括创建的课程和作为教师加入的课程）
// @Tags         课程
// @Accept       json
// @Produce      json
// @Param        page_num query int false "页码"
// @Param        page_size query int false "每页数量"
// @Param        status query int false "课程状态"
// @Param        name query string false "课程名称"
// @Success      200 {object} models.CourseListResponse "课程列表"
// @Failure      400 {object} models.ErrorResponse "请求参数错误"
// @Security     BearerAuth
// @Router       /courses/teaching [get]
func (s *Server) PageQueryTeacherCourses(c *gin.Context) {
	var req models.PageQueryTeacherCoursesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	userID, _ := c.Get("user_id")
	res, err := s.svc.CourseService.PageQueryTeacherCourses(c.Request.Context(), &req, userID.(string))
	if err != nil {
		utils.BadRequestResponse(c, err.Error())
		return
	}
	utils.SuccessResponse(c, res)
}
