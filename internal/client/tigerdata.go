package client

// TODO
type TigerData struct{}

func NewTigerData() *TigerData {
	return &TigerData{}
}

func (t *TigerData) Ping() error {
	return nil
}

func (t *TigerData) Query(query string) (*Response, error) {
	return &Response{Duration: 1}, nil
}
