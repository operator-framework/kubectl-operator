package action

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type creator interface {
	Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
}

type deleter interface {
	Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error
}

type getter interface {
	Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error
}
