package service

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/config"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	rediscli "github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/redis"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"go.uber.org/zap"
)

// 推荐系统（方案 B）：标签画像 + 双路召回 + ε-greedy 打散 + refresh_token 稳定分页

// UserProfile 用户兴趣画像
type UserProfile struct {
	UserID      uint      `json:"user_id"`
	TopTags     []string  `json:"top_tags"`    // 降序，top-N
	TopAuthors  []int     `json:"top_authors"` // 降序，top-N
	IsColdStart bool      `json:"is_cold_start"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type recommendService struct{}

var recoSrv = &recommendService{}

// Recommend 返回推荐服务单例
func Recommend() *recommendService { return recoSrv }

// BuildProfile 构建/读取用户画像（Redis 缓存优先，miss 时从 DB 聚合并回写）
func (s *recommendService) BuildProfile(ctx context.Context, userID uint) (*UserProfile, error) {
	if userID == 0 {
		return &UserProfile{IsColdStart: true}, nil
	}
	key := profileCacheKey(userID)
	if rediscli.Client != nil {
		if raw, err := rediscli.Client.Get(ctx, key).Result(); err == nil && raw != "" {
			var p UserProfile
			if jerr := json.Unmarshal([]byte(raw), &p); jerr == nil {
				return &p, nil
			}
		}
	}

	profile := &UserProfile{UserID: userID, UpdatedAt: time.Now()}
	// 冷启动判定：近 30 天 <3 条行为即视为冷启动，画像为空
	cnt, err := dao.UserBehavior().CountRecent(ctx, userID, config.RecoRecentDays)
	if err != nil {
		return nil, err
	}
	if cnt < 3 {
		profile.IsColdStart = true
		s.cacheProfile(ctx, profile)
		return profile, nil
	}

	tags, err := dao.UserBehavior().AggregateTopTags(ctx, userID, config.RecoRecentDays, config.RecoBehaviorTimeDecayDays, config.RecoTopTagsLimit)
	if err != nil {
		logger.Warn(ctx, "recommend: aggregate top tags failed", zap.Error(err))
	}
	authors, err := dao.UserBehavior().AggregateTopAuthors(ctx, userID, config.RecoRecentDays, config.RecoBehaviorTimeDecayDays, config.RecoTopAuthorsLimit)
	if err != nil {
		logger.Warn(ctx, "recommend: aggregate top authors failed", zap.Error(err))
	}

	for _, t := range tags {
		name := strings.TrimSpace(t.Name)
		if name != "" {
			profile.TopTags = append(profile.TopTags, name)
		}
	}
	for _, a := range authors {
		if a.AuthorID > 0 {
			profile.TopAuthors = append(profile.TopAuthors, a.AuthorID)
		}
	}

	// 若 tags 和 authors 都空（例如 tags 表为空），仍标记为冷启动以走纯探索
	if len(profile.TopTags) == 0 && len(profile.TopAuthors) == 0 {
		profile.IsColdStart = true
	}
	s.cacheProfile(ctx, profile)
	return profile, nil
}

func (s *recommendService) cacheProfile(ctx context.Context, p *UserProfile) {
	if rediscli.Client == nil || p == nil || p.UserID == 0 {
		return
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return
	}
	ttl := time.Duration(config.RecoProfileTTL) * time.Second
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	_ = rediscli.Client.Set(ctx, profileCacheKey(p.UserID), raw, ttl).Err()
}

// InvalidateProfile 用户产生较强信号（like/collect/comment）时主动失效画像，下次请求会重算
func (s *recommendService) InvalidateProfile(ctx context.Context, userID uint) {
	if rediscli.Client == nil || userID == 0 {
		return
	}
	_ = rediscli.Client.Del(ctx, profileCacheKey(userID)).Err()
}

// RecordBehavior 异步打点：不阻塞主流程，失败仅日志警告
func (s *recommendService) RecordBehavior(ctx context.Context, userID uint, extType int, extID int, action int, keyword string) {
	if userID == 0 {
		return
	}
	weight := behaviorWeight(action)
	ub := &model.UserBehavior{
		UserID:  int(userID),
		ExtType: int16(extType),
		ExtID:   extID,
		Action:  int16(action),
		Weight:  float32(weight),
		Keyword: keyword,
	}
	// 解绑 request ctx：异步落库不要被 gin 请求结束打断
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := dao.UserBehavior().Record(bgCtx, ub); err != nil {
			logger.Warn(bgCtx, "recommend: record behavior failed",
				zap.Uint("user_id", userID),
				zap.Int("ext_type", extType),
				zap.Int("ext_id", extID),
				zap.Int("action", action),
				zap.Error(err))
		}
	}()

	// 强正向信号：主动失效画像缓存，下次请求重算
	if action == constant.BehaviorLike || action == constant.BehaviorCollect || action == constant.BehaviorComment {
		s.InvalidateProfile(ctx, userID)
	}
}

func behaviorWeight(action int) float64 {
	switch action {
	case constant.BehaviorView:
		return config.RecoBehaviorWeightView
	case constant.BehaviorLike:
		return config.RecoBehaviorWeightLike
	case constant.BehaviorCollect:
		return config.RecoBehaviorWeightCollect
	case constant.BehaviorComment:
		return config.RecoBehaviorWeightComment
	case constant.BehaviorSearch:
		return config.RecoBehaviorWeightSearch
	case constant.BehaviorUnlike, constant.BehaviorUncollect:
		// 反向信号不存正分，记 0（仅作为历史痕迹）
		return 0
	}
	return 1
}

// SeenIDs 返回用户在当前推荐流中已曝光 + 近 N 天 view/like/collect/comment 过的 ext_id 合集，用于去重
func (s *recommendService) SeenIDs(ctx context.Context, userID uint, extType int) ([]int, error) {
	if userID == 0 {
		return nil, nil
	}
	ids := make(map[int]struct{}, 128)
	// 1. 历史行为（从 DB，近 N 天）
	if list, err := dao.UserBehavior().RecentViewedIDs(ctx, userID, extType, config.RecoSeenTTLDays); err == nil {
		for _, id := range list {
			ids[id] = struct{}{}
		}
	}
	// 2. 当前推荐流曝光（Redis SET，TTL N 天）
	if rediscli.Client != nil {
		key := seenKey(userID, extType)
		if members, err := rediscli.Client.SMembers(ctx, key).Result(); err == nil {
			for _, m := range members {
				var v int
				if _, err := fmt.Sscanf(m, "%d", &v); err == nil && v > 0 {
					ids[v] = struct{}{}
				}
			}
		}
	}
	out := make([]int, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	return out, nil
}

// MarkSeenArticles 把 ids 加入当前用户的 seen 集合（Redis SET，TTL N 天）
func (s *recommendService) MarkSeenArticles(ctx context.Context, userID uint, extType int, ids []int) {
	if rediscli.Client == nil || userID == 0 || len(ids) == 0 {
		return
	}
	key := seenKey(userID, extType)
	members := make([]interface{}, len(ids))
	for i, id := range ids {
		members[i] = fmt.Sprintf("%d", id)
	}
	pipe := rediscli.Client.Pipeline()
	pipe.SAdd(ctx, key, members...)
	ttl := time.Duration(config.RecoSeenTTLDays) * 24 * time.Hour
	if ttl <= 0 {
		ttl = 3 * 24 * time.Hour
	}
	pipe.Expire(ctx, key, ttl)
	_, _ = pipe.Exec(ctx)
}

// EnsureRefreshToken 为推荐请求生成/复用 refresh_token：
// - 传入为空则生成 16 字节随机 token 的 hex 形式；
// - 传入非空则原样返回（用来翻页稳定）
func (s *recommendService) EnsureRefreshToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		var b [16]byte
		if _, err := cryptorand.Read(b[:]); err != nil {
			// 退化：时间戳 + 伪随机
			return fmt.Sprintf("%x", time.Now().UnixNano())
		}
		return hex.EncodeToString(b[:])
	}
	if len(token) > 64 {
		return token[:64]
	}
	return token
}

// RecallArticles 执行「兴趣池 + 探索池」双路召回，合并并按 ε-greedy 打散后返回 pageSize 条
// articleType: 0=全部 1帖 2问 3答
// page: 从 1 起；refreshToken 用于同一次推荐流翻页稳定
func (s *recommendService) RecallArticles(
	ctx context.Context,
	userID uint,
	viewerSchoolID uint,
	articleType int,
	page, pageSize int,
	refreshToken string,
) ([]*model.Article, int64, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}
	profile, err := s.BuildProfile(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	seenIDs, _ := s.SeenIDs(ctx, userID, articleType)
	// 兴趣配额 & 探索配额：分页时按 overall 比例放大抓取，避免第 2 页撞空
	interestQuota := config.RecoInterestQuota
	if interestQuota <= 0 || interestQuota > 1 {
		interestQuota = 0.6
	}
	if profile.IsColdStart {
		interestQuota = 0 // 冷启动：100% 探索
	}

	// 抓取规模：按 page × pageSize × multiplier 抓足，保证翻页到第 N 页也有内容
	fetchTotal := page * pageSize * config.RecoCandidateMultiplier
	if fetchTotal < pageSize*3 {
		fetchTotal = pageSize * 3
	}
	fetchInterest := int(float64(fetchTotal) * interestQuota)
	fetchExplore := fetchTotal - fetchInterest

	commonPop := struct {
		C, L, V int
	}{
		C: config.SearchWeightCollect,
		L: config.SearchWeightLike,
		V: config.SearchWeightView,
	}

	// 兴趣池
	var interestList []*model.Article
	if fetchInterest > 0 && !profile.IsColdStart {
		interestList, err = dao.Article().ArticleRecommendCandidates(ctx, dao.ArticleRecommendParams{
			ArticleType:       articleType,
			ViewerSchoolID:    viewerSchoolID,
			TopTagNames:       profile.TopTags,
			TopAuthorIDs:      profile.TopAuthors,
			ExcludeIDs:        seenIDs,
			Limit:             fetchInterest,
			FreshnessDecay:    config.RecoFreshnessDecayDays,
			PopularityCollect: commonPop.C,
			PopularityLike:    commonPop.L,
			PopularityView:    commonPop.V,
			RequireInterest:   true,
		})
		if err != nil {
			return nil, 0, err
		}
	}
	interestIDs := articleIDSet(interestList)
	// 探索池：排除兴趣池和 seen
	exploreExclude := append([]int{}, seenIDs...)
	for id := range interestIDs {
		exploreExclude = append(exploreExclude, id)
	}
	var exploreList []*model.Article
	if fetchExplore > 0 {
		exploreList, err = dao.Article().ArticleRecommendCandidates(ctx, dao.ArticleRecommendParams{
			ArticleType:       articleType,
			ViewerSchoolID:    viewerSchoolID,
			ExcludeIDs:        exploreExclude,
			Limit:             fetchExplore,
			FreshnessDecay:    config.RecoFreshnessDecayDays,
			PopularityCollect: commonPop.C,
			PopularityLike:    commonPop.L,
			PopularityView:    commonPop.V,
			RequireInterest:   false,
		})
		if err != nil {
			return nil, 0, err
		}
	}

	// 合并：ε-greedy 打散，按 interestQuota:1-interestQuota 交替取
	merged := mergeArticles(interestList, exploreList, interestQuota, refreshToken, userID)

	// 分页切片（稳定：在同一 refreshToken 下顺序一致）
	total := int64(len(merged))
	start := (page - 1) * pageSize
	if start >= len(merged) {
		return []*model.Article{}, total, nil
	}
	end := start + pageSize
	if end > len(merged) {
		end = len(merged)
	}
	pageList := merged[start:end]

	// 标记曝光（异步）
	go func(ids []int) {
		if len(ids) == 0 {
			return
		}
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.MarkSeenArticles(bgCtx, userID, articleType, ids)
	}(articleIDs(pageList))

	return pageList, total, nil
}

// RecallGoods 商品推荐召回，语义同 RecallArticles
func (s *recommendService) RecallGoods(
	ctx context.Context,
	userID uint,
	viewerSchoolID uint,
	page, pageSize int,
	refreshToken string,
) ([]*model.Good, int64, error) {
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	if page < 1 {
		page = 1
	}
	profile, err := s.BuildProfile(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	// 商品走独立 seen（ext_type=4），避免和文章互相污染
	seenIDs, _ := s.SeenIDs(ctx, userID, constant.ExtTypeGoods)

	interestQuota := config.RecoInterestQuota
	if profile.IsColdStart {
		interestQuota = 0
	}
	fetchTotal := page * pageSize * config.RecoCandidateMultiplier
	if fetchTotal < pageSize*3 {
		fetchTotal = pageSize * 3
	}
	fetchInterest := int(float64(fetchTotal) * interestQuota)
	fetchExplore := fetchTotal - fetchInterest

	var interestList []*model.Good
	if fetchInterest > 0 && !profile.IsColdStart {
		interestList, err = dao.Good().GoodRecommendCandidates(ctx, dao.GoodRecommendParams{
			ViewerSchoolID:    viewerSchoolID,
			TopTagNames:       profile.TopTags,
			TopAuthorIDs:      profile.TopAuthors,
			ExcludeIDs:        seenIDs,
			Limit:             fetchInterest,
			FreshnessDecay:    config.RecoFreshnessDecayDays,
			PopularityCollect: config.SearchWeightCollect,
			PopularityLike:    config.SearchWeightLike,
			RequireInterest:   true,
		})
		if err != nil {
			return nil, 0, err
		}
	}
	exploreExclude := append([]int{}, seenIDs...)
	for _, g := range interestList {
		exploreExclude = append(exploreExclude, int(g.ID))
	}
	var exploreList []*model.Good
	if fetchExplore > 0 {
		exploreList, err = dao.Good().GoodRecommendCandidates(ctx, dao.GoodRecommendParams{
			ViewerSchoolID:    viewerSchoolID,
			ExcludeIDs:        exploreExclude,
			Limit:             fetchExplore,
			FreshnessDecay:    config.RecoFreshnessDecayDays,
			PopularityCollect: config.SearchWeightCollect,
			PopularityLike:    config.SearchWeightLike,
			RequireInterest:   false,
		})
		if err != nil {
			return nil, 0, err
		}
	}

	merged := mergeGoods(interestList, exploreList, interestQuota, refreshToken, userID)
	total := int64(len(merged))
	start := (page - 1) * pageSize
	if start >= len(merged) {
		return []*model.Good{}, total, nil
	}
	end := start + pageSize
	if end > len(merged) {
		end = len(merged)
	}
	pageList := merged[start:end]

	go func(ids []int) {
		if len(ids) == 0 {
			return
		}
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.MarkSeenArticles(bgCtx, userID, constant.ExtTypeGoods, ids)
	}(goodIDs(pageList))

	return pageList, total, nil
}

// ---------- helpers ----------

func profileCacheKey(userID uint) string {
	return fmt.Sprintf("recos:profile:%d", userID)
}

func seenKey(userID uint, extType int) string {
	return fmt.Sprintf("recos:seen:%d:%d", userID, extType)
}

// articleIDSet / articleIDs
func articleIDSet(list []*model.Article) map[int]struct{} {
	m := make(map[int]struct{}, len(list))
	for _, a := range list {
		m[int(a.ID)] = struct{}{}
	}
	return m
}
func articleIDs(list []*model.Article) []int {
	out := make([]int, len(list))
	for i, a := range list {
		out[i] = int(a.ID)
	}
	return out
}
func goodIDs(list []*model.Good) []int {
	out := make([]int, len(list))
	for i, g := range list {
		out[i] = int(g.ID)
	}
	return out
}

// mergeArticles ε-greedy 打散：
// - 先按 refresh_token + user_id 构造确定性随机种子，对两路各自进行稳定 shuffle（避免每页都是完全同一顺序造成机械感）；
// - 再按 interestQuota:1-interestQuota 的比例交替插入，兴趣项大概率在前；
// - 去重相同 ID。
func mergeArticles(interest, explore []*model.Article, interestQuota float64, token string, userID uint) []*model.Article {
	if interestQuota <= 0 {
		return dedupArticles(append(append([]*model.Article{}, explore...), interest...), token, userID)
	}
	if interestQuota >= 1 {
		return dedupArticles(append(append([]*model.Article{}, interest...), explore...), token, userID)
	}
	rngInterest := rngFromSeed(token, userID, 0xA1)
	rngExplore := rngFromSeed(token, userID, 0xB2)
	// 兴趣池：保留 top N%（高分标签/作者命中）不打散
	shuffleArticlesStable(interest, rngInterest, config.RecoInterestKeepTopRatio)
	// 探索池：保留 top N%（最热+最新）不打散，保证热门内容始终占据探索槽位的前面
	shuffleArticlesStable(explore, rngExplore, config.RecoExploreKeepTopRatio)

	// 比例: 每 K 条里 floor(K*interestQuota) 条来自 interest，其余 explore
	// 用 probabilistic insertion 更自然
	result := make([]*model.Article, 0, len(interest)+len(explore))
	seen := make(map[uint]struct{})
	i, e := 0, 0
	mergeRng := rngFromSeed(token, userID, 0xC3)
	for i < len(interest) || e < len(explore) {
		useInterest := mergeRng.Float64() < interestQuota
		if useInterest && i >= len(interest) {
			useInterest = false
		}
		if !useInterest && e >= len(explore) {
			useInterest = true
		}
		var pick *model.Article
		if useInterest {
			pick = interest[i]
			i++
		} else {
			pick = explore[e]
			e++
		}
		if _, dup := seen[pick.ID]; dup {
			continue
		}
		seen[pick.ID] = struct{}{}
		result = append(result, pick)
	}
	return result
}

func mergeGoods(interest, explore []*model.Good, interestQuota float64, token string, userID uint) []*model.Good {
	if interestQuota <= 0 {
		return dedupGoods(append(append([]*model.Good{}, explore...), interest...))
	}
	if interestQuota >= 1 {
		return dedupGoods(append(append([]*model.Good{}, interest...), explore...))
	}
	rngInterest := rngFromSeed(token, userID, 0xA1)
	rngExplore := rngFromSeed(token, userID, 0xB2)
	shuffleGoodsStable(interest, rngInterest, config.RecoInterestKeepTopRatio)
	shuffleGoodsStable(explore, rngExplore, config.RecoExploreKeepTopRatio)

	result := make([]*model.Good, 0, len(interest)+len(explore))
	seen := make(map[uint]struct{})
	i, e := 0, 0
	mergeRng := rngFromSeed(token, userID, 0xC3)
	for i < len(interest) || e < len(explore) {
		useInterest := mergeRng.Float64() < interestQuota
		if useInterest && i >= len(interest) {
			useInterest = false
		}
		if !useInterest && e >= len(explore) {
			useInterest = true
		}
		var pick *model.Good
		if useInterest {
			pick = interest[i]
			i++
		} else {
			pick = explore[e]
			e++
		}
		if _, dup := seen[pick.ID]; dup {
			continue
		}
		seen[pick.ID] = struct{}{}
		result = append(result, pick)
	}
	return result
}

func dedupArticles(list []*model.Article, token string, userID uint) []*model.Article {
	seen := make(map[uint]struct{}, len(list))
	out := make([]*model.Article, 0, len(list))
	for _, a := range list {
		if a == nil {
			continue
		}
		if _, dup := seen[a.ID]; dup {
			continue
		}
		seen[a.ID] = struct{}{}
		out = append(out, a)
	}
	shuffleArticlesStable(out, rngFromSeed(token, userID, 0xD4), config.RecoInterestKeepTopRatio)
	return out
}

func dedupGoods(list []*model.Good) []*model.Good {
	seen := make(map[uint]struct{}, len(list))
	out := make([]*model.Good, 0, len(list))
	for _, g := range list {
		if g == nil {
			continue
		}
		if _, dup := seen[g.ID]; dup {
			continue
		}
		seen[g.ID] = struct{}{}
		out = append(out, g)
	}
	return out
}

// rngFromSeed 由 (token, userID, salt) 生成确定性随机源
func rngFromSeed(token string, userID uint, salt byte) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(token))
	h.Write([]byte{salt})
	var uidBuf [8]byte
	u := uint64(userID)
	for i := 0; i < 8; i++ {
		uidBuf[i] = byte(u >> (8 * i))
	}
	h.Write(uidBuf[:])
	seed := int64(h.Sum64())
	return rand.New(rand.NewSource(seed))
}

// shuffleArticlesStable 前 keepRatio 比例的元素保留原分数排序，其余打散。
// 兴趣池传较小值（如 0.1），让最匹配的先露面；
// 探索池传较大值（如 0.3），让最热门的始终占据探索槽位前面。
func shuffleArticlesStable(list []*model.Article, r *rand.Rand, keepRatio float64) {
	if len(list) < 2 {
		return
	}
	if keepRatio < 0 {
		keepRatio = 0
	} else if keepRatio > 1 {
		keepRatio = 1
	}
	keep := int(float64(len(list)) * keepRatio)
	if keep >= len(list) {
		return
	}
	rest := list[keep:]
	r.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
}

func shuffleGoodsStable(list []*model.Good, r *rand.Rand, keepRatio float64) {
	if len(list) < 2 {
		return
	}
	if keepRatio < 0 {
		keepRatio = 0
	} else if keepRatio > 1 {
		keepRatio = 1
	}
	keep := int(float64(len(list)) * keepRatio)
	if keep >= len(list) {
		return
	}
	rest := list[keep:]
	r.Shuffle(len(rest), func(i, j int) { rest[i], rest[j] = rest[j], rest[i] })
}

// 以下两个 helper 供管理/调试使用，暂无外部调用也保留便于排查 ----------

// DebugFormatProfile 将画像格式化为人类可读串（调试/日志用）
func (p *UserProfile) DebugFormatProfile() string {
	if p == nil {
		return ""
	}
	top := p.TopTags
	if len(top) > 5 {
		top = top[:5]
	}
	return fmt.Sprintf("uid=%d cold=%v tags=%v authors=%d@%v", p.UserID, p.IsColdStart, top, len(p.TopAuthors), p.TopAuthors)
}

// Encode 画像编码为短哈希（便于在 API 返回 debug 字段时带上，不泄露细节）
func (p *UserProfile) Encode() string {
	if p == nil {
		return ""
	}
	sort.Strings(p.TopTags)
	sort.Ints(p.TopAuthors)
	h := fnv.New64a()
	for _, t := range p.TopTags {
		h.Write([]byte(t))
		h.Write([]byte{'\x00'})
	}
	for _, a := range p.TopAuthors {
		h.Write([]byte(fmt.Sprintf("%d ", a)))
	}
	var buf [8]byte
	v := h.Sum64()
	for i := 0; i < 8; i++ {
		buf[i] = byte(v >> (8 * i))
	}
	return hex.EncodeToString(buf[:])
}
