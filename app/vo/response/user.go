package response

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
