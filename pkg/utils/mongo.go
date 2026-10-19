/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: mongo.go
@Description: MongoDB 连接管理
*/

package utils

import (
	"context"
	"fmt"
	"time"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"zk-code-arena-server/conf"
)

var (
	MongoClient *mongo.Client
	MongoDB     *mongo.Database
)

// InitMongoDB 初始化 MongoDB 连接
func InitMongoDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 设置客户端选项
	clientOptions := options.Client().ApplyURI(conf.Config.Mongo.Uri)
	
	// 连接 MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("连接 MongoDB 失败: %v", err)
	}

	// 测试连接
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return fmt.Errorf("MongoDB 连接测试失败: %v", err)
	}

	MongoClient = client
	MongoDB = client.Database(conf.Config.Mongo.DbName)
	
	return nil
}

// CloseMongoDB 关闭 MongoDB 连接
func CloseMongoDB(ctx context.Context) error {
	if MongoClient != nil {
		return MongoClient.Disconnect(ctx)
	}
	return nil
}

// GetCollection 获取集合
func GetCollection(name string) *mongo.Collection {
	if MongoDB == nil {
		return nil
	}
	return MongoDB.Collection(name)
}
