#!/bin/bash

# MinIO 部署脚本
# 用于在服务器上安装和配置 MinIO 对象存储服务

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 配置变量
MINIO_VERSION="RELEASE.2024-12-13T18-30-20Z"
MINIO_USER="minio"
MINIO_GROUP="minio"
MINIO_HOME="/opt/minio"
MINIO_DATA_DIR="/data/minio"
MINIO_CONFIG_DIR="/etc/minio"
MINIO_PORT="50001"
MINIO_CONSOLE_PORT="50002"
MINIO_ROOT_USER="minioadmin"
MINIO_ROOT_PASSWORD=""

# 生成随机密码（如果未设置）
if [ -z "$MINIO_ROOT_PASSWORD" ]; then
    MINIO_ROOT_PASSWORD=$(openssl rand -base64 32)
fi

echo -e "${GREEN}开始部署 MinIO...${NC}"

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}请使用 root 用户运行此脚本${NC}"
    exit 1
fi

# 1. 创建 MinIO 用户和组
echo -e "${YELLOW}[1/8] 创建 MinIO 用户和组...${NC}"
if ! id "$MINIO_USER" &>/dev/null; then
    groupadd -r $MINIO_GROUP
    useradd -r -g $MINIO_GROUP -d $MINIO_HOME -s /sbin/nologin $MINIO_USER
    echo -e "${GREEN}✓ 用户创建成功${NC}"
else
    echo -e "${YELLOW}✓ 用户已存在${NC}"
fi

# 2. 创建必要的目录
echo -e "${YELLOW}[2/8] 创建目录结构...${NC}"
mkdir -p $MINIO_HOME
mkdir -p $MINIO_DATA_DIR
mkdir -p $MINIO_CONFIG_DIR
chown -R $MINIO_USER:$MINIO_GROUP $MINIO_HOME
chown -R $MINIO_USER:$MINIO_GROUP $MINIO_DATA_DIR
chown -R $MINIO_USER:$MINIO_GROUP $MINIO_CONFIG_DIR
echo -e "${GREEN}✓ 目录创建成功${NC}"

# 3. 下载 MinIO 二进制文件
echo -e "${YELLOW}[3/8] 下载 MinIO 二进制文件...${NC}"
cd /tmp
MINIO_BINARY="minio.${MINIO_VERSION}"
if [ ! -f "$MINIO_BINARY" ]; then
    wget -q "https://dl.min.io/server/minio/release/linux-amd64/archive/${MINIO_BINARY}" -O $MINIO_BINARY
    chmod +x $MINIO_BINARY
    echo -e "${GREEN}✓ MinIO 下载成功${NC}"
else
    echo -e "${YELLOW}✓ MinIO 文件已存在${NC}"
fi

# 4. 安装 MinIO 二进制文件
echo -e "${YELLOW}[4/8] 安装 MinIO 二进制文件...${NC}"
cp $MINIO_BINARY /usr/local/bin/minio
chmod +x /usr/local/bin/minio
chown $MINIO_USER:$MINIO_GROUP /usr/local/bin/minio
echo -e "${GREEN}✓ MinIO 安装成功${NC}"

# 5. 创建环境变量文件
echo -e "${YELLOW}[5/8] 创建环境变量文件...${NC}"
cat > $MINIO_CONFIG_DIR/minio.env <<EOF
# MinIO 配置
MINIO_ROOT_USER=${MINIO_ROOT_USER}
MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}
MINIO_VOLUMES="${MINIO_DATA_DIR}"
MINIO_OPTS="--address :${MINIO_PORT} --console-address :${MINIO_CONSOLE_PORT}"
EOF
chown $MINIO_USER:$MINIO_GROUP $MINIO_CONFIG_DIR/minio.env
chmod 600 $MINIO_CONFIG_DIR/minio.env
echo -e "${GREEN}✓ 环境变量文件创建成功${NC}"
echo -e "${YELLOW}  Root User: ${MINIO_ROOT_USER}${NC}"
echo -e "${YELLOW}  Root Password: ${MINIO_ROOT_PASSWORD}${NC}"
echo -e "${RED}  请妥善保存密码！${NC}"

