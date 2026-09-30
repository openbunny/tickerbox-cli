// SPDX-License-Identifier: MIT

package concurrent

import "sync"

func Run(limit int, tasks ...func()) {
	if len(tasks) == 0 {
		return
	}
	if limit <= 0 || limit > len(tasks) {
		limit = len(tasks)
	}

	sem := make(chan struct{}, limit)
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Add(1)
		sem <- struct{}{}
		go func(task func()) {
			defer wg.Done()
			defer func() { <-sem }()
			task()
		}(task)
	}
	wg.Wait()
}
