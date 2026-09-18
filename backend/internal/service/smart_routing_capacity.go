package service

import "context"

func routingCapacity(ctx context.Context, concurrency *ConcurrencyService, accounts []Account) (float64, bool) {
	if concurrency == nil || len(accounts) == 0 {
		return 0, false
	}
	pool := make([]AccountWithConcurrency, 0, len(accounts))
	for _, a := range accounts {
		pool = append(pool, AccountWithConcurrency{ID: a.ID, MaxConcurrency: a.Concurrency})
	}
	loads, err := concurrency.GetAccountsLoadBatch(ctx, pool)
	if err != nil {
		return 0, false
	}
	free := 0.0
	for _, a := range accounts {
		load, ok := loads[a.ID]
		if !ok || load == nil {
			return 0, false
		}
		free += 1 - clamp01(float64(load.LoadRate)/100)
	}
	return free / float64(len(accounts)), true
}

func (s *GatewayService) SmartRoutingCapacity(ctx context.Context, accounts []Account) (float64, bool) {
	if s == nil {
		return 0, false
	}
	return routingCapacity(ctx, s.concurrencyService, accounts)
}

func (s *OpenAIGatewayService) SmartRoutingCapacity(ctx context.Context, accounts []Account) (float64, bool) {
	if s == nil {
		return 0, false
	}
	return routingCapacity(ctx, s.concurrencyService, accounts)
}
