package wikifier

type clearBlock struct {
	*parserBlock
}

func newClearBlock(name string, b *parserBlock) block {
	return &clearBlock{parserBlock: b}
}
