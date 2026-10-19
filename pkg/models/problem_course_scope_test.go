package models

import (
	"encoding/json"
	"testing"
)

func TestCreateProblemRequestSupportsCourseIDsAndDefaultsToPublic(t *testing.T) {
	payload := []byte(`{
		"title":"数组求和",
		"description":"给定一个数组，计算所有元素之和并输出结果。",
		"difficulty":"easy",
		"course_ids":["507f1f77bcf86cd799439011"]
	}`)

	var req CreateProblemRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("unmarshal CreateProblemRequest failed: %v", err)
	}

	if len(req.CourseIDs) != 1 || req.CourseIDs[0] != "507f1f77bcf86cd799439011" {
		t.Fatalf("expected course_ids to be parsed, got %#v", req.CourseIDs)
	}
	if !req.EffectiveIsPublic() {
		t.Fatalf("expected omitted is_public to default to true")
	}
}

func TestUpdateProblemRequestSupportsClearingCourseIDs(t *testing.T) {
	payload := []byte(`{"course_ids":[]}`)

	var req UpdateProblemRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		t.Fatalf("unmarshal UpdateProblemRequest failed: %v", err)
	}

	if req.CourseIDs == nil {
		t.Fatalf("expected course_ids pointer to distinguish omitted and explicit empty list")
	}
	if len(*req.CourseIDs) != 0 {
		t.Fatalf("expected explicit empty course_ids, got %#v", *req.CourseIDs)
	}
}
