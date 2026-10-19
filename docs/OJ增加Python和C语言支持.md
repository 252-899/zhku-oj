# OJ 增加 Python 和 C 语言支持指南

本说明介绍如何在本 OJ 刷题平台中启用 **Python** 和 **C** 语言支持（项目已内置配置，主要需确保判题环境与前端一致）。

---

## 一、当前支持情况概览

| 层级         | C | C++ | Java | Python | Go |
|--------------|---|-----|------|--------|-----|
| 后端配置     | ✅ | ✅  | ✅   | ✅     | ✅  |
| 提交校验     | ✅ | ✅  | ✅   | ✅     | ✅  |
| 判题逻辑     | ✅ | ✅  | ✅   | ✅     | ✅  |
| **判题环境** | 需镜像支持 | 需镜像支持 | 需镜像支持 | 需镜像支持 | 需镜像支持 |
| 前端语言选项 | 需与后端一致 | 需与后端一致 | 需与后端一致 | 需与后端一致 | 需与后端一致 |

后端（`conf/config.yaml`、`pkg/models/submit.go`、提交接口）已支持 `c`、`cpp`、`java`、`python`、`go`。  
要真正能提交并判题 **Python** 和 **C**，需要完成两件事：

1. **判题 Docker 镜像**里安装对应编译器/解释器（见下文第二节）。
2. **前端**（若有）提供 C / Python 选项并传 `language: "c"` / `"python"`（见第三节）。

---

## 二、判题环境（Docker 镜像）

判题由 **go-judge** 在容器内执行命令，命令来自 `conf/config.yaml` 的 `Judge.Languages`。  
因此容器内必须存在对应可执行文件，例如：

- **C**：`/usr/bin/gcc`（config 中为 `main.c` → `main`）
- **Python**：`/usr/bin/python3`（config 中为 `main.py`，无需编译）

### 2.1 使用项目提供的多语言判题镜像（推荐）

在仓库中已提供 `deploy/judge/Dockerfile.multilang`，用于构建包含 **C / C++ / Java / Python / Go** 的判题镜像。

```bash
# 在项目根目录执行
docker build -f deploy/judge/Dockerfile.multilang -t zhku-oj-judge:multilang ./deploy/judge
```

修改 `docker-compose.yml` 或 `docker-compose.dev.yml` 中的 judge 服务镜像为该镜像，例如：

```yaml
judge:
  image: zhku-oj-judge:multilang   # 替换原来的 criyle/go-judge 或 zhku-oj-java:1.0
  container_name: zk-arena-judge
  # ... 其余配置不变
```

然后重启 judge 服务：

```bash
docker compose up -d judge
# 或
docker-compose -f docker-compose.dev.yml up -d judge
```

### 2.2 实验室/无代理服务器无法构建时（推荐：本机构建再导入）

若服务器无法访问外网（如 archive.ubuntu.com、GitHub 连接超时）且不能设置代理，**不要在服务器上构建**，改为在能联网的电脑上构建镜像，导出后传到服务器再加载。

**步骤简述：**

1. **在本机（能访问外网）执行：**
   ```bash
   cd <项目根目录>
   docker build -f deploy/judge/Dockerfile.multilang -t zhku-oj-judge:multilang ./deploy/judge
   docker save -o zhku-oj-judge-multilang.tar zhku-oj-judge:multilang
   ```
2. **将 `zhku-oj-judge-multilang.tar` 传到服务器**（如 `scp -i id_rsa zhku-oj-judge-multilang.tar admin@服务器IP:/home/admin/zhku_oj_projuct/`）。
3. **在服务器上执行：**
   ```bash
   cd /home/admin/zhku_oj_projuct
   docker load -i zhku-oj-judge-multilang.tar
   # 修改 docker-compose.dev.yml 中 judge 的 image 为 zhku-oj-judge:multilang
   docker compose -f docker-compose.dev.yml down && docker compose -f docker-compose.dev.yml up -d
   ```

更多细节（含“仅能访问国内源”时的 Dockerfile 用法）见 **deploy/judge/README.md**。

### 2.3 若使用官方 criyle/go-judge 镜像

官方镜像 `criyle/go-judge:latest` 通常只带基础系统，**不包含** gcc、python3、openjdk 等。  
要支持 C / Python，有两种方式：

