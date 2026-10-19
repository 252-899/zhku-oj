/*
@Author: omenkk7
@Date: 2025/10/25
@Description: 每日一题服务 - 提供每日题目推荐和缓存机制
*/

package service

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"zk-code-arena-server/pkg/app/api-server/repository"
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"
)

// DailyProblemService 每日一题服务
type DailyProblemService struct {
	problemRepo *repository.ProblemRepository
	// 缓存相关
	cache       *models.ProblemList  // 缓存的每日推荐题目
	lastUpdate  time.Time           // 最后更新时间
	cacheExpiry time.Duration       // 缓存过期时间 (24小时)
	mutex       sync.RWMutex        // 读写锁保证并发安全
}

// NewDailyProblemService 创建每日一题服务实例
func NewDailyProblemService(problemRepo *repository.ProblemRepository) *DailyProblemService {
	return &DailyProblemService{
		problemRepo: problemRepo,
		cacheExpiry: 24 * time.Hour, // 24小时缓存
		mutex:       sync.RWMutex{},
	}
}

// GetDailyProblem 获取每日推荐题目
func (s *DailyProblemService) GetDailyProblem(ctx context.Context, userID *primitive.ObjectID) (*models.ProblemList, error) {
	// 尝试从缓存获取
	dailyProblem, err := s.getCachedDailyProblem(ctx)
	if err != nil {
		utils.Logger.Errorf("GetDailyProblem: 获取缓存失败, error=%v", err)
		return nil, fmt.Errorf("获取每日推荐失败: %w", err)
	}

	// 如果需要用户状态，则查询并设置
	if userID != nil && dailyProblem != nil {
		userStatusMap, err := s.problemRepo.GetUserProblemStatuses(ctx, *userID, []primitive.ObjectID{dailyProblem.ID})
		if err != nil {
			utils.Logger.Warnf("GetDailyProblem: 获取用户状态失败, userID=%s, error=%v", userID.Hex(), err)
			// 不影响主要功能，继续返回题目
		} else if status, exists := userStatusMap[dailyProblem.ID]; exists {
			dailyProblem.UserStatus = &status
		}
	}

	utils.Logger.Infof("GetDailyProblem: 返回每日推荐题目, problemID=%s, userID=%v", 
		dailyProblem.ID.Hex(), userID)
	return dailyProblem, nil
}

// getCachedDailyProblem 获取缓存的每日推荐（线程安全）
func (s *DailyProblemService) getCachedDailyProblem(ctx context.Context) (*models.ProblemList, error) {
	// 先尝试读取缓存
	s.mutex.RLock()
	if s.cache != nil && s.isCacheValid() {
		cachedProblem := s.cache // 复制引用
		s.mutex.RUnlock()
		utils.Logger.Debugf("getCachedDailyProblem: 缓存命中, problemID=%s", cachedProblem.ID.Hex())
		return cachedProblem, nil
	}
	s.mutex.RUnlock()

	// 缓存过期或不存在，获取写锁进行刷新
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 双重检查，防止并发刷新
	if s.cache != nil && s.isCacheValid() {
		utils.Logger.Debugf("getCachedDailyProblem: 双重检查缓存命中, problemID=%s", s.cache.ID.Hex())
		return s.cache, nil
	}

	// 刷新缓存
	utils.Logger.Infof("getCachedDailyProblem: 缓存过期或不存在，开始刷新")
	if err := s.refreshDailyRecommendationUnsafe(ctx); err != nil {
		return nil, fmt.Errorf("刷新每日推荐失败: %w", err)
	}

	return s.cache, nil
}

// isCacheValid 检查缓存是否有效（需要在锁保护下调用）
func (s *DailyProblemService) isCacheValid() bool {
	return time.Since(s.lastUpdate) < s.cacheExpiry
}

// refreshDailyRecommendationUnsafe 刷新每日推荐（非线程安全，需要在锁保护下调用）
func (s *DailyProblemService) refreshDailyRecommendationUnsafe(ctx context.Context) error {
	utils.Logger.Infof("refreshDailyRecommendation: 开始刷新每日推荐")

	// 查询所有已发布的公开题目
	// 使用Repository的标准查询方法
	problems, _, err := s.problemRepo.GetProblemsWithUserStatus(
		ctx,
		1,                               // page
		50,                              // pageSize - 获取50个候选题目提高随机性
		"",                              // difficulty - 不限制难度
		[]string{},                      // tags - 不限制标签
		false,                           // includePrivate - 只要公开题目
		models.RoleStudent,              // role - 学生角色
		nil,                             // userID - 不获取用户状态
	)
	if err != nil {
		utils.Logger.Errorf("refreshDailyRecommendation: 查询题目失败, error=%v", err)
		return fmt.Errorf("查询候选题目失败: %w", err)
	}

	if len(problems) == 0 {
		utils.Logger.Warnf("refreshDailyRecommendation: 没有可推荐的题目")
		return fmt.Errorf("没有可推荐的题目")
	}

	// 使用当天日期作为随机种子，确保同一天返回相同题目
	today := time.Now().Format("2006-01-02")
	seed := int64(0)
	for _, b := range []byte(today) {
		seed += int64(b)
	}
	
	rng := rand.New(rand.NewSource(seed))
	selectedIndex := rng.Intn(len(problems))
	selectedProblem := problems[selectedIndex]

	// 更新缓存
	s.cache = selectedProblem
	s.lastUpdate = time.Now()

	utils.Logger.Infof("refreshDailyRecommendation: 每日推荐更新成功, problemID=%s, title=%s", 
		selectedProblem.ID.Hex(), selectedProblem.Title)
	return nil
}

// RefreshDailyRecommendation 手动刷新每日推荐（公开方法，用于管理接口）
func (s *DailyProblemService) RefreshDailyRecommendation(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	return s.refreshDailyRecommendationUnsafe(ctx)
}

// GetCacheStatus 获取缓存状态（用于监控和调试）
func (s *DailyProblemService) GetCacheStatus() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	status := map[string]interface{}{
		"has_cache":    s.cache != nil,
		"last_update":  s.lastUpdate,
		"cache_expiry": s.cacheExpiry,
		"is_valid":     s.isCacheValid(),
	}

	if s.cache != nil {
		status["cached_problem_id"] = s.cache.ID.Hex()
		status["cached_problem_title"] = s.cache.Title
	}

	return status
}
