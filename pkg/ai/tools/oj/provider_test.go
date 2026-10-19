package oj

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestProviderBuildsAvailableTools(t *testing.T) {
	provider := NewProvider(Dependencies{
		UserReader:    fakeUserReader{},
		ClazzReader:   fakeClazzReader{},
		ProblemReader: fakeProblemReader{},
	}, primitive.NewObjectID().Hex())

	tools, err := provider.BuildTools(context.Background())
	if err != nil {
		t.Fatalf("BuildTools() error = %v", err)
	}
	if len(tools) != 9 {
		t.Fatalf("expected 9 tools, got %d", len(tools))
	}
}

func TestExecuteGetUserProfileReturnsSafeFields(t *testing.T) {
	userID := primitive.NewObjectID()
	provider := NewProvider(Dependencies{UserReader: fakeUserReader{userID: userID}}, primitive.NewObjectID().Hex())

	result, err := provider.Execute(context.Background(), ToolGetUserProfile, `{"student_id":"20240001"}`)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if strings.Contains(result, "password") || strings.Contains(result, "phone") || strings.Contains(result, "email") {
		t.Fatalf("expected safe profile without sensitive fields, got %s", result)
	}

	var output UserSummary
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if output.StudentID != "20240001" {
		t.Fatalf("unexpected student id: %#v", output)
	}
}

func TestExecuteTaskCompletionUsesActorAndSummarizesPage(t *testing.T) {
	actorID := primitive.NewObjectID()
	provider := NewProvider(Dependencies{ClazzReader: fakeClazzReader{}}, actorID.Hex())

	result, err := provider.Execute(context.Background(), ToolGetTaskCompletion, `{"class_id":"`+primitive.NewObjectID().Hex()+`","task_id":"`+primitive.NewObjectID().Hex()+`","page_size":2}`)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var output TaskCompletionSummary
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if output.Summary.CompletedCount != 1 || output.Summary.UncompletedCount != 1 {
		t.Fatalf("unexpected summary: %#v", output.Summary)
	}
}

type fakeUserReader struct {
	userID primitive.ObjectID
}

func (f fakeUserReader) GetUserByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	return f.user(id), nil
}

func (f fakeUserReader) GetUserByStudentID(ctx context.Context, stdId string) (*models.User, error) {
	id := f.userID
	if id.IsZero() {
		id = primitive.NewObjectID()
	}
	user := f.user(id)
	user.StudentID = stdId
	return user, nil
}

func (f fakeUserReader) user(id primitive.ObjectID) *models.User {
	return &models.User{
		ID:        id,
		Username:  "student1",
		Password:  "secret",
		Email:     "student@example.com",
		Phone:     "13800138000",
		RealName:  "张同学",
		StudentID: "20240001",
		Role:      models.RoleStudent,
		IsActive:  true,
	}
}

type fakeClazzReader struct{}

func (fakeClazzReader) ListTeacherClasses(ctx context.Context, userID string) ([]models.Clazz, error) {
	return []models.Clazz{{ID: primitive.NewObjectID(), Name: "23 软件 1 班", MemberIDs: []primitive.ObjectID{primitive.NewObjectID()}}}, nil
}

func (fakeClazzReader) GetClazzByID(ctx context.Context, clazzID string, userID string) (*models.GetClazzResponse, error) {
	classID, _ := primitive.ObjectIDFromHex(clazzID)
	return &models.GetClazzResponse{
		ID:         classID,
		Name:       "23 软件 1 班",
		AddNums:    2,
		MaxMembers: 60,
		Status:     models.ClassStatusActive,
		CTime:      time.Now(),
	}, nil
}

func (fakeClazzReader) GetClassStudents(ctx context.Context, classID string) ([]*models.UserProfile, error) {
	return []*models.UserProfile{{ID: primitive.NewObjectID(), RealName: "张同学", StudentID: "20240001", Role: models.RoleStudent}}, nil
}

func (fakeClazzReader) GetTasksByClazzID(ctx context.Context, clazzID string, userID string) ([]models.Task, error) {
	return []models.Task{{ID: primitive.NewObjectID(), Title: "数组作业", Type: models.TaskTypeProblem}}, nil
}

func (fakeClazzReader) GetTaskByID(ctx context.Context, taskID string, userID string) (*models.TaskResponse, error) {
	id, _ := primitive.ObjectIDFromHex(taskID)
	return &models.TaskResponse{ID: id, Title: "数组作业", Questions: []models.QuestionDetail{{Title: "两数之和", Difficulty: "easy"}}}, nil
}

func (fakeClazzReader) PageQueryTaskCompletion(ctx context.Context, clazzID string, taskID string, req *models.PageQueryTaskCompletionRequest) (*models.PageQueryTaskCompletionResponse, error) {
	return &models.PageQueryTaskCompletionResponse{
		Total:    2,
		PageNum:  *req.PageNum,
		PageSize: *req.PageSize,
		Completion: []models.TaskCompletionResponse{
			{UserID: primitive.NewObjectID(), RealName: "张同学", StudentID: "20240001", Completed: true, TotalQuestions: 2, FinishedQuestions: 2},
			{UserID: primitive.NewObjectID(), RealName: "李同学", StudentID: "20240002", Completed: false, TotalQuestions: 2, FinishedQuestions: 1},
		},
	}, nil
}

type fakeProblemReader struct{}

func (fakeProblemReader) SearchProblems(ctx context.Context, keyword string, difficulty models.ProblemDifficulty, tags []string, page, pageSize int, userID *primitive.ObjectID, inFavorite bool) ([]*models.ProblemList, int64, error) {
	return []*models.ProblemList{{ID: primitive.NewObjectID(), Title: "两数之和", Difficulty: models.DifficultyEasy}}, 1, nil
}

func (fakeProblemReader) GetProblemDetail(ctx context.Context, problemID primitive.ObjectID) (*models.ProblemDetailResponse, error) {
	return &models.ProblemDetailResponse{Problem: &models.Problem{ID: problemID, Title: "两数之和"}}, nil
}
