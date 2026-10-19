package models

// PageQueryUserTasksFilterRequest defines query params for /user/tasks.
type PageQueryUserTasksFilterRequest struct {
	PageNum  *int64  `json:"page_num,omitempty" form:"page_num"`
	PageSize *int64  `json:"page_size,omitempty" form:"page_size"`
	State    *int    `json:"state,omitempty" form:"state"`
	Type     *int8   `json:"type,omitempty" form:"type"`
	CourseID *string `json:"course_id,omitempty" form:"course_id"`
	ClazzID  *string `json:"clazz_id,omitempty" form:"clazz_id"`
	Status   *int8   `json:"status,omitempty" form:"status"`
}
