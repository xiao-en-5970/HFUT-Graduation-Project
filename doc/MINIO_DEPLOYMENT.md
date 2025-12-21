# MinIO 部署指南

本指南介绍如何在 Linux 服务器上部署 MinIO 对象存储服务（不使用 Docker）。

## 前置要求

- Linux 服务器（推荐 Ubuntu 20.04+ 或 CentOS 7+）
- Root 权限
- 至少 2GB 可用磁盘空间
- 网络连接（用于下载 MinIO 二进制文件）

## 快速部署

### 方法一：使用部署脚本（推荐）

1. **上传脚本到服务器**
   ```bash
   scp scripts/deploy-minio.sh root@your-server:/tmp/
   ```

2. **在服务器上执行**
   ```bash
   ssh root@your-server
   chmod +x /tmp/deploy-minio.sh
   /tmp/deploy-minio.sh
   ```

3. **脚本会自动完成以下操作：**
   - 创建 MinIO 用户和组
   - 创建必要的目录结构
   - 下载并安装 MinIO 二进制文件
   - 配置环境变量
   - 创建 systemd 服务
   - 启动并启用 MinIO 服务
   - 配置防火墙规则

### 方法二：手动部署

#### 1. 创建 MinIO 用户

```bash
groupadd -r minio
useradd -r -g minio -d /opt/minio -s /sbin/nologin minio
```

#### 2. 创建目录结构

```bash
mkdir -p /opt/minio
mkdir -p /data/minio
mkdir -p /etc/minio
chown -R minio:minio /opt/minio
chown -R minio:minio /data/minio
chown -R minio:minio /etc/minio
```

#### 3. 下载 MinIO

```bash
cd /tmp
wget https://dl.min.io/server/minio/release/linux-amd64/archive/minio.RELEASE.2024-12-13T18-30-20Z
chmod +x minio.RELEASE.2024-12-13T18-30-20Z
cp minio.RELEASE.2024-12-13T18-30-20Z /usr/local/bin/minio
chown minio:minio /usr/local/bin/minio
```

#### 4. 创建环境变量文件

```bash
cat > /etc/minio/minio.env <<EOF
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=your-strong-password-here
MINIO_VOLUMES="/data/minio"
MINIO_OPTS="--address :50001 --console-address :50002"
EOF

chown minio:minio /etc/minio/minio.env
chmod 600 /etc/minio/minio.env
```

**重要：** 请将 `your-strong-password-here` 替换为强密码。

#### 5. 创建 systemd 服务

```bash
cat > /etc/systemd/system/minio.service <<EOF
[Unit]
Description=MinIO Object Storage
Documentation=https://docs.min.io
Wants=network-online.target
After=network-online.target
AssertFileIsExecutable=/usr/local/bin/minio

[Service]
WorkingDirectory=/opt/minio
User=minio
Group=minio
EnvironmentFile=/etc/minio/minio.env
ExecStartPre=/bin/bash -c "if [ -z \"\${MINIO_VOLUMES}\" ]; then echo \"Variable MINIO_VOLUMES not set in /etc/minio/minio.env\"; exit 1; fi"
ExecStart=/usr/local/bin/minio server \$MINIO_OPTS \$MINIO_VOLUMES

Restart=always
LimitNOFILE=65536
TasksMax=infinity
SendSIGKILL=no

[Install]
WantedBy=multi-user.target
EOF
```

#### 6. 启动服务

```bash
systemctl daemon-reload
systemctl enable minio
systemctl start minio
systemctl status minio
```

#### 7. 配置防火墙

**firewalld (CentOS/RHEL):**
```bash
firewall-cmd --permanent --add-port=50001/tcp
firewall-cmd --permanent --add-port=50002/tcp
firewall-cmd --reload
```

**ufw (Ubuntu/Debian):**
```bash
ufw allow 50001/tcp
ufw allow 50002/tcp
```

## 验证部署

### 检查服务状态

```bash
systemctl status minio
```

### 查看日志

```bash
journalctl -u minio -f
```

### 测试访问

1. **API 访问**（用于应用程序）
   ```
   http://your-server-ip:50001
   ```

