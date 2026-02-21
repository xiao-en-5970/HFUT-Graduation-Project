package response

// UserProfile 用户公开身份信息（供他人查看，仅非删用户）
type UserProfile struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	Background  string `json:"background"`
	FollowCount int    `json:"follow_count"`
	FansCount   int    `json:"fans_count"`
}

type UserInfo struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	Background  string `json:"background"`
	FollowCount int    `json:"follow_count"`
	FansCount   int    `json:"fans_count"`
	Role        int16  `json:"role"`
	Status      int16  `json:"status"`
	SchoolID    uint   `json:"school_id"`
}
