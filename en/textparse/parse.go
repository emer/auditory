package textparse

type parse struct {
	Table map[string]string
}

func (p *parse) Init() {
	p.Table = make(map[string]string)
	p.Table["emergent"] = "/c // /0 # /w /l i./*m_er_r.j_uh_n_t # // /c"

}

// Lookup returns the pregenerated phonetic version of the string argument e.g. "emergent" returns "/c // /0 # /w /l i./*m_er_r.j_uh_n_t # // /c"
func (p *parse) Lookup(s string) string {
	return p.Table[s]
}
