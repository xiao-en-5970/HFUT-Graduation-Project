#!/bin/bash

# 排查并结束占用 PostgreSQL (5432) 和 Redis (6379) 端口的进程

echo "=========================================="
echo "端口占用排查和清理脚本"
echo "=========================================="
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查并处理端口占用
check_and_kill_port() {
    local port=$1
    local service=$2
    
    echo "检查端口 $port ($service)..."
    
    # 检查端口占用情况
    if command -v lsof >/dev/null 2>&1; then
        # 使用 lsof (macOS/Linux)
        pid=$(lsof -ti :$port 2>/dev/null)
        if [ -n "$pid" ]; then
            echo -e "${YELLOW}⚠️  端口 $port 被进程占用:${NC}"
            lsof -i :$port
            echo ""
            echo "进程信息:"
            ps -p $pid -o pid,ppid,user,cmd
            echo ""
            read -p "是否结束这些进程? (y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                kill -9 $pid 2>/dev/null || true
                sleep 1
                if lsof -ti :$port >/dev/null 2>&1; then
                    echo -e "${RED}❌ 无法释放端口 $port${NC}"
                    return 1
                else
                    echo -e "${GREEN}✅ 端口 $port 已释放${NC}"
                fi
            else
                echo "跳过端口 $port"
            fi
        else
            echo -e "${GREEN}✅ 端口 $port 未被占用${NC}"
        fi
    elif command -v netstat >/dev/null 2>&1; then
        # 使用 netstat (Linux)
        pid=$(netstat -tlnp 2>/dev/null | grep ":$port " | awk '{print $7}' | cut -d'/' -f1 | head -1)
        if [ -n "$pid" ] && [ "$pid" != "-" ]; then
            echo -e "${YELLOW}⚠️  端口 $port 被进程占用:${NC}"
            netstat -tlnp | grep ":$port "
            echo ""
            echo "进程信息:"
            ps -p $pid -o pid,ppid,user,cmd 2>/dev/null || echo "无法获取进程信息"
            echo ""
            read -p "是否结束这些进程? (y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                kill -9 $pid 2>/dev/null || true
                sleep 1
                if netstat -tlnp 2>/dev/null | grep -q ":$port "; then
                    echo -e "${RED}❌ 无法释放端口 $port${NC}"
                    return 1
                else
                    echo -e "${GREEN}✅ 端口 $port 已释放${NC}"
                fi
            else
                echo "跳过端口 $port"
            fi
        else
            echo -e "${GREEN}✅ 端口 $port 未被占用${NC}"
        fi
    elif command -v ss >/dev/null 2>&1; then
        # 使用 ss (现代 Linux)
        pid=$(ss -tlnp 2>/dev/null | grep ":$port " | grep -oP 'pid=\K[0-9]+' | head -1)
        if [ -n "$pid" ]; then
            echo -e "${YELLOW}⚠️  端口 $port 被进程占用:${NC}"
            ss -tlnp | grep ":$port "
            echo ""
            echo "进程信息:"
            ps -p $pid -o pid,ppid,user,cmd 2>/dev/null || echo "无法获取进程信息"
            echo ""
            read -p "是否结束这些进程? (y/n): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                kill -9 $pid 2>/dev/null || true
                sleep 1
                if ss -tlnp 2>/dev/null | grep -q ":$port "; then
                    echo -e "${RED}❌ 无法释放端口 $port${NC}"
                    return 1
                else
                    echo -e "${GREEN}✅ 端口 $port 已释放${NC}"
                fi
            else
                echo "跳过端口 $port"
            fi
        else
            echo -e "${GREEN}✅ 端口 $port 未被占用${NC}"
        fi
    else
        echo -e "${RED}❌ 未找到可用的端口检查工具 (lsof/netstat/ss)${NC}"
        return 1
    fi
    echo ""
}

# 检查 PostgreSQL 端口 (5432)
check_and_kill_port 5432 "PostgreSQL"

# 检查 Redis 端口 (6379)
check_and_kill_port 6379 "Redis"

echo "=========================================="
echo "检查完成"
echo "=========================================="



