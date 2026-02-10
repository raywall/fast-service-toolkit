/*
Package easyrepo fornece uma abstração genérica para o padrão Service-Repository
utilizando Amazon DynamoDB.

O objetivo deste pacote é reduzir o boilerplate em microserviços Go, entregando:
  - Validação de entrada automática via struct tags (validator/v10).
  - Operações CRUD padronizadas com suporte a Generics.
  - Integração simplificada com o toolkit dyndb.

Exemplo de uso:

	type User struct {
		ID    string `validate:"required"`
		Email string `validate:"required,email"`
	}

	service, _ := easyrepo.NewService[User](dynamoClient, tableConfig)
	err := service.Create(ctx, &User{ID: "1", Email: "test@example.com"})
*/
package easyrepo
