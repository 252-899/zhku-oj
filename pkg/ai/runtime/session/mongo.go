package session

import (
	"context"
	"time"

	aiapi "zk-code-arena-server/pkg/ai/api"
	"zk-code-arena-server/pkg/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const aiSessionCollection = "ai_sessions"

type MongoSessionService struct {
	collection *mongo.Collection
}

func NewMongoSessionService(db *mongo.Database) *MongoSessionService {
	if db == nil {
		return nil
	}
	service := &MongoSessionService{collection: db.Collection(aiSessionCollection)}
	service.ensureIndexes(context.Background())
	return service
}

func (s *MongoSessionService) CreateIfNeeded(ctx context.Context, req *aiapi.RuntimeRequest) error {
	return nil
}

func (s *MongoSessionService) SaveQuestion(ctx context.Context, req *aiapi.RuntimeRequest) error {
	if s == nil || s.collection == nil || req == nil {
		return nil
	}
	now := time.Now()
	_, err := s.collection.InsertOne(ctx, &models.AISession{
		SessionID: req.SessionID,
		RoundID:   req.RoundID,
		UserID:    req.UserID,
		AgentType: req.TaskType,
		Question:  req.Message,
		Status:    "running",
		CreatedAt: now,
		UpdatedAt: now,
	})
	return err
}

func (s *MongoSessionService) SaveResult(ctx context.Context, input *SaveResultInput) error {
	if s == nil || s.collection == nil || input == nil {
		return nil
	}
	update := bson.M{"$set": bson.M{
		"answer":            input.Answer,
		"thinking":          input.Thinking,
		"reference":         input.Reference,
		"recommend":         input.Recommend,
		"tools":             input.Tools,
		"first_response_ms": input.FirstResponseMS,
		"total_response_ms": input.TotalResponseMS,
		"status":            input.Status,
		"error_message":     input.ErrorMessage,
		"updated_at":        time.Now(),
	}}

	filter := bson.M{"session_id": input.SessionID, "round_id": input.RoundID}
	if input.UserID != "" {
		filter["user_id"] = input.UserID
	}

	opts := options.FindOneAndUpdate().SetReturnDocument(options.After).SetUpsert(true)
	return s.collection.FindOneAndUpdate(ctx, filter, update, opts).Err()
}

func (s *MongoSessionService) LoadRecent(ctx context.Context, sessionID, userID string, limit int) ([]*SessionSummary, error) {
	if s == nil || s.collection == nil {
		return nil, nil
	}
	filter := bson.M{"session_id": sessionID, "status": bson.M{"$ne": "running"}}
	if userID != "" {
		filter["user_id"] = userID
	}
	if limit <= 0 {
		limit = 10
	}
	cursor, err := s.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	records := make([]*models.AISession, 0, limit)
	if err := cursor.All(ctx, &records); err != nil {
		return nil, err
	}

	result := make([]*SessionSummary, 0, len(records))
	for i := len(records) - 1; i >= 0; i-- {
		record := records[i]
		if record == nil {
			continue
		}
		result = append(result, &SessionSummary{
			SessionID: record.SessionID,
			RoundID:   record.RoundID,
			UserID:    record.UserID,
			TaskType:  record.AgentType,
			Question:  record.Question,
			Answer:    record.Answer,
			CreatedAt: record.CreatedAt,
		})
	}
	return result, nil
}

func (s *MongoSessionService) ensureIndexes(ctx context.Context) {
	if s == nil || s.collection == nil {
		return
	}
	_, _ = s.collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "round_id", Value: 1}}},
		{Keys: bson.D{{Key: "session_id", Value: 1}, {Key: "status", Value: 1}}},
	})
}
