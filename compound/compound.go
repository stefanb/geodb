package compound

import "fmt"

type Prefix string

const (
	Object Prefix = "object_"
	Event  Prefix = "event_"
)

func ToObjectKey(uid string) []byte {
	return []byte(fmt.Sprintf("%s%s", Object, uid))
}
