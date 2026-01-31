#!/bin/bash

# NanoLog 服务构建与运行脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 颜色
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # 无颜色

echo -e "${GREEN}=== NanoLog 服务端 ===${NC}"

# 解析参数
ACTION="${1:-run}"
ARGS="${@:2}"

# 如果第一个参数是 flag，则默认为 'run' 并使用所有参数
if [[ "$ACTION" == -* ]]; then
    ARGS="$@"
    ACTION="run"
fi

case "$ACTION" in
    build)
        echo -e "${YELLOW}正在构建...${NC}"
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        echo -e "${GREEN}构建完成: bin/nanolog${NC}"
        ;;
    run)
        echo -e "${YELLOW}正在以开发模式运行...${NC}"
        CGO_ENABLED=0 go run cmd/nanolog/main.go $ARGS
        ;;
    start)
        echo -e "${YELLOW}正在构建并启动...${NC}"
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog $ARGS
        ;;
    standalone)
        echo -e "${YELLOW}正在启动单机模式 (全功能)...${NC}"
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog --role=standalone $ARGS
        ;;
    console)
        echo -e "${YELLOW}正在启动控制台模式 (管理与查询聚合)...${NC}"
        if [[ -z "$DATA_NODES" ]]; then
            echo -e "${YELLOW}提示: 请设置 DATA_NODES 环境变量或通过 --data-nodes 指定存储节点。${NC}"
        fi
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog --role=console $ARGS
        ;;
    ingester)
        echo -e "${YELLOW}正在启动存储节点模式 (存储与本地查询)...${NC}"
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog --role=ingester $ARGS
        ;;
    test)
        echo -e "${YELLOW}正在运行测试...${NC}"
        go test ./... -v
        ;;
    tidy)
        echo -e "${YELLOW}正在整理模块...${NC}"
        go mod tidy
        echo -e "${GREEN}完成${NC}"
        ;;
    reset-password)
        USER_NAME="${2}"
        NEW_PASS="${3}"
        if [[ -z "$USER_NAME" || -z "$NEW_PASS" ]]; then
            echo -e "${YELLOW}用法: $0 reset-password [用户名] [新密码]${NC}"
            exit 1
        fi
        echo -e "${YELLOW}正在为用户 $USER_NAME 重置密码...${NC}"
        CGO_ENABLED=0 go build -o bin/nanolog cmd/nanolog/main.go
        ./bin/nanolog --reset-password-user="$USER_NAME" --reset-password-val="$NEW_PASS" $ARGS
        ;;
    *)
        echo "用法: $0 {build|run|start|standalone|console|ingester|test|tidy|reset-password} [选项]"
        echo ""
        echo "命令:"
        echo "  build      - 编译到 bin/nanolog"
        echo "  run        - 以开发模式运行 (默认)"
        echo "  start      - 构建并运行，附带自定义参数"
        echo "  standalone - 以完整单机模式启动"
        echo "  console    - 以控制台节点启动 (需要 --data-nodes)"
        echo "  ingester   - 以存储节点启动"
        echo "  test       - 运行所有测试"
        echo "  tidy       - 运行 go mod tidy"
        echo "  reset-password - 重置指定用户的密码"
        echo ""
        echo "示例:"
        echo "  $0 standalone --port 8080"
        echo "  $0 console --data-nodes=http://node1:8081,http://node2:8081"
        echo "  $0 ingester --port 8081"
        echo "  $0 reset-password admin newpassword123"
        exit 1
        ;;
esac
