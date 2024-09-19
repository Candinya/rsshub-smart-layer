package translate

type Provider interface {
	Translate(string, string, bool) (*string, error)
}
