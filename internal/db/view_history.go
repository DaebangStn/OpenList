package db

import (
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

// RecordViewHistory moves path to the top of the user's history and keeps at
// most keep rows for that user, dropping the oldest ones.
func RecordViewHistory(userID uint, path string, at time.Time, keep int) error {
	return errors.WithStack(db.Transaction(func(tx *gorm.DB) error {
		// Find rather than First: a first visit is the common case and First
		// logs every miss as an error.
		var rows []model.ViewHistory
		err := tx.Where(&model.ViewHistory{UserID: userID, Path: path}).Limit(1).Find(&rows).Error
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			err = tx.Create(&model.ViewHistory{UserID: userID, Path: path, ViewedAt: at, Count: 1}).Error
		} else {
			rows[0].ViewedAt = at
			rows[0].Count++
			err = tx.Save(&rows[0]).Error
		}
		if err != nil {
			return err
		}
		if keep <= 0 {
			return nil
		}
		// OFFSET without LIMIT is not portable, and a history holds at most
		// keep+1 rows here, so trim in Go.
		var ids []uint
		err = tx.Model(&model.ViewHistory{}).
			Where(&model.ViewHistory{UserID: userID}).
			Order(columnName("viewed_at")+" DESC").Order(columnName("id")+" DESC").
			Pluck("id", &ids).Error
		if err != nil || len(ids) <= keep {
			return err
		}
		return tx.Delete(&model.ViewHistory{}, ids[keep:]).Error
	}))
}

// ListViewHistory returns the user's history, most recent first.
func ListViewHistory(userID uint, limit int) ([]model.ViewHistory, error) {
	var rows []model.ViewHistory
	q := db.Where(&model.ViewHistory{UserID: userID}).
		Order(columnName("viewed_at") + " DESC").Order(columnName("id") + " DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return rows, nil
}

// DeleteViewHistory removes one path from the user's history, or the whole
// history when path is empty.
func DeleteViewHistory(userID uint, path string) error {
	q := db.Where(&model.ViewHistory{UserID: userID})
	if path != "" {
		q = q.Where(columnName("path")+" = ?", path)
	}
	return errors.WithStack(q.Delete(&model.ViewHistory{}).Error)
}