# 6. 检测是否支持 systemd
SYSTEMD_AVAILABLE=false
if command -v systemctl &> /dev/null && systemctl --version &> /dev/null; then
    # 检查是否真的可以连接 systemd
    if systemctl list-units &> /dev/null 2>&1; then
        SYSTEMD_AVAILABLE=true
    fi
fi

# 7. 创建启动脚本
echo -e "${YELLOW}[7/8] 创建启动脚本...${NC}"
cat > ${MINIO_HOME}/start-minio.sh <<EOF
#!/bin/bash
# 加载环境变量
set -a
. ${MINIO_CONFIG_DIR}/minio.env
set +a
cd ${MINIO_HOME}
exec /usr/local/bin/minio server \$MINIO_OPTS \$MINIO_VOLUMES
EOF
chmod +x ${MINIO_HOME}/start-minio.sh
chown ${MINIO_USER}:${MINIO_GROUP} ${MINIO_HOME}/start-minio.sh
echo -e "${GREEN}✓ 启动脚本创建成功${NC}"

# 8. 启动 MinIO 服务
echo -e "${YELLOW}[8/8] 启动 MinIO 服务...${NC}"

if [ "$SYSTEMD_AVAILABLE" = true ]; then
    # 使用 systemd
    # 使用变量替换，但避免 heredoc 中的 bash 解析问题
    {
        cat <<EOF
[Unit]
Description=MinIO Object Storage
Documentation=https://docs.min.io
Wants=network-online.target
After=network-online.target
AssertFileIsExecutable=/usr/local/bin/minio

[Service]
WorkingDirectory=${MINIO_HOME}
User=${MINIO_USER}
Group=${MINIO_GROUP}
EnvironmentFile=${MINIO_CONFIG_DIR}/minio.env
ExecStartPre=/bin/bash -c 'if [ -z "\${MINIO_VOLUMES}" ]; then echo "Variable MINIO_VOLUMES not set in ${MINIO_CONFIG_DIR}/minio.env"; exit 1; fi'
ExecStart=/usr/local/bin/minio server \${MINIO_OPTS} \${MINIO_VOLUMES}

Restart=always
LimitNOFILE=65536
TasksMax=infinity
SendSIGKILL=no

[Install]
WantedBy=multi-user.target
EOF
    } > /etc/systemd/system/minio.service
    systemctl daemon-reload
    systemctl enable minio
    systemctl start minio
    sleep 3
    
    if systemctl is-active --quiet minio; then
        echo -e "${GREEN}✓ MinIO 服务启动成功（使用 systemd）${NC}"
    else
        echo -e "${RED}✗ MinIO 服务启动失败，请检查日志: journalctl -u minio${NC}"
        exit 1
    fi
else
    # 使用 nohup 后台运行
    echo -e "${YELLOW}  检测到系统不支持 systemd，使用 nohup 方式启动...${NC}"
    
    # 停止可能正在运行的 MinIO 进程
    pkill -u ${MINIO_USER} -f "minio server" 2>/dev/null || true
    sleep 1
    
    # 创建日志目录
    mkdir -p ${MINIO_HOME}/logs
    chown -R ${MINIO_USER}:${MINIO_GROUP} ${MINIO_HOME}/logs
    
    # 使用 nohup 启动 MinIO（使用 runuser 或 su 指定 shell）
    if command -v runuser &> /dev/null; then
        runuser -u ${MINIO_USER} -- bash -c "cd ${MINIO_HOME} && nohup ${MINIO_HOME}/start-minio.sh > ${MINIO_HOME}/logs/minio.log 2>&1 &"
    else
        su -s /bin/bash ${MINIO_USER} -c "cd ${MINIO_HOME} && nohup ${MINIO_HOME}/start-minio.sh > ${MINIO_HOME}/logs/minio.log 2>&1 &"
    fi
    sleep 3
    
    # 检查进程是否运行
    if pgrep -u ${MINIO_USER} -f "minio server" > /dev/null; then
        echo -e "${GREEN}✓ MinIO 服务启动成功（使用 nohup）${NC}"
        echo -e "${YELLOW}  日志文件: ${MINIO_HOME}/logs/minio.log${NC}"
        
        # 创建停止脚本
        cat > ${MINIO_HOME}/stop-minio.sh <<'STOPEOF'