- **方式 A**：改用本项目提供的多语言镜像（见 2.1），推荐。  
- **方式 B**：自己基于 Ubuntu/Debian 做镜像，安装 `gcc`、`g++`、`python3`、`openjdk-17-jdk` 等，再在容器内运行 go-judge（从 [go-judge Releases](https://github.com/criyle/go-judge/releases) 下载对应平台二进制即可）。

### 2.4 验证判题环境

进入 judge 容器检查是否存在 C 和 Python：

```bash
docker exec -it zk-arena-judge bash
# 或
docker exec -it zk-arena-judge-dev bash

# 检查
which gcc python3
gcc --version
python3 --version
```

若都能找到且版本正常，说明判题环境已具备 C 和 Python 支持。

### 2.5 一键测试脚本（推荐）

项目内提供了测试脚本，在**服务器上**（judge 已启动且映射 5050 端口）执行：

```bash
cd /home/admin/zhku_oj_projuct
git pull origin test/test-20260131-v1.1.0   # 拉取包含脚本的最新代码
bash scripts/test_judge_python_c.sh
```

脚本会依次：① 在判题容器内检查是否有 gcc、python3；② 测试 Python（运行 print(1+1)，期望输出 2）；③ 测试 C（编译并运行 printf("hello")）。全部通过会输出「全部通过：Python 和 C 判题环境可用」。若当前是仅 Java 镜像，前两步会失败，需先换成多语言镜像再测。

可选环境变量：`JUDGE_URL` 默认 `http://localhost:5050`；`JUDGE_CONTAINER` 默认 `zk-arena-judge-dev`。

---

## 三、配置文件说明（无需改即可支持 C/Python）

`conf/config.yaml` 中已包含 C 和 Python 的完整配置，**一般无需修改**。仅作说明：

### 3.1 C 语言（已配置）

- 源文件：`main.c`
- 编译：`gcc -std=c11 -O2 -Wall main.c -o main -lm`
- 运行：`./main`
- 时间/内存限制在 config 中已设置

### 3.2 Python（已配置）

- 源文件：`main.py`
- 无编译阶段（解释型）
- 运行：`/usr/bin/python3 -u main.py`
- 时间限制略大（如 2 秒），因解释型语言通常更慢

若你希望改用其他文件名或参数（例如 `solution.c`、`python3.11`），只需在 `conf/config.yaml` 的 `Judge.Languages.c` / `Judge.Languages.python` 下修改 `source_file` 和 `run.args` 等即可。

---

## 四、前端（若有）

若你有独立前端项目（Vue/React 等），需要保证：

1. **提交时传参**：`language` 字段使用后端支持的取值：`"c"`、`"cpp"`、`"java"`、`"python"`、`"go"`。
2. **语言列表**：下拉框或选项卡中包含「C」和「Python」，并分别对应 `c` 和 `python`。
3. **代码模板/占位符**（可选）：  
   - C：建议默认内容为 `main` 函数、`#include <stdio.h>` 等，且保存为 `main.c` 逻辑（后端会按 config 把用户代码当作 `main.c` 内容）。  
   - Python：同理，对应 `main.py`。

后端 API 已校验并支持上述语言，只要前端传 `language: "c"` 或 `"python"` 即可。

---

## 五、用户提交约定（与 config 一致）

- **C**：用户代码即“整份 C 源码”，判题端会以 `main.c` 形式交给 gcc 编译并运行。  
- **Python**：用户代码即“整份 Python 源码”，判题端会以 `main.py` 形式交给 `python3` 运行。  

题目描述或前端提示中可说明：C 使用 `main.c`（主程序），Python 使用 `main.py`（主程序），与当前配置一致即可。

---

## 六、小结：你要做的步骤

1. **判题镜像**：构建并改用多语言判题镜像（见 2.1），或自建包含 gcc、python3 的 go-judge 镜像。  
2. **重启 judge 服务**：使用新镜像后执行 `docker compose up -d judge`（或你的 compose 命令）。  
3. **前端**：若有前端，在语言选项中增加 C、Python，并传 `language: "c"` / `"python"`。  
4. **（可选）** 在题目或帮助中说明 C 使用 `main.c`、Python 使用 `main.py`。

完成后，用户即可选择 C 或 Python 提交代码并被正常判题。
