#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
@Author: ZKCodeArena Test Team
@Date: 2025-10-27
@Description: ZK Code Arena API 自动化测试脚本
"""

import requests
import json
import time
from datetime import datetime
from typing import Dict, Optional, List, Tuple
from colorama import init, Fore, Style

# 初始化 colorama（Windows 彩色输出支持）
init(autoreset=True)

# ===================== 配置 =====================
BASE_URL = "http://localhost:8080/api/v1"

# 测试账号配置
ADMIN_ACCOUNT = {
    "student_id": "1231234",
    "password": "123456",
    "role": "admin"
}

TEACHER_ACCOUNT = {
    "student_id": "1231235", 
    "password": "123456",
    "role": "teacher"
}

# 测试结果统计
test_results = {
    "total": 0,
    "passed": 0,
    "failed": 0,
    "skipped": 0,
    "errors": []
}

# ===================== 工具函数 =====================

def print_header(text: str):
    """打印标题"""
    print(f"\n{Fore.CYAN}{'=' * 80}")
    print(f"{Fore.CYAN}{text.center(80)}")
    print(f"{Fore.CYAN}{'=' * 80}\n")

def print_success(text: str):
    """打印成功信息"""
    print(f"{Fore.GREEN}✓ {text}")

def print_error(text: str):
    """打印错误信息"""
    print(f"{Fore.RED}✗ {text}")

def print_warning(text: str):
    """打印警告信息"""
    print(f"{Fore.YELLOW}⚠ {text}")

def print_info(text: str):
    """打印信息"""
    print(f"{Fore.BLUE}ℹ {text}")

class APITester:
    """API 测试类"""
    
    def __init__(self, base_url: str):
        self.base_url = base_url
        self.admin_token = None
        self.teacher_token = None
        self.created_resources = {
            "problems": [],
            "testcases": [],
            "courses": [],
            "classes": []
        }
    
    def login(self, student_id: str, password: str) -> Optional[str]:
        """用户登录"""
        url = f"{self.base_url}/user/login"
        data = {
            "student_id": student_id,
            "password": password
        }
        
        try:
            response = requests.post(url, json=data)
            if response.status_code == 200:
                result = response.json()
                if result.get("code") == 200:
                    token = result.get("data", {}).get("token")
                    return token
            return None
        except Exception as e:
            print_error(f"登录失败: {e}")
            return None
    
    def make_request(self, method: str, endpoint: str, token: Optional[str] = None, 
                     data: Optional[Dict] = None, params: Optional[Dict] = None) -> Tuple[bool, Optional[Dict], int]:
        """发送 HTTP 请求"""
        url = f"{self.base_url}{endpoint}"
        headers = {}
        
        if token:
            headers["Authorization"] = f"Bearer {token}"
        
        try:
            if method.upper() == "GET":
                response = requests.get(url, headers=headers, params=params)
            elif method.upper() == "POST":
                response = requests.post(url, headers=headers, json=data)
            elif method.upper() == "PUT":
                response = requests.put(url, headers=headers, json=data)
            elif method.upper() == "DELETE":
                response = requests.delete(url, headers=headers)
            else:
                return False, None, 0
            
            result = response.json() if response.text else {}
            return response.status_code < 400, result, response.status_code
        except Exception as e:
            return False, {"error": str(e)}, 0
    
    def test_api(self, name: str, method: str, endpoint: str, token: Optional[str] = None,
                 data: Optional[Dict] = None, params: Optional[Dict] = None,
                 expected_code: int = 200) -> bool:
        """测试单个 API"""
        global test_results
        test_results["total"] += 1
        
        print(f"\n{Fore.CYAN}测试: {name}")
        print(f"  请求: {method} {endpoint}")
        
        success, result, status_code = self.make_request(method, endpoint, token, data, params)
        
        if status_code == expected_code:
            test_results["passed"] += 1
            print_success(f"通过 - 状态码: {status_code}")
            if result and result.get("data"):
                print_info(f"响应数据: {json.dumps(result.get('data'), ensure_ascii=False, indent=2)[:200]}...")
            return True
        else:
            test_results["failed"] += 1
            error_msg = f"{name} - 期望 {expected_code}, 实际 {status_code}"
            test_results["errors"].append(error_msg)
            print_error(f"失败 - 期望: {expected_code}, 实际: {status_code}")
            if result:
                print_error(f"错误信息: {json.dumps(result, ensure_ascii=False, indent=2)[:200]}")
            return False

# ===================== 测试用例 =====================

def test_user_apis(tester: APITester):
    """测试用户相关接口"""
    print_header("用户模块测试")
    
    # 1. 管理员登录
    print_info("测试管理员登录...")
    tester.admin_token = tester.login(ADMIN_ACCOUNT["student_id"], ADMIN_ACCOUNT["password"])
    if tester.admin_token:
        print_success(f"管理员登录成功，Token: {tester.admin_token[:20]}...")
    else:
        print_error("管理员登录失败")
        return
    
    # 2. 教师登录
    print_info("测试教师登录...")
    tester.teacher_token = tester.login(TEACHER_ACCOUNT["student_id"], TEACHER_ACCOUNT["password"])
    if tester.teacher_token:
        print_success(f"教师登录成功，Token: {tester.teacher_token[:20]}...")
    else:
        print_error("教师登录失败")
    
    # 3. 获取当前用户信息
    tester.test_api(
        "获取管理员信息",
        "GET",
        "/user/profile",
        token=tester.admin_token
    )
    
    # 4. 获取用户列表（管理员）
    tester.test_api(
        "获取用户列表（管理员）",
        "GET",
        "/user",
        token=tester.admin_token,
        params={"page": 1, "page_size": 10}
    )
    
    # 5. 教师尝试获取用户列表（应该失败）
    tester.test_api(
        "教师获取用户列表（应该403）",
        "GET",
        "/user",
        token=tester.teacher_token,
        expected_code=403
    )

def test_problem_apis(tester: APITester):
    """测试题目相关接口"""
    print_header("题目模块测试")
    
    # 1. 创建题目（管理员）
    problem_data = {
        "title": f"测试题目-{int(time.time())}",
        "description": "这是一个自动化测试创建的题目，描述内容需要至少10个字符。",
        "input": "第一行一个整数 n",
        "output": "输出一个整数",
        "sample_input": "5",
        "sample_output": "5",
        "difficulty": "easy",
        "time_limit": 1000,
        "memory_limit": 256,
        "tags": ["测试", "自动化"],
        "status": "published",
        "is_public": True
    }
    
    success, result, _ = tester.make_request("POST", "/problem", tester.admin_token, problem_data)
    if success and result.get("data"):
        problem_id = result["data"].get("id")
        tester.created_resources["problems"].append(problem_id)
        print_success(f"创建题目成功，ID: {problem_id}")
    else:
        print_error("创建题目失败")
        return
    
    # 2. 获取题目列表
    tester.test_api(
        "获取题目列表（用户端）",
        "GET",
        "/problem",
        params={"page": 1, "page_size": 10}
    )
    
    # 3. 获取题目详情
    if tester.created_resources["problems"]:
        problem_id = tester.created_resources["problems"][0]
        tester.test_api(
            "获取题目详情",
            "GET",
            f"/problem/{problem_id}"
        )
    
    # 4. 更新题目（测试之前的 500 错误）
    if tester.created_resources["problems"]:
        problem_id = tester.created_resources["problems"][0]
        update_data = {
            "title": f"更新后的题目-{int(time.time())}",
            "description": "这是更新后的描述内容，需要至少10个字符才能通过验证。",
            "difficulty": "medium"
        }
        tester.test_api(
            "更新题目（之前会500错误）",
            "PUT",
            f"/problem/{problem_id}",
            token=tester.admin_token,
            data=update_data
        )
    
    # 5. 搜索题目
    tester.test_api(
        "搜索题目",
        "GET",
        "/problem/search",
        params={"keyword": "测试", "page": 1, "page_size": 10}
    )
    
    # 6. 获取每日推荐题目
    tester.test_api(
        "获取每日推荐题目",
        "GET",
        "/daily-problem"
    )
    
    # 7. 获取题目列表（管理员端）
    tester.test_api(
        "获取题目列表（管理员端）",
        "GET",
        "/admin/problems",
        token=tester.admin_token,
        params={"page": 1, "page_size": 10}
    )

def test_testcase_apis(tester: APITester):
    """测试测试用例相关接口"""
    print_header("测试用例模块测试")
    
    if not tester.created_resources["problems"]:
        print_warning("跳过测试用例测试：没有可用的题目")
        return
    
    problem_id = tester.created_resources["problems"][0]
    
    # 1. 创建测试用例
    testcase_data = {
        "problem_id": problem_id,
        "input": "5",
        "output": "5",
        "is_sample": True,
        "score": 10
    }
    
    success, result, _ = tester.make_request("POST", "/testcase", tester.admin_token, testcase_data)
    if success and result.get("data"):
        testcase_id = result["data"].get("id")
        tester.created_resources["testcases"].append(testcase_id)
        print_success(f"创建测试用例成功，ID: {testcase_id}")
        
        # 验证是否包含 updated_at 字段
        if "updated_at" in result["data"]:
            print_success("✓ 测试用例包含 updated_at 字段")
        else:
            print_error("✗ 测试用例缺少 updated_at 字段")
    else:
        print_error("创建测试用例失败")
        return
    
    # 2. 获取题目的测试用例列表
    tester.test_api(
        "获取题目的测试用例列表",
        "GET",
        f"/testcase/problem/{problem_id}",
        token=tester.admin_token
    )
    
    # 3. 获取测试用例详情
    if tester.created_resources["testcases"]:
        testcase_id = tester.created_resources["testcases"][0]
        tester.test_api(
            "获取测试用例详情",
            "GET",
            f"/testcase/{testcase_id}",
            token=tester.admin_token
        )
    
    # 4. 更新测试用例
    if tester.created_resources["testcases"]:
        testcase_id = tester.created_resources["testcases"][0]
        update_data = {
            "input": "10",
            "output": "10",
            "is_sample": True
        }
        tester.test_api(
            "更新测试用例",
            "PUT",
            f"/testcase/{testcase_id}",
            token=tester.admin_token,
            data=update_data
        )
    
    # 5. 批量创建测试用例
    batch_data = {
        "problem_id": problem_id,
        "test_cases": [
            {"input": "1", "output": "1", "is_sample": False, "score": 10},
            {"input": "2", "output": "2", "is_sample": False, "score": 10},
            {"input": "3", "output": "3", "is_sample": False, "score": 10}
        ]
    }
    tester.test_api(
        "批量创建测试用例",
        "POST",
        "/testcase/batch",
        token=tester.admin_token,
        data=batch_data
    )

def test_statistics_apis(tester: APITester):
    """测试统计相关接口"""
    print_header("统计模块测试")
    
    # 1. 获取当前用户统计
    tester.test_api(
        "获取当前用户统计",
        "GET",
        "/statistics/user",
        token=tester.admin_token
    )
    
    # 2. 获取系统统计（管理员）
    tester.test_api(
        "获取系统统计（管理员）",
        "GET",
        "/statistics/system",
        token=tester.admin_token
    )
    
    # 3. 获取题目难度统计
    tester.test_api(
        "获取题目难度统计",
        "GET",
        "/problems/difficulty-stats"
    )

def test_course_apis(tester: APITester):
    """测试课程相关接口"""
    print_header("课程模块测试")
    
    # 1. 创建课程（教师）
    course_data = {
        "name": f"自动化测试课程-{int(time.time())}",
        "description": "这是一个自动化测试创建的课程"
    }
    
    success, result, _ = tester.make_request("POST", "/course", tester.teacher_token, course_data)
    if success and result.get("data"):
        course_id = result["data"]
        tester.created_resources["courses"].append(course_id)
        print_success(f"创建课程成功，ID: {course_id}")
    else:
        print_error("创建课程失败")
        return
    
    # 2. 获取课程列表
    tester.test_api(
        "获取课程列表",
        "GET",
        "/course/list",
        token=tester.teacher_token
    )
    
    # 3. 获取课程详情
    if tester.created_resources["courses"]:
        course_id = tester.created_resources["courses"][0]
        tester.test_api(
            "获取课程详情",
            "GET",
            f"/course/{course_id}",
            token=tester.teacher_token
        )

def test_submit_apis(tester: APITester):
    """测试提交相关接口"""
    print_header("提交模块测试")
    
    # 1. 获取提交列表
    tester.test_api(
        "获取提交列表",
        "GET",
        "/submit",
        token=tester.admin_token,
        params={"page": 1, "page_size": 10}
    )
    
    # 2. 如果有题目，可以测试代码提交
    if tester.created_resources["problems"]:
        problem_id = tester.created_resources["problems"][0]
        submit_data = {
            "problem_id": problem_id,
            "language": "python",
            "code": "print(input())"
        }
        tester.test_api(
            "提交代码",
            "POST",
            "/submit",
            token=tester.admin_token,
            data=submit_data
        )

# ===================== 清理函数 =====================

def cleanup_resources(tester: APITester):
    """清理测试创建的资源"""
    print_header("清理测试资源")
    
    # 删除测试用例
    for testcase_id in tester.created_resources["testcases"]:
        success, _, _ = tester.make_request("DELETE", f"/testcase/{testcase_id}", tester.admin_token)
        if success:
            print_success(f"删除测试用例: {testcase_id}")
        else:
            print_warning(f"删除测试用例失败: {testcase_id}")
    
    # 删除题目
    for problem_id in tester.created_resources["problems"]:
        success, _, _ = tester.make_request("DELETE", f"/problem/{problem_id}", tester.admin_token)
        if success:
            print_success(f"删除题目: {problem_id}")
        else:
            print_warning(f"删除题目失败: {problem_id}")

# ===================== 主函数 =====================

def print_test_summary():
    """打印测试摘要"""
    print_header("测试结果摘要")
    
    total = test_results["total"]
    passed = test_results["passed"]
    failed = test_results["failed"]
    
    print(f"总测试数: {total}")
    print(f"{Fore.GREEN}通过: {passed}")
    print(f"{Fore.RED}失败: {failed}")
    
    if failed > 0:
        print(f"\n{Fore.RED}失败的测试:")
        for error in test_results["errors"]:
            print(f"  {Fore.RED}✗ {error}")
    
    # 计算通过率
    if total > 0:
        pass_rate = (passed / total) * 100
        print(f"\n通过率: {pass_rate:.2f}%")
        
        if pass_rate == 100:
            print(f"\n{Fore.GREEN}{'🎉 所有测试通过！'.center(80)}")
        elif pass_rate >= 80:
            print(f"\n{Fore.YELLOW}{'⚠ 大部分测试通过'.center(80)}")
        else:
            print(f"\n{Fore.RED}{'❌ 需要修复失败的测试'.center(80)}")

def main():
    """主函数"""
    print_header("ZK Code Arena API 自动化测试")
    print(f"测试时间: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"服务器地址: {BASE_URL}")
    
    # 创建测试器
    tester = APITester(BASE_URL)
    
    try:
        # 执行测试
        test_user_apis(tester)
        test_problem_apis(tester)
        test_testcase_apis(tester)
        test_statistics_apis(tester)
        test_course_apis(tester)
        test_submit_apis(tester)
        
        # 清理资源
        cleanup_resources(tester)
        
    except KeyboardInterrupt:
        print_warning("\n测试被用户中断")
    except Exception as e:
        print_error(f"测试过程中出现错误: {e}")
    finally:
        # 打印测试摘要
        print_test_summary()

if __name__ == "__main__":
    main()

