package meta

type Meta byte

func (m Meta) Byte() byte {
	return byte(m)
}

const (
	ObjectMeta Meta = 1
	EventMeta  Meta = 2
)
