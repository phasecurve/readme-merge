package remote

type CompositeReader struct {
	local  localReader
	remote *Resolver
}

type localReader interface {
	ReadFile(path string, ref string) (string, error)
}

func NewCompositeReader(local localReader, cacheBaseDir string) *CompositeReader {
	return &CompositeReader{
		local:  local,
		remote: NewResolver(cacheBaseDir),
	}
}

func (c *CompositeReader) ReadFile(path string, ref string) (string, error) {
	if IsCrossRepo(path) {
		return c.remote.ReadFile(path, ref)
	}
	return c.local.ReadFile(path, ref)
}
