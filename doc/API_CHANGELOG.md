# API 变更记录

每次接口文档更新时，在此记录变更内容，供前端同步适配。

格式：按日期倒序，每条列出 **日期**、**变更类型**（新增/修改/废弃）、** affected 接口**、**说明**。

---

## 2025-02-17（学校登录与绑定）

### 新增

| 接口                               | 变更说明                                            |
|----------------------------------|-------------------------------------------------|
| `POST /api/v1/user/school-login` | 学校端登录，仅需 school_code、username、password，对接学校 CAS |
| `user_cert` 表                    | 记录用户在某学校的认证信息，cert_info 为 JSONB 存储学生信息          |

### 修改

| 接口                              | 变更说明                                                                                                        |
|---------------------------------|-------------------------------------------------------------------------------------------------------------|
| `POST /api/v1/user/bind/school` | **Breaking**：请求体改为 `{ school_id, username, password }`，需学校端账号密码验证；成功后写入 user_cert                           |
| `GET /api/v1/search/articles`   | 热度公式配比全部环境变量可配置：SEARCH_WEIGHT_COLLECT/LIKE/VIEW(10/5/1)、SEARCH_INTERACTION_DECAY_DAYS(90)、SEARCH_COMBINED_* |

---

## 2025-02-17

### 修改

| 接口                            | 变更说明                                                      |
|-------------------------------|-----------------------------------------------------------|
| `GET /api/v1/search/articles` | 描述：补充 zhparser 中文智能分词；排序说明改为 zhparser                     |
| `GET /api/v1/post/search`     | 描述：补充 zhparser；`q` 改为可选，空则退化为列表；page/pageSize 默认值         |
| `GET /api/v1/question/search` | 同上                                                        |
| `GET /api/v1/answer/search`   | 同上                                                        |
| `GET /api/v1/post/drafts`     | 描述：补充 data 结构 `{list,total,page,page_size}`，list 含 author |
| `GET /api/v1/question/drafts` | 同上                                                        |
| `GET /api/v1/answer/drafts`   | 同上                                                        |

---

*后续更新请在本文件顶部追加新日期条目。*
