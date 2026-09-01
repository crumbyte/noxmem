package pprofx

// Delta defines a type for storing two values of the same type, where the first
// value represents a current state and the second value contains the previous
// state. The generic type T defines an arbitrary type, while V represents
// comparable values of type T.
type Delta[T any, V int | int64 | uint | uint64] struct {
	prev    T
	current T
}

// Prev returns a previous value of type T stored within the Delta instance.
// nolint:ireturn
func (d *Delta[T, V]) Prev() T {
	return d.prev
}

// Current returns the current value of type T stored within the Delta instance.
// nolint:ireturn
func (d *Delta[T, V]) Current() T {
	return d.current
}

// Update updates the current and the previous values of type T within the Delta
// instance.
func (d *Delta[T, V]) Update(newVal T) {
	d.prev, d.current = d.current, newVal
}

// DiffPercent calculates a percent difference between two values of type V. The
// diff value includes a sign to indicate whether the previous value is greater
// than the current value.
func (d *Delta[T, V]) DiffPercent(prev, current V) float64 {
	if prev == current {
		return 0
	}

	sign := float64(1)
	minStat, maxStat := float64(prev), float64(current)

	if prev > current {
		sign = -1

		minStat, maxStat = maxStat, minStat
	}

	return (100 / maxStat) * (maxStat - minStat) * sign
}