2. **Web 控制台访问**（用于管理）
   ```
   http://your-server-ip:50002
   ```
   使用配置的 `MINIO_ROOT_USER` 和 `MINIO_ROOT_PASSWORD` 登录

## 常用管理命令

```bash
# 启动服务
systemctl start minio

# 停止服务
systemctl stop minio

# 重启服务
systemctl restart minio

# 查看状态
systemctl status minio

# 查看日志
journalctl -u minio -f

# 禁用开机自启
systemctl disable minio

# 启用开机自启
systemctl enable minio
```

## 配置说明

### 端口配置

- **50001**: MinIO API 端口（用于应用程序访问）
- **50002**: MinIO Console 端口（用于 Web 管理界面）

如需修改端口，编辑 `/etc/minio/minio.env` 文件中的 `MINIO_OPTS` 参数：

```bash
MINIO_OPTS="--address :新端口 --console-address :新控制台端口"
```

然后重启服务：
```bash
systemctl restart minio
```

### 数据存储路径

默认数据存储在 `/data/minio`。如需修改：

1. 编辑 `/etc/minio/minio.env`：
   ```bash
   MINIO_VOLUMES="/新路径"
   ```

2. 创建目录并设置权限：
   ```bash
   mkdir -p /新路径
   chown -R minio:minio /新路径
   ```

3. 重启服务：
   ```bash
   systemctl restart minio
   ```

### 修改 Root 密码

1. 编辑 `/etc/minio/minio.env`：
   ```bash
   MINIO_ROOT_PASSWORD=新密码
   ```

2. 重启服务：
   ```bash
   systemctl restart minio
   ```

## 安全建议

1. **使用强密码**：确保 `MINIO_ROOT_PASSWORD` 是强密码
2. **限制访问**：使用防火墙限制只有特定 IP 可以访问
3. **使用 HTTPS**：生产环境建议配置 SSL/TLS 证书
4. **定期备份**：定期备份 `/data/minio` 目录
5. **监控日志**：定期检查日志文件，关注异常访问

## 故障排查

### 服务无法启动

1. 检查日志：
   ```bash
   journalctl -u minio -n 50
   ```

2. 检查权限：
   ```bash
   ls -la /data/minio
   ls -la /usr/local/bin/minio
   ```

3. 检查端口占用：
   ```bash
   netstat -tlnp | grep 50001
   ```

### 无法访问 Web 控制台

1. 检查防火墙规则
2. 检查服务是否运行：`systemctl status minio`
3. 检查端口监听：`netstat -tlnp | grep 50002`

### 忘记密码

1. 停止服务：
   ```bash
   systemctl stop minio
   ```

2. 编辑配置文件：
   ```bash
   vi /etc/minio/minio.env
   # 修改 MINIO_ROOT_PASSWORD
   ```

3. 启动服务：
   ```bash
   systemctl start minio
   ```

## 与 Go 应用集成

在 Go 应用中使用 MinIO，需要安装 MinIO Go SDK：

```bash
go get github.com/minio/minio-go/v7
```

示例代码：

```go
package main

import (
    "context"
    "log"
    "github.com/minio/minio-go/v7"
    "github.com/minio/minio-go/v7/pkg/credentials"
)

func main() {
    endpoint := "your-server-ip:50001"
    accessKeyID := "minioadmin"
    secretAccessKey := "your-password"
    useSSL := false

    minioClient, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
        Secure: useSSL,
    })
    if err != nil {
        log.Fatalln(err)
    }

    // 创建 bucket
    bucketName := "my-bucket"
    location := "us-east-1"

    err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{Region: location})
    if err != nil {
        exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
        if errBucketExists == nil && exists {
            log.Printf("Bucket %s already exists\n", bucketName)
        } else {
            log.Fatalln(err)
        }
    } else {
        log.Printf("Successfully created bucket %s\n", bucketName)
    }
}
```

## 参考资源

- [MinIO 官方文档](https://docs.min.io/)
- [MinIO Go SDK 文档](https://docs.min.io/docs/golang-client-quickstart-guide.html)
- [MinIO GitHub](https://github.com/minio/minio)

