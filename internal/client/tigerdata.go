package client

// TODO
type TigerData struct{}

func NewTigerData() (*TigerData, error) {
	return &TigerData{}, nil
}

// TODO probably we can simplify the interface and ping and smoke test in the constructor
func (t *TigerData) Ping() error {
	return nil
}

func (t *TigerData) Query(query string) (*Response, error) {
	return &Response{Duration: 1}, nil
}
