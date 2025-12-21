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

# 6. 创建 systemd 服务文件
echo -e "${YELLOW}[6/8] 创建 systemd 服务文件...${NC}"
cat > /etc/systemd/system/minio.service <<EOF
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
ExecStartPre=/bin/bash -c "if [ -z \"\${MINIO_VOLUMES}\" ]; then echo \"Variable MINIO_VOLUMES not set in \${MINIO_CONFIG_DIR}/minio.env\"; exit 1; fi"
ExecStart=/usr/local/bin/minio server \$MINIO_OPTS \$MINIO_VOLUMES

Restart=always
LimitNOFILE=65536
TasksMax=infinity
SendSIGKILL=no

[Install]
WantedBy=multi-user.target
EOF
echo -e "${GREEN}✓ systemd 服务文件创建成功${NC}"

# 7. 重新加载 systemd 并启动服务
echo -e "${YELLOW}[7/8] 启动 MinIO 服务...${NC}"
systemctl daemon-reload
systemctl enable minio
systemctl start minio
sleep 3

# 检查服务状态
if systemctl is-active --quiet minio; then
    echo -e "${GREEN}✓ MinIO 服务启动成功${NC}"
else
    echo -e "${RED}✗ MinIO 服务启动失败，请检查日志: journalctl -u minio${NC}"
    exit 1
fi

# 8. 配置防火墙（如果存在）
echo -e "${YELLOW}[8/8] 配置防火墙...${NC}"
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
echo -e "  查看状态:   systemctl status minio"
echo -e "  查看日志:   journalctl -u minio -f"
echo -e "  重启服务:   systemctl restart minio"
echo -e "  停止服务:   systemctl stop minio"
echo -e "  启动服务:   systemctl start minio"
echo ""

