package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
)

func entityOrNil[T any](entity *T, result *gorm.DB) (*T, error) {
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return entity, nil
}

type DBTX = *gorm.DB

type TxManager interface {
	RunInTx(ctx context.Context, fn func(tx DBTX) error) error
}

type GORMTxManager struct {
	db *gorm.DB
}

func NewTxManager(database *gorm.DB) *GORMTxManager {
	return &GORMTxManager{db: database}
}

func (manager *GORMTxManager) RunInTx(ctx context.Context, fn func(tx DBTX) error) error {
	return manager.db.WithContext(ctx).Transaction(fn)
}
