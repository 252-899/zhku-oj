/**
 * 数据库关联关系只读审计脚本
 * 用途：检测因功能迭代产生的孤立数据（外键指向不存在的主数据）
 * 约定：不读取、不修改 problems / test_cases（题库数据）
 *
 * 使用: mongosh "mongodb://localhost:27017" --quiet --file scripts/audit_db_relations.js
 * 或:   mongosh "mongodb://localhost:27017/zk_code_arena" --quiet --file scripts/audit_db_relations.js
 */

const dbName = 'zk_code_arena';
const db = db.getSiblingDB(dbName);

function printSection(title) {
  print('\n' + '='.repeat(60));
  print(title);
  print('='.repeat(60));
}

function runCheck(collectionName, label, pipeline) {
  const coll = db.getCollection(collectionName);
  const countResult = coll.aggregate([...pipeline, { $count: 'n' }]).toArray();
  const n = (countResult[0] && countResult[0].n) || 0;
  print('  数量: ' + n);
  if (n > 0) {
    const samples = coll.aggregate([...pipeline, { $limit: 5 }, { $project: { _id: 1 } }]).toArray();
    print('  示例 _id: ' + samples.map((d) => d._id).join(', '));
  }
  return n;
}

let totalIssues = 0;

printSection('1. courses：created_by 不在 users');
totalIssues += runCheck('courses', 'created_by 无效', [
  { $lookup: { from: 'users', localField: 'created_by', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
]);

printSection('2. courses：teacher_ids 中存在不在 users 的 ID');
totalIssues += runCheck('courses', 'teacher_ids 无效', [
  { $unwind: { path: '$teacher_ids', preserveNullAndEmptyArrays: true } },
  { $match: { teacher_ids: { $ne: null } } },
  { $lookup: { from: 'users', localField: 'teacher_ids', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
  { $group: { _id: '$_id' } },
]);

printSection('3. clazzes：course_id 不在 courses');
totalIssues += runCheck('clazzes', 'course_id 无效', [
  { $lookup: { from: 'courses', localField: 'course_id', foreignField: '_id', as: 'c' } },
  { $match: { c: { $size: 0 } } },
]);

printSection('4. clazzes：member_ids 中存在不在 users 的 ID');
totalIssues += runCheck('clazzes', 'member_ids 无效', [
  { $unwind: { path: '$member_ids', preserveNullAndEmptyArrays: true } },
  { $match: { member_ids: { $ne: null } } },
  { $lookup: { from: 'users', localField: 'member_ids', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
  { $group: { _id: '$_id' } },
]);

printSection('5. clazzes：teacher_ids 中存在不在 users 的 ID');
totalIssues += runCheck('clazzes', 'teacher_ids 无效', [
  { $unwind: { path: '$teacher_ids', preserveNullAndEmptyArrays: true } },
  { $match: { teacher_ids: { $ne: null } } },
  { $lookup: { from: 'users', localField: 'teacher_ids', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
  { $group: { _id: '$_id' } },
]);

printSection('6. clazzes：major_class_ids 中存在不在 clazzes 的 ID');
totalIssues += runCheck('clazzes', 'major_class_ids 无效', [
  { $unwind: { path: '$major_class_ids', preserveNullAndEmptyArrays: true } },
  { $match: { major_class_ids: { $ne: null } } },
  { $lookup: { from: 'clazzes', localField: 'major_class_ids', foreignField: '_id', as: 'c' } },
  { $match: { c: { $size: 0 } } },
  { $group: { _id: '$_id' } },
]);

printSection('7. tasks：course_id 不在 courses');
totalIssues += runCheck('tasks', 'course_id 无效', [
  { $lookup: { from: 'courses', localField: 'course_id', foreignField: '_id', as: 'c' } },
  { $match: { c: { $size: 0 } } },
]);

printSection('8. student_classes：student_id 不在 users');
totalIssues += runCheck('student_classes', 'student_id 无效', [
  { $lookup: { from: 'users', localField: 'student_id', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
]);

printSection('9. student_classes：class_id 不在 clazzes');
totalIssues += runCheck('student_classes', 'class_id 无效', [
  { $lookup: { from: 'clazzes', localField: 'class_id', foreignField: '_id', as: 'c' } },
  { $match: { c: { $size: 0 } } },
]);

printSection('10. student_classes：course_id 不在 courses');
totalIssues += runCheck('student_classes', 'course_id 无效', [
  { $lookup: { from: 'courses', localField: 'course_id', foreignField: '_id', as: 'c' } },
  { $match: { c: { $size: 0 } } },
]);

printSection('11. task_clazz_relations：task_id 不在 tasks');
totalIssues += runCheck('task_clazz_relations', 'task_id 无效', [
  { $lookup: { from: 'tasks', localField: 'task_id', foreignField: '_id', as: 't' } },
  { $match: { t: { $size: 0 } } },
]);

printSection('12. task_clazz_relations：clazz_id 不在 clazzes');
totalIssues += runCheck('task_clazz_relations', 'clazz_id 无效', [
  { $lookup: { from: 'clazzes', localField: 'clazz_id', foreignField: '_id', as: 'c' } },
  { $match: { c: { $size: 0 } } },
]);

printSection('13. task_clazz_relations：course_id 不在 courses');
totalIssues += runCheck('task_clazz_relations', 'course_id 无效', [
  { $lookup: { from: 'courses', localField: 'course_id', foreignField: '_id', as: 'c' } },
  { $match: { c: { $size: 0 } } },
]);

printSection('14. user_task：task_id 不在 tasks');
totalIssues += runCheck('user_task', 'task_id 无效', [
  { $lookup: { from: 'tasks', localField: 'task_id', foreignField: '_id', as: 't' } },
  { $match: { t: { $size: 0 } } },
]);

printSection('15. user_task：user_id 不在 users');
totalIssues += runCheck('user_task', 'user_id 无效', [
  { $lookup: { from: 'users', localField: 'user_id', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
]);

printSection('16. user_relation_question：task_id 不在 tasks');
totalIssues += runCheck('user_relation_question', 'task_id 无效', [
  { $lookup: { from: 'tasks', localField: 'task_id', foreignField: '_id', as: 't' } },
  { $match: { t: { $size: 0 } } },
]);

printSection('17. user_relation_question：user_id 不在 users');
totalIssues += runCheck('user_relation_question', 'user_id 无效', [
  { $lookup: { from: 'users', localField: 'user_id', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
]);

printSection('18. relations_users：task_id 不在 tasks');
totalIssues += runCheck('relations_users', 'task_id 无效', [
  { $lookup: { from: 'tasks', localField: 'task_id', foreignField: '_id', as: 't' } },
  { $match: { t: { $size: 0 } } },
]);

printSection('19. relations_users：user_id 不在 users');
totalIssues += runCheck('relations_users', 'user_id 无效', [
  { $lookup: { from: 'users', localField: 'user_id', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
]);

printSection('20. submits：user_id 不在 users');
totalIssues += runCheck('submits', 'user_id 无效', [
  { $lookup: { from: 'users', localField: 'user_id', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
]);

printSection('21. user_favorite_problems：user_id 不在 users');
totalIssues += runCheck('user_favorite_problems', 'user_id 无效', [
  { $lookup: { from: 'users', localField: 'user_id', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
]);

printSection('22. tags：created_by 不在 users（存在 created_by 时）');
totalIssues += runCheck('tags', 'created_by 无效', [
  { $match: { created_by: { $exists: true, $ne: null } } },
  { $lookup: { from: 'users', localField: 'created_by', foreignField: '_id', as: 'u' } },
  { $match: { u: { $size: 0 } } },
]);

printSection('审计结束');
print('  检测项合计（含重复文档多类问题）: ' + totalIssues);
print('  未对 problems / test_cases 做任何读取或修改。');
print('  详细清理建议见 docs/DATA_AUDIT_GUIDE.md');
print('');
