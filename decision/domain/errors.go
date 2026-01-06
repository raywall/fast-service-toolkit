package domain

// ValidationError representa um tipo de erro customizado
type ValidationError struct {
	Message string `yaml:"msg"`
	Code    int    `yaml:"code"`
}

func (v *ValidationError) Error() string {
	return v.Message
}
