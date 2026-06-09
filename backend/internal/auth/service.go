package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"

	"ddz/backend/internal/profile"
)

var ErrInvalidAuthConfig = errors.New("invalid auth config")

const AccountTypeGuest = "guest"

// User 表示认证系统中的当前用户信息。
type User struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	AccountType string `json:"account_type"`
}

// Profile 表示当前用户的大厅侧基础资料。
type Profile struct {
	Level       int `json:"level"`
	CoinBalance int `json:"coin_balance"`
	TotalGames  int `json:"total_games"`
	Wins        int `json:"wins"`
}

// Identity 表示通过认证后的用户身份快照。
type Identity struct {
	User    User    `json:"user"`
	Profile Profile `json:"profile"`
}

// GuestLoginInput 表示游客登录请求参数。
type GuestLoginInput struct {
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
}

// GuestLoginResult 表示游客登录成功后的返回结果。
type GuestLoginResult struct {
	User        User   `json:"user"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// Service 负责游客身份生成、资料初始化和 token 签发。
type Service struct {
	jwt            *JWTManager
	profileService *profile.Service
	userSeq        atomic.Uint64
}

// NewService 创建认证服务。
func NewService(jwt *JWTManager, profileService *profile.Service) *Service {
	return &Service{
		jwt:            jwt,
		profileService: profileService,
	}
}

// GuestLogin 创建游客用户并签发 access token。
func (s *Service) GuestLogin(input GuestLoginInput) (GuestLoginResult, error) {
	if s.jwt == nil {
		return GuestLoginResult{}, ErrInvalidAuthConfig
	}

	seq := s.userSeq.Add(1)
	displayName := strings.TrimSpace(input.DisplayName)
	if displayName == "" {
		displayName = formatGuestDisplayName(seq)
	}

	user := User{
		ID:          formatUserID(seq),
		DisplayName: displayName,
		AvatarURL:   strings.TrimSpace(input.AvatarURL),
		AccountType: AccountTypeGuest,
	}

	if s.profileService != nil {
		if _, err := s.profileService.EnsureProfile(context.Background(), user.ID); err != nil {
			return GuestLoginResult{}, err
		}
	}

	token, claims, err := s.jwt.Issue(user)
	if err != nil {
		return GuestLoginResult{}, err
	}

	return GuestLoginResult{
		User:        user,
		AccessToken: token,
		ExpiresIn:   claims.ExpiresAt - claims.IssuedAt,
	}, nil
}

// IdentityFromClaims 根据 JWT claims 还原用户身份信息。
func IdentityFromClaims(claims Claims, userProfile Profile) Identity {
	return Identity{
		User: User{
			ID:          claims.Subject,
			DisplayName: claims.DisplayName,
			AvatarURL:   claims.AvatarURL,
			AccountType: claims.AccountType,
		},
		Profile: userProfile,
	}
}

// DefaultProfile 返回未持久化用户的默认资料。
func DefaultProfile() Profile {
	return Profile{
		Level:       1,
		CoinBalance: 10000,
		TotalGames:  0,
		Wins:        0,
	}
}

func formatUserID(seq uint64) string {
	return fmt.Sprintf("u_%06d", seq)
}

func formatGuestDisplayName(seq uint64) string {
	return fmt.Sprintf("Guest%06d", seq)
}
