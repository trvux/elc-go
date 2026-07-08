package presentation

import (
	"time"

	"github.com/trvux/elc-go/internal/auth/domain"
)

type userResponse struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Name        string     `json:"name"`
	Phone       string     `json:"phone"`
	AvatarURL   string     `json:"avatar_url"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:          u.ID(),
		Username:    u.Username(),
		Email:       u.Email(),
		Name:        u.Name(),
		Phone:       u.Phone(),
		AvatarURL:   u.AvatarURL(),
		Role:        string(u.Role()),
		Status:      string(u.Status()),
		LastLoginAt: u.LastLoginAt(),
		CreatedAt:   u.CreatedAt(),
	}
}

func toUserResponseList(users []*domain.User) []userResponse {
	result := make([]userResponse, 0, len(users))
	for _, u := range users {
		result = append(result, toUserResponse(u))
	}
	return result
}

type loginRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

// accessTokenResponse also carries the raw refresh token in the body, not
// just as a Set-Cookie — the only intended caller is a trusted first-party
// BFF (elc-tem's Server Actions calling this API server-to-server), which
// needs the raw value to set its own session cookie on its own domain
// rather than relying on this response's Set-Cookie ever reaching a browser
// directly. A public/untrusted client shouldn't be calling this API at all;
// see ARCHITECTURE.md's note about RequireRole/RequirePermission not yet
// covering every module.
type accessTokenResponse struct {
	User         userResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
}

type forgotPasswordRequest struct {
	Email string `json:"email"`
}

type resetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type acceptInviteRequest struct {
	Token    string `json:"token"`
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
}

type createInviteRequest struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}

// updateUserRequest fields are pointers so the handler can tell "not
// provided" (nil) apart from "explicitly cleared" — not that role/status
// have a meaningful empty value, but this keeps a future third field from
// silently becoming mandatory in every request body.
type updateUserRequest struct {
	Role   *string `json:"role"`
	Status *string `json:"status"`
}

// updateProfileRequest fields are pointers for the same "not provided" vs
// "explicitly cleared" reason as updateUserRequest.
type updateProfileRequest struct {
	Name      *string `json:"name"`
	Email     *string `json:"email"`
	AvatarURL *string `json:"avatar_url"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type inviteResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	ExpiresAt time.Time `json:"expires_at"`
}

func toInviteResponse(t *domain.VerificationToken) inviteResponse {
	return inviteResponse{
		ID:        t.ID(),
		Email:     t.Email(),
		Role:      string(t.Role()),
		ExpiresAt: t.ExpiresAt(),
	}
}

type messageResponse struct {
	Message string `json:"message"`
}
