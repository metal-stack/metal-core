package db

type View map[string]struct{}

func NewView(size int) View {
	return make(View, size)
}

func (v View) Add(item string) {
	v[item] = struct{}{}
}

func (v View) Remove(item string) {
	delete(v, item)
}

func (v View) Has(item string) bool {
	_, ok := v[item]
	return ok
}
