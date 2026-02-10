package types

// ParamMapping mapeia param da req para campo nos dados
type ParamMapping struct {
	Name   string `json:"name"`
	MapsTo string `json:"maps_to"`
}

// Response para status e body
type Response struct {
	Status int         `json:"status"`
	Body   interface{} `json:"body,omitempty"`
}
