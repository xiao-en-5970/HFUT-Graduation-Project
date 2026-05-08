// bot_admin_sql.go —— 给 QQ-bot 运维群提供"任意只读 SQL"查询入口。
//
// 设计要点（**安全优先**）：
//
//  1. 鉴权：本接口挂在 /api/v1/bot/* 路由组下，由 BotServiceAuth 保护——只有
//     持有共享 secret 自签 service JWT 的 bot 进程能调用。外部 / 普通用户进不来。
//
//  2. 静态白名单：解析 SQL 字符串首个非空白非注释 token，必须是
//     select / with / explain / show / values 之一；任何 update / insert / delete /
//     create / alter / drop / truncate / grant / vacuum / copy 直接拒绝。
//
//  3. 多语句拒绝：包含中间分号的语句一律拒绝，避免 "select 1; drop table"。
//
//  4. PG 事务级 READ ONLY：开 read-only tx + SET LOCAL statement_timeout=10s。
//     即便绕过静态检查，PG 也会在执行 DML/DDL 时返回 "cannot execute UPDATE in
//     a read-only transaction"。这是双保险。
//
//  5. 行数上限：SELECT/WITH 包一层 `SELECT * FROM (...) LIMIT N`，N 默认 200，
//     最大 1000。EXPLAIN/SHOW/VALUES 不包，按原样执行。
//
//  6. 字段截断：单 cell 超过 4KB 截断（PG bytea / 长 jsonb 才会触发）。
//
//  7. 不允许跨 schema：当前 PG 连的是 graduation_project 库，cross-schema 仅限
//     pg_catalog / information_schema 这种系统视图，本系统不阻断（运维查 schema
//     是合理需求）。
package controller

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)

// BotAdminExecSQL POST /api/v1/bot/admin/sql
//
// Body:
//
//	{
//	  "sql":   "SELECT count(*) FROM users",
//	  "limit": 200            // 可选；默认 200，上限 1000
//	}
//
// 200 响应：
//
//	{
//	  "code": 200, "data": {
//	    "columns":     ["count"],
//	    "column_types": ["BIGINT"],
//	    "rows":         [[1234]],
//	    "row_count":     1,
//	    "elapsed_ms":    7
//	  }
//	}
//
// 400：SQL 静态检查失败（如非 SELECT、含分号、空 SQL）
// 500：执行失败（含 PG 拒绝写入 / 超时 / schema 错误）
func BotAdminExecSQL(c *gin.Context) {
	var body struct {
		SQL   string `json:"sql"   binding:"required"`
		Limit int    `json:"limit,omitempty"`
	}
	if err := c.BindJSON(&body); err != nil {
		reply.ReplyInvalidParams(c, err)
		return
	}
	body.SQL = strings.TrimSpace(body.SQL)
	if body.SQL == "" {
		reply.ReplyInvalidParams(c, errors.New("sql 不能为空"))
		return
	}
	if body.Limit <= 0 || body.Limit > 1000 {
		body.Limit = 200
	}

	kind, normalized, err := validateReadOnlySQL(body.SQL)
	if err != nil {
		reply.ReplyInvalidParams(c, err)
		return
	}

	finalSQL := normalized
	// 仅对 SELECT/WITH 强加 LIMIT；EXPLAIN/SHOW/VALUES 维持原样
	if kind == "select" || kind == "with" {
		finalSQL = fmt.Sprintf("SELECT * FROM (%s) AS _bot_admin_q LIMIT %d", normalized, body.Limit)
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 12*time.Second)
	defer cancel()

	t0 := time.Now()
	cols, types, rows, err := execReadOnlyQuery(ctx, finalSQL)
	if err != nil {
		reply.ReplyInternalError(c, err)
		return
	}
	reply.ReplyOKWithData(c, gin.H{
		"columns":      cols,
		"column_types": types,
		"rows":         rows,
		"row_count":    len(rows),
		"elapsed_ms":   time.Since(t0).Milliseconds(),
	})
}

