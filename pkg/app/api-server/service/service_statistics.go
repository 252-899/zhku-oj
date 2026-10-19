/*
@Author: omenkk7
@Date: 2025/10/7
@Description: 统计服务
*/

package service

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
)

// StatisticsService 统计服务
type StatisticsService struct {
	submitService  *SubmitService
	problemService *ProblemService
	userService    *UserService
	problemRepo    *repository.ProblemRepository
	// 难度统计缓存
	difficultyStatsCache *DifficultyStatsCache
}

// DifficultyStatsCache 难度统计缓存
type DifficultyStatsCache struct {
	Data       *ProblemDifficultyStats
	LastUpdate time.Time
	Mutex      sync.RWMutex
	TTL        time.Duration // 1小时TTL
}

// ProblemDifficultyStats 题目难度分布统计
type ProblemDifficultyStats struct {
	Easy   int64 `json:"easy"`   // 简单题数量
	Medium int64 `json:"medium"` // 中等题数量
	Hard   int64 `json:"hard"`   // 困难题数量
	Total  int64 `json:"total"`  // 总题目数量
}

// NewStatisticsService 创建统计服务实例
func NewStatisticsService(submitService *SubmitService, problemService *ProblemService, userService *UserService, problemRepo *repository.ProblemRepository) *StatisticsService {
	return &StatisticsService{
		submitService:  submitService,
		problemService: problemService,
		userService:    userService,
		problemRepo:    problemRepo,
		difficultyStatsCache: &DifficultyStatsCache{
			TTL: time.Hour, // 1小时TTL
		},
	}
}

// UserStatistics 用户统计数据
type UserStatistics struct {
	TotalSubmits    int64              `json:"total_submits"`    // 总提交数
	ACSubmits       int64              `json:"ac_submits"`       // AC提交数
	TotalProblems   int64              `json:"total_problems"`   // 尝试过的题目数
	SolvedProblems  int64              `json:"solved_problems"`  // 通过的题目数
	AcceptanceRate  float64            `json:"acceptance_rate"`  // 通过率（%）
	RecentSubmits   []RecentSubmitInfo `json:"recent_submits"`   // 最近提交（10条）
	LanguageStats   map[string]int     `json:"language_stats"`   // 各语言提交统计
	DifficultyStats map[string]int     `json:"difficulty_stats"` // 各难度通过统计
}

// RecentSubmitInfo 最近提交信息
type RecentSubmitInfo struct {
	SubmitID     primitive.ObjectID  `json:"submit_id"`
	ProblemID    primitive.ObjectID  `json:"problem_id"`
	ProblemTitle string              `json:"problem_title"`
	Language     string              `json:"language"`
	Status       models.SubmitStatus `json:"status"`
	SubmittedAt  time.Time           `json:"submitted_at"`
}

// GetUserStatistics 获取用户统计信息
func (s *StatisticsService) GetUserStatistics(ctx context.Context, userID primitive.ObjectID) (*UserStatistics, error) {
	submitsCollection := utils.GetCollection("submits")

	stats := &UserStatistics{
		LanguageStats:   make(map[string]int),
		DifficultyStats: make(map[string]int),
	}

	// 1. 获取总提交数
	totalSubmits, err := submitsCollection.CountDocuments(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	stats.TotalSubmits = totalSubmits

	// 2. 获取AC提交数
	acSubmits, err := submitsCollection.CountDocuments(ctx, bson.M{
		"user_id":       userID,
		"result.status": models.StatusAccepted,
	})
	if err != nil {
		return nil, err
	}
	stats.ACSubmits = acSubmits

	// 3. 计算通过率
	if totalSubmits > 0 {
		stats.AcceptanceRate = float64(acSubmits) / float64(totalSubmits) * 100
	}

	// 4. 获取尝试过的题目数（去重）
	problemIDs, err := submitsCollection.Distinct(ctx, "problem_id", bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	stats.TotalProblems = int64(len(problemIDs))

	// 5. 获取通过的题目数（去重）
	solvedProblemIDs, err := submitsCollection.Distinct(ctx, "problem_id", bson.M{
		"user_id":       userID,
		"result.status": models.StatusAccepted,
	})
	if err != nil {
		return nil, err
	}
	stats.SolvedProblems = int64(len(solvedProblemIDs))

	// 6. 获取最近提交（10条）
	recentSubmits, err := s.getRecentSubmits(ctx, userID, 10)
	if err != nil {
		return nil, err
	}
	stats.RecentSubmits = recentSubmits

	// 7. 获取语言统计
	languageStats, err := s.getLanguageStats(ctx, userID)
	if err != nil {
		return nil, err
	}
	stats.LanguageStats = languageStats

	// 8. 获取难度统计（通过的题目按难度分类）
	difficultyStats, err := s.getDifficultyStats(ctx, userID, solvedProblemIDs)
	if err != nil {
		return nil, err
	}
	stats.DifficultyStats = difficultyStats

	return stats, nil
}

// getRecentSubmits 获取最近提交
func (s *StatisticsService) getRecentSubmits(ctx context.Context, userID primitive.ObjectID, limit int) ([]RecentSubmitInfo, error) {
	submitsCollection := utils.GetCollection("submits")
	problemsCollection := utils.GetCollection("problems")

	// 查询最近的提交
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit))
	cursor, err := submitsCollection.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var recentSubmits []RecentSubmitInfo
	for cursor.Next(ctx) {
		var submit models.Submit
		if err := cursor.Decode(&submit); err != nil {
			continue
		}

		// 获取题目标题
		var problem models.Problem
		problemsCollection.FindOne(ctx, bson.M{"_id": submit.ProblemID}).Decode(&problem)

		recentSubmits = append(recentSubmits, RecentSubmitInfo{
			SubmitID:     submit.ID,
			ProblemID:    submit.ProblemID,
			ProblemTitle: problem.Title,
			Language:     string(submit.Language),
			Status:       submit.Result.Status,
			SubmittedAt:  submit.CreatedAt,
		})
	}

	return recentSubmits, nil
}

