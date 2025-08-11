package migrations

type User struct {

	ID        int
	FirstName string
	LastName  string
	Email     string
	Password  string
	Role      string

}

var users = []User{
	{

		ID:        1,
		FirstName: "Leonardo",
		LastName:  "Herrera",
		Email:     "leonardoherrerac10@gmail.com",
		Password:  "supersecret",
		Role:      "super_admin",

	},
}

func GetUserByEmail(email string) *User {
	for i, u := range users {
		if u.Email == email {
			return &users[i]
		}
	}
	return nil
}
