package usage

import "context"

// Adapter wraps a Store and returns usage values as int64.
type Adapter struct {
	Store Store
}

func NewAdapter(store Store) *Adapter {
	return &Adapter{Store: store}
}

func (a *Adapter) GetUsage(ctx context.Context, organizationID string, metric string) (int64, error) {
	counter, err := a.Store.GetUsage(ctx, organizationID, metric)
	if err != nil {
		return 0, err
	}
	if counter == nil {
		return 0, nil
	}
	return counter.Value, nil
}