// validateReadOnlySQL 静态检查 + 末尾分号清洗。
//
// 返回 (kind, normalized, err)：
//
//	kind        识别出的语句种类：select / with / explain / show / values
//	normalized  去掉末尾分号 + 首尾空白的 SQL，调用方拿来构造最终 SQL
//	err         不通过的原因（bind 给前端时是 400）
func validateReadOnlySQL(in string) (string, string, error) {
	s := strings.TrimSpace(in)
	s = strings.TrimSuffix(s, ";")
	s = strings.TrimSpace(s)

	// 多语句拒绝：检查是否还有中间分号（在字符串字面量里的分号也会被一刀切，
	// 这是有意为之——运维查询不该需要嵌入分号字面量；强行 escape 也会被静态检查挡住）
	if strings.Contains(s, ";") {
		return "", "", errors.New("禁止多语句执行：SQL 中不允许包含分号")
	}

	first, err := firstSQLToken(s)
	if err != nil {
		return "", "", err
	}
	switch first {
	case "select", "with", "explain", "show", "values":
		return first, s, nil
	default:
		return "", "", fmt.Errorf("仅允许 select/with/explain/show/values 开头的只读语句，收到首词 %q", first)
	}
}

// firstSQLToken 跳过前置注释与空白，取第一个 SQL 单词（小写）。
//
// 支持：
//
//	-- 行注释 直到 \n
//	/* 块注释 */
//	空白 / tab / \r / \n
//
// 找不到任何 token 时返回错误。
func firstSQLToken(s string) (string, error) {
	i := 0
	for i < len(s) {
		switch {
		case s[i] == ' ' || s[i] == '\t' || s[i] == '\r' || s[i] == '\n':
			i++
		case i+1 < len(s) && s[i] == '-' && s[i+1] == '-':
			// 行注释吃到 \n
			j := strings.IndexByte(s[i:], '\n')
			if j < 0 {
				return "", errors.New("SQL 仅含注释，没有可执行语句")
			}
			i += j + 1
		case i+1 < len(s) && s[i] == '/' && s[i+1] == '*':
			// 块注释吃到 */
			j := strings.Index(s[i+2:], "*/")
			if j < 0 {
				return "", errors.New("SQL 块注释未闭合")
			}
			i += j + 4
		default:
			// 把当前位置开始的连续字母收进来
			start := i
			for i < len(s) && isIdentByte(s[i]) {
				i++
			}
			if i == start {
				return "", fmt.Errorf("SQL 首字符 %q 不像合法关键字", s[start])
			}
			return strings.ToLower(s[start:i]), nil
		}
	}
	return "", errors.New("SQL 为空")
}

func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || b == '_'
}

// execReadOnlyQuery 在 read-only 事务里执行 SQL，把结果转成 JSON 友好形式。
//
// 行结构是 [][]any（cells），每个 cell 用 driver scan 拿到的原生 Go 类型——
// time.Time 转 RFC3339，[]byte 大于 4KB 截断，其余直接走 JSON 默认编码。
func execReadOnlyQuery(ctx context.Context, sqlStr string) ([]string, []string, [][]any, error) {
	rawDB, err := pgsql.DB.WithContext(ctx).DB()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("get raw DB: %w", err)
	}

	tx, err := rawDB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SET LOCAL statement_timeout = '10s'"); err != nil {
		return nil, nil, nil, fmt.Errorf("set timeout: %w", err)
	}

	rows, err := tx.QueryContext(ctx, sqlStr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("columns: %w", err)
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("column types: %w", err)
	}
	typeNames := make([]string, len(colTypes))
	for i, ct := range colTypes {
		typeNames[i] = ct.DatabaseTypeName()
	}

	const maxCellBytes = 4 * 1024
	out := make([][]any, 0, 32)
	for rows.Next() {
		holders := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range holders {
			ptrs[i] = &holders[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, nil, nil, fmt.Errorf("scan: %w", err)
		}
		row := make([]any, len(cols))
		for i, v := range holders {
			row[i] = normalizeCell(v, maxCellBytes)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, fmt.Errorf("rows iter: %w", err)
	}
	return cols, typeNames, out, nil
}

// normalizeCell 把 driver 返回的原生 Go 值变成 JSON 友好形式。
//
//   - time.Time → RFC3339 字符串
//   - []byte → 优先按 UTF-8 字符串解；超过 maxBytes 截断标记 "...(truncated)"
//   - 其它原样（int64 / float64 / bool / string / nil 等都能直接 JSON）
func normalizeCell(v any, maxBytes int) any {
	switch x := v.(type) {
	case nil:
		return nil
	case time.Time:
		return x.Format(time.RFC3339Nano)
	case []byte:
		if len(x) > maxBytes {
			return string(x[:maxBytes]) + "...(truncated)"
		}
		return string(x)
	case string:
		if len(x) > maxBytes {
			return x[:maxBytes] + "...(truncated)"
		}
		return x
	default:
		return x
	}
}
