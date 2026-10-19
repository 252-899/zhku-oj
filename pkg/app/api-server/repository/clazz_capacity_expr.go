/*
@Author: omenkk7
@Date: 2026/03/22
@Description: 班级学生名额相关的 MongoDB 查询片段（以 member_ids 人数为准，不含教师）
*/

package repository

import "go.mongodb.org/mongo-driver/bson"

// ClazzStudentSeatAvailableExpr 用于 UpdateOne 过滤：当前学生数（member_ids）< max_members。
func ClazzStudentSeatAvailableExpr() bson.M {
	return bson.M{"$expr": bson.M{"$lt": []interface{}{
		bson.M{"$size": bson.M{"$ifNull": []interface{}{"$member_ids", bson.A{}}}},
		"$max_members",
	}}}
}
