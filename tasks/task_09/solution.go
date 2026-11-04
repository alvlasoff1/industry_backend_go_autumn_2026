package main

import (
	"context"
	"errors"
	"sync"
)

type Result[R any] struct {
	Value R
	Err   error
}

var ErrInvalidWorkers = errors.New("workers must be positive")

func ParallelMap[T any, R any](ctx context.Context, workers int, in []T, fn func(context.Context, T) (R, error)) ([]Result[R], error) {
	if workers <= 0 {
		return nil, ErrInvalidWorkers
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result := make([]Result[R], len(in))
	type element struct {
		i int
		v T
	}
	elements := make(chan element)
	var wg sync.WaitGroup
	worker := func() {
		for el := range elements {
			r, err := fn(ctx, el.v)
			if err != nil {
				result[el.i] = Result[R]{Err: err}
				continue
			}
			result[el.i] = Result[R]{Value: r}
		}
	}
	for i := 0; i < workers; i++ {
		wg.Go(worker)
	}
	for i, v := range in {
		select {
		case elements <- element{i, v}:
		case <-ctx.Done():
			close(elements)
			wg.Wait()
			return nil, ctx.Err()
		}
	}
	close(elements)
	wg.Wait()
	return result, nil
}
