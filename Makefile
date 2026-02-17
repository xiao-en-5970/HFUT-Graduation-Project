# 同步 .env 到服务器（首次或配置变更时执行）
# 用法: make deploy-env 或 DEPLOY_HOST=47.94.197.213 DEPLOY_USER=root make deploy-env
deploy-env:
	@HOST="$${DEPLOY_HOST:-47.94.197.213}"; USER="$${DEPLOY_USER:-root}"; \
	echo "scp .env $${USER}@$${HOST}:/opt/app/.env"; \
	ssh $${USER}@$${HOST} "mkdir -p /opt/app"; \
	scp .env $${USER}@$${HOST}:/opt/app/.env
