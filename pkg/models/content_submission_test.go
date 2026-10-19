/*
@Author: Codex
@Date: 2026/05/08
@Description: 统一内容提交记录模型测试
*/

package models

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestContentSubmissionValidate(t *testing.T) {
	taskID := primitive.NewObjectID()
	relationID := primitive.NewObjectID()

	testCases := []struct {
		name       string
		submission *ContentSubmission
		wantErr    bool
	}{
		{
			name: "standalone without task relation",
			submission: &ContentSubmission{
				SubmissionMode: SubmissionModeStandalone,
			},
			wantErr: false,
		},
		{
			name: "standalone rejects task relation",
			submission: &ContentSubmission{
				SubmissionMode: SubmissionModeStandalone,
				TaskID:         &taskID,
			},
			wantErr: true,
		},
		{
			name: "task requires both task ids",
			submission: &ContentSubmission{
				SubmissionMode: SubmissionModeTask,
				TaskID:         &taskID,
			},
			wantErr: true,
		},
		{
			name: "task accepts both task ids",
			submission: &ContentSubmission{
				SubmissionMode:        SubmissionModeTask,
				TaskID:                &taskID,
				TaskContentRelationID: &relationID,
			},
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		if err := tc.submission.Validate(); (err != nil) != tc.wantErr {
			t.Fatalf("%s: expected wantErr=%v, got err=%v", tc.name, tc.wantErr, err)
		}
	}
}
