.PHONY: tag

# 版本发布：Tag=1.1.10 make tag 或 Tag=v1.1.10 make tag
# 未带 v 前缀会自动补全，以匹配 CI 的 v* tag 触发
tag:
ifndef Tag
	$(error 请指定 Tag，如: Tag=1.1.10 make tag)
endif
	@t="$(Tag)"; [ "$${t#v}" = "$$t" ] && t="v$$t"; \
	echo "创建并推送 tag: $$t"; \
	git tag $$t && git push origin $$t