// getLanguageStats 获取语言统计
func (s *StatisticsService) getLanguageStats(ctx context.Context, userID primitive.ObjectID) (map[string]int, error) {
	submitsCollection := utils.GetCollection("submits")

	// 使用聚合查询统计各语言提交数
	pipeline := []bson.M{
		{"$match": bson.M{"user_id": userID}},
		{"$group": bson.M{
			"_id":   "$language",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := submitsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	languageStats := make(map[string]int)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		languageStats[result.ID] = result.Count
	}

	return languageStats, nil
}

// getDifficultyStats 获取难度统计
func (s *StatisticsService) getDifficultyStats(ctx context.Context, userID primitive.ObjectID, solvedProblemIDs []interface{}) (map[string]int, error) {
	problemsCollection := utils.GetCollection("problems")

	if len(solvedProblemIDs) == 0 {
		return map[string]int{}, nil
	}

	// 查询已通过题目的难度
	cursor, err := problemsCollection.Find(ctx, bson.M{"_id": bson.M{"$in": solvedProblemIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	difficultyStats := make(map[string]int)
	for cursor.Next(ctx) {
		var problem models.Problem
		if err := cursor.Decode(&problem); err != nil {
			continue
		}
		difficultyStats[string(problem.Difficulty)]++
	}

	return difficultyStats, nil
}

// SystemStatistics 系统统计数据
type SystemStatistics struct {
	TotalUsers     int64 `json:"total_users"`      // 用户总数
	TotalProblems  int64 `json:"total_problems"`   // 题目总数
	TotalSubmits   int64 `json:"total_submits"`    // 提交总数
	TotalACSubmits int64 `json:"total_ac_submits"` // AC提交总数

	// 各难度题目数量
	DifficultyDistribution map[string]int64 `json:"difficulty_distribution"`

	// 最近24小时统计
	RecentSubmitsCount int64 `json:"recent_submits_count"` // 24小时内提交数
	RecentUsersCount   int64 `json:"recent_users_count"`   // 24小时内活跃用户数
}

// GetSystemStatistics 获取系统统计信息
func (s *StatisticsService) GetSystemStatistics(ctx context.Context) (*SystemStatistics, error) {
	usersCollection := utils.GetCollection("users")
	problemsCollection := utils.GetCollection("problems")
	submitsCollection := utils.GetCollection("submits")

	stats := &SystemStatistics{
		DifficultyDistribution: make(map[string]int64),
	}

	// 1. 用户总数
	totalUsers, err := usersCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	stats.TotalUsers = totalUsers

	// 2. 题目总数
	totalProblems, err := problemsCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	stats.TotalProblems = totalProblems

	// 3. 提交总数
	totalSubmits, err := submitsCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	stats.TotalSubmits = totalSubmits

	// 4. AC提交总数
	totalACSubmits, err := submitsCollection.CountDocuments(ctx, bson.M{
		"result.status": models.StatusAccepted,
	})
	if err != nil {
		return nil, err
	}
	stats.TotalACSubmits = totalACSubmits

	// 5. 各难度题目分布
	difficultyDistribution, err := s.getDifficultyDistribution(ctx)
	if err != nil {
		return nil, err
	}
	stats.DifficultyDistribution = difficultyDistribution

	// 6. 最近24小时提交数
	recentTime := time.Now().Add(-24 * time.Hour)
	recentSubmitsCount, err := submitsCollection.CountDocuments(ctx, bson.M{
		"submitted_at": bson.M{"$gte": recentTime},
	})
	if err != nil {
		return nil, err
	}
	stats.RecentSubmitsCount = recentSubmitsCount

	// 7. 最近24小时活跃用户数
	recentUserIDs, err := submitsCollection.Distinct(ctx, "user_id", bson.M{
		"submitted_at": bson.M{"$gte": recentTime},
	})
	if err != nil {
		return nil, err
	}
	stats.RecentUsersCount = int64(len(recentUserIDs))

	return stats, nil
}

// getDifficultyDistribution 获取各难度题目分布
func (s *StatisticsService) getDifficultyDistribution(ctx context.Context) (map[string]int64, error) {
	problemsCollection := utils.GetCollection("problems")

	// 使用聚合查询统计各难度题目数
	pipeline := []bson.M{
		{"$group": bson.M{
			"_id":   "$difficulty",
			"count": bson.M{"$sum": 1},
		}},
	}

	cursor, err := problemsCollection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	difficultyDistribution := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			continue
		}
		difficultyDistribution[result.ID] = result.Count
	}

	return difficultyDistribution, nil
}

// GetProblemDifficultyStats 获取题目难度分布统计（带缓存）
func (s *StatisticsService) GetProblemDifficultyStats(ctx context.Context) (*ProblemDifficultyStats, error) {
	// 检查缓存
	s.difficultyStatsCache.Mutex.RLock()
	if s.difficultyStatsCache.Data != nil && 
		time.Since(s.difficultyStatsCache.LastUpdate) < s.difficultyStatsCache.TTL {
		// 缓存命中且未过期
		cached := s.difficultyStatsCache.Data
		s.difficultyStatsCache.Mutex.RUnlock()
		utils.Logger.Infof("GetProblemDifficultyStats: 缓存命中")
		return cached, nil
	}
	s.difficultyStatsCache.Mutex.RUnlock()

	// 缓存未命中或已过期，需要查询数据库
	utils.Logger.Infof("GetProblemDifficultyStats: 缓存未命中，查询数据库")
	
	// 使用Repository层进行聚合查询
	collection := utils.GetCollection("problems")
	
	// MongoDB聚合管道：统计各难度已发布公开题目数量
	pipeline := []bson.M{
		// 只统计已发布且公开的题目
		{"$match": bson.M{
			"status":    models.StatusPublished,
			"is_public": true,
		}},
		// 按难度分组统计
		{"$group": bson.M{
			"_id":   "$difficulty",
			"count": bson.M{"$sum": 1},
		}},
	}
	
	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		utils.Logger.Errorf("GetProblemDifficultyStats: 聚合查询失败, error=%v", err)
		return nil, err
	}
	defer cursor.Close(ctx)
	
	// 解析结果
	stats := &ProblemDifficultyStats{}
	for cursor.Next(ctx) {
		var result struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := cursor.Decode(&result); err != nil {
			utils.Logger.Warnf("GetProblemDifficultyStats: 解析结果失败, error=%v", err)
			continue
		}
		
		switch result.ID {
		case string(models.DifficultyEasy):
			stats.Easy = result.Count
		case string(models.DifficultyMedium):
			stats.Medium = result.Count
		case string(models.DifficultyHard):
			stats.Hard = result.Count
		}
	}
	
	// 计算总数
	stats.Total = stats.Easy + stats.Medium + stats.Hard
	
	// 更新缓存
	s.difficultyStatsCache.Mutex.Lock()
	s.difficultyStatsCache.Data = stats
	s.difficultyStatsCache.LastUpdate = time.Now()
	s.difficultyStatsCache.Mutex.Unlock()
	
	utils.Logger.Infof("GetProblemDifficultyStats: 查询完成, easy=%d, medium=%d, hard=%d, total=%d", 
		stats.Easy, stats.Medium, stats.Hard, stats.Total)
	
	return stats, nil
}

