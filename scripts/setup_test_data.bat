@echo off
chcp 65001 >nul

REM 测试数据初始化脚本 (Windows)
REM Author: omenkk7
REM Date: 2024/10/26

echo 🚀 开始初始化测试数据...

REM 检查 Docker 是否运行
docker ps >nul 2>&1
if %errorlevel% neq 0 (
    echo ❌ Docker 未运行或无法访问
    echo 请确保 Docker Desktop 已启动
    pause
    exit /b 1
)

REM 检查 MongoDB 容器是否运行
docker ps | findstr mongo >nul
if %errorlevel% neq 0 (
    echo ❌ MongoDB 容器未运行，请先启动 Docker Compose
    echo 运行命令: docker-compose up -d
    pause
    exit /b 1
)

REM 获取 MongoDB 容器名称
for /f "tokens=*" %%i in ('docker ps --format "{{.Names}}" ^| findstr mongo') do set MONGO_CONTAINER=%%i

if "%MONGO_CONTAINER%"=="" (
    echo ❌ 找不到 MongoDB 容器
    pause
    exit /b 1
)

echo 📦 找到 MongoDB 容器: %MONGO_CONTAINER%

REM 检查数据库是否已有数据
for /f %%i in ('docker exec %MONGO_CONTAINER% mongo zk_code_arena --quiet --eval "db.users.count()"') do set USER_COUNT=%%i

if %USER_COUNT% gtr 1 (
    echo ⚠️  数据库中已存在 %USER_COUNT% 个用户
    set /p "REPLY=是否要清空现有数据并重新生成？(y/N): "
    if /i "%REPLY%"=="y" (
        echo 🗑️  清空现有数据...
        docker exec %MONGO_CONTAINER% mongo zk_code_arena --eval "db.users.drop();db.problems.drop();db.test_cases.drop();db.submits.drop();db.courses.drop();db.clazzes.drop();db.tasks.drop();db.relations_users.drop();print('数据清空完成');"
    ) else (
        echo ❌ 操作已取消
        pause
        exit /b 0
    )
)

REM 复制脚本到容器
echo 📋 复制测试数据生成脚本到容器...
docker cp scripts/generate_test_data.js %MONGO_CONTAINER%:/tmp/generate_test_data.js

REM 执行测试数据生成脚本
echo 📝 执行测试数据生成脚本...
docker exec %MONGO_CONTAINER% mongo zk_code_arena /tmp/generate_test_data.js

REM 验证数据生成结果
echo.
echo ✅ 验证数据生成结果:
docker exec %MONGO_CONTAINER% mongo zk_code_arena --quiet --eval "print('👤 用户数量: ' + db.users.count());print('📚 题目数量: ' + db.problems.count());print('🧪 测试用例: ' + db.test_cases.count());print('💻 提交记录: ' + db.submits.count());print('📖 课程数量: ' + db.courses.count());print('👥 班级数量: ' + db.clazzes.count());print('📋 任务数量: ' + db.tasks.count());"

echo.
echo 🎉 测试数据初始化完成！
echo.
echo 🔑 默认登录账号:
echo 管理员: admin / admin123
echo 教师1: teacher1 / admin123
echo 教师2: teacher2 / admin123
echo 学生: student1-student20 / admin123
echo.
echo 🌐 访问地址:
echo - API文档 (Scalar): http://localhost:8080/docs
echo - API文档 (Swagger): http://localhost:8080/swagger/index.html
echo - 应用首页: http://localhost:8080

pause
