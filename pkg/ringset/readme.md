a ringBuffer store unique elements, like a set
use the Key() of an element to differentiate it from others
so element must implement Keyer:
```
type Keyer interface {
	Key() string
}

```