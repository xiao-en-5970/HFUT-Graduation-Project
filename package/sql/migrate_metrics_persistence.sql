-- migrate_metrics_persistence.sql
-- ─────────────────────────────────────────────────────────────────────────────
-- 运维面板指标 + bot 事件持久化。
--
-- 设计要点：
--
--   1. 时序统一用"分钟桶"：minute_ts 是对齐到分钟的 epoch 秒（值 % 60 == 0）。
--      这是粒度上限——更细需要 sub-minute 分桶，本系统业务体量不需要。
--   2. metric_minute 是宽长表（long format），单行 = 单个 metric 在某分钟的值；
--      读取时用 GROUP BY metric/floor(minute_ts/step) 做"按 step 下采样"得到
--      折线图数据点；这样面板上"1h vs 7d"切换不必新建 schema，只改 step。
--   3. counters（requests / errors_5xx / dispatch_ok / ...）直接 SUM；
--      平均延迟需要 latency_sum / latency_count 两条 metric 同步存。
--   4. bot 事件表分开（bot_dispatch_event）：每条事件一行，含 AI 判断的 reason。
--      用 fingerprint UNIQUE 做幂等——避免 hfut backend 反复 pull bot 时重复入库。
-- ─────────────────────────────────────────────────────────────────────────────

-- 时序表：所有数值型指标的分钟桶聚合
CREATE TABLE IF NOT EXISTS metric_minute
(
    minute_ts bigint      NOT NULL, -- 对齐到分钟的 epoch 秒
    source    varchar(16) NOT NULL, -- 'http' = hfut 后端, 'bot' = QQ-bot
    metric    varchar(64) NOT NULL, -- 见下方常量列表
    value     bigint      NOT NULL DEFAULT 0,
    PRIMARY KEY (minute_ts, source, metric)
);

-- 按 (source, metric) 时间序列查询的组合索引（最常见的图表请求路径）
CREATE INDEX IF NOT EXISTS idx_metric_minute_source_metric_ts
    ON metric_minute (source, metric, minute_ts);

-- 跨 metric 的按时间窗清理用（DELETE WHERE minute_ts < cutoff）
CREATE INDEX IF NOT EXISTS idx_metric_minute_ts
    ON metric_minute (minute_ts);

COMMENT ON TABLE metric_minute IS
    '运维指标的分钟桶聚合。source/metric 维度长表，便于面板按 step 下采样。
     已采集的 metric：
       source=http: requests, errors_4xx, errors_5xx, biz_errors, latency_sum, latency_count
       source=bot:  ws_message, recognize_ok, recognize_err, recognize_quota,
                    dispatch_ok, dispatch_err, rate_limit, ops_notify,
                    private_access_request';

-- ─────────────────────────────────────────────────────────────────────────────

-- bot 业务事件流：一条 = 一次 AI 识别 + 派发动作
CREATE TABLE IF NOT EXISTS bot_dispatch_event
(
    id          bigserial PRIMARY KEY,
    occurred_at timestamptz NOT NULL,       -- 事件发生时刻（UTC，前端展示前自行转时区）
    group_id    bigint,                     -- QQ 群号；私聊时为 NULL
    user_id     bigint,                     -- QQ 用户号
    action      varchar(32) NOT NULL,       -- publish_good / seek_goods / off_shelf / publish_question / chitchat / ...
    outcome     varchar(16) NOT NULL,       -- ok / err / rate_limited
    title       varchar(256),               -- 商品/求助/问题标题
    category    smallint,                   -- 1=二手 2=求物品 3=求解答
    price_cents integer,                    -- 价格（分），仅对商品/求物品有意义
    confidence  real,                       -- Kimi 给出的置信度 [0,1]
    reason      text,                       -- AI 判断的原因——给运维理解 bot 决策
    err_message text,                       -- outcome=err 时的错误描述
    fingerprint varchar(64) NOT NULL UNIQUE -- bot 端给的去重 hash
);

CREATE INDEX IF NOT EXISTS idx_bot_event_occurred_desc
    ON bot_dispatch_event (occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_bot_event_action_occurred
    ON bot_dispatch_event (action, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_bot_event_outcome_occurred
    ON bot_dispatch_event (outcome, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_bot_event_user_occurred
    ON bot_dispatch_event (user_id, occurred_at DESC);

COMMENT ON TABLE bot_dispatch_event IS
    'QQ-bot 自动识别 + 派发的事件流。
     fingerprint = sha1(group_id|user_id|action|occurred_at_unix) 由 bot 计算并保证幂等。
     hfut backend 每分钟从 /internal/metrics/events 拉一次增量入库。';
