package db

import (
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/pkg/errors"
	"gorm.io/gorm/clause"
)

// GetUIState returns the user's value for key, and false when none is stored.
func GetUIState(userID uint, key string) (string, bool, error) {
	var rows []model.UIState
	err := db.Where(&model.UIState{UserID: userID, Key: key}).Limit(1).Find(&rows).Error
	if err != nil {
		return "", false, errors.WithStack(err)
	}
	if len(rows) == 0 {
		return "", false, nil
	}
	return rows[0].Value, true, nil
}

// SetUIState stores value under key for the user, replacing any old value.
func SetUIState(userID uint, key, value string) error {
	row := model.UIState{UserID: userID, Key: key, Value: value}
	return errors.WithStack(db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
	}).Create(&row).Error)
}