#!/bin/bash
pkill -u minio -f "minio server"
echo "MinIO 服务已停止"
STOPEOF
        chmod +x ${MINIO_HOME}/stop-minio.sh
        chown ${MINIO_USER}:${MINIO_GROUP} ${MINIO_HOME}/stop-minio.sh
        
        # 创建重启脚本
        cat > ${MINIO_HOME}/restart-minio.sh <<EOF
#!/bin/bash
SCRIPT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
\$SCRIPT_DIR/stop-minio.sh
sleep 2
if command -v runuser &> /dev/null; then
    runuser -u ${MINIO_USER} -- bash -c "cd \\\${SCRIPT_DIR} && nohup \\\${SCRIPT_DIR}/start-minio.sh > \\\${SCRIPT_DIR}/logs/minio.log 2>&1 &"
else
    su -s /bin/bash ${MINIO_USER} -c "cd \\\${SCRIPT_DIR} && nohup \\\${SCRIPT_DIR}/start-minio.sh > \\\${SCRIPT_DIR}/logs/minio.log 2>&1 &"
fi
echo "MinIO 服务已重启"
EOF
        chmod +x ${MINIO_HOME}/restart-minio.sh
        chown ${MINIO_USER}:${MINIO_GROUP} ${MINIO_HOME}/restart-minio.sh
        
    else
        echo -e "${RED}✗ MinIO 服务启动失败，请检查日志: ${MINIO_HOME}/logs/minio.log${NC}"
        exit 1
    fi
fi

# 9. 配置防火墙（如果存在）
echo -e "${YELLOW}[9/9] 配置防火墙...${NC}"
if command -v firewall-cmd &> /dev/null; then
    firewall-cmd --permanent --add-port=${MINIO_PORT}/tcp
    firewall-cmd --permanent --add-port=${MINIO_CONSOLE_PORT}/tcp
    firewall-cmd --reload
    echo -e "${GREEN}✓ 防火墙规则已添加${NC}"
elif command -v ufw &> /dev/null; then
    ufw allow ${MINIO_PORT}/tcp
    ufw allow ${MINIO_CONSOLE_PORT}/tcp
    echo -e "${GREEN}✓ 防火墙规则已添加${NC}"
else
    echo -e "${YELLOW}未检测到防火墙，请手动开放端口 ${MINIO_PORT} 和 ${MINIO_CONSOLE_PORT}${NC}"
fi

# 显示服务信息
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}MinIO 部署完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo -e "API 地址:     http://$(hostname -I | awk '{print $1}'):${MINIO_PORT}"
echo -e "控制台地址:   http://$(hostname -I | awk '{print $1}'):${MINIO_CONSOLE_PORT}"
echo -e "Root User:    ${MINIO_ROOT_USER}"
echo -e "Root Password: ${MINIO_ROOT_PASSWORD}"
echo -e "${RED}请妥善保存上述信息！${NC}"
echo ""
echo -e "常用命令："
if [ "$SYSTEMD_AVAILABLE" = true ]; then
    echo -e "  查看状态:   systemctl status minio"
    echo -e "  查看日志:   journalctl -u minio -f"
    echo -e "  重启服务:   systemctl restart minio"
    echo -e "  停止服务:   systemctl stop minio"
    echo -e "  启动服务:   systemctl start minio"
else
    echo -e "  查看进程:   ps aux | grep minio"
    echo -e "  查看日志:   tail -f ${MINIO_HOME}/logs/minio.log"
    echo -e "  停止服务:   ${MINIO_HOME}/stop-minio.sh"
    echo -e "  重启服务:   ${MINIO_HOME}/restart-minio.sh"
    echo -e "  启动服务:   ${MINIO_HOME}/restart-minio.sh"
fi
echo ""

