// examples/users/models/user.go
package models

type User struct {
	UserID    string `dynamodbav:"userId"`
	Email     string `dynamodbav:"email"`
	Name      string `dynamodbav:"name"`
	Status    string `dynamodbav:"status"`
	CreatedAt int64  `dynamodbav:"createdAt"`
	ExpiresAt int64  `dynamodbav:"expiresAt,omitempty" ttl:"expiresAt"`
}
