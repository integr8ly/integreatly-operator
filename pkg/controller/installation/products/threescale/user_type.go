package threescale

type Users struct {
	Users []*User `json:"users"`
}

type User struct {
	UserDetails UserDetails `json:"user"`
}

type UserDetails struct {
	Id       int    `json:"id"`
	State    string `json:"state"`
	Role     string `json:"role"`
	Username string `json:"username"`
	Email    string `json:"email"`
}
