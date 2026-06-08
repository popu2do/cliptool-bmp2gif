package core

const (
	DefaultDelayMS = 500
	MinDelayMS     = 100
	MaxDelayMS     = 3000
	MinImages      = 2
)

type GifOptions struct {
	DelayMS int
}

func (o GifOptions) Normalized() GifOptions {
	if o.DelayMS == 0 {
		o.DelayMS = DefaultDelayMS
	}
	if o.DelayMS < MinDelayMS {
		o.DelayMS = MinDelayMS
	}
	if o.DelayMS > MaxDelayMS {
		o.DelayMS = MaxDelayMS
	}
	return o
}

func (o GifOptions) DelayUnits() int {
	normalized := o.Normalized()
	delayUnits := normalized.DelayMS / 10
	if delayUnits < 1 {
		return 1
	}
	return delayUnits
}
