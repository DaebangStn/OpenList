package handles

import (
	"strconv"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

// viewHistoryKeep is how many files each user's history holds.
const viewHistoryKeep = 1000

type ViewHistoryReq struct {
	Path string `json:"path" form:"path"`
}

// viewHistoryPath cleans a path as the web UI shows it (relative to the
// user's base path) and rejects ".." escapes.
func viewHistoryPath(user *model.User, path string) (string, error) {
	if _, err := user.JoinPath(path); err != nil {
		return "", err
	}
	return utils.FixAndCleanPath(path), nil
}

func AddViewHistory(c *gin.Context) {
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	var req ViewHistoryReq
	if err := c.ShouldBind(&req); err != nil || req.Path == "" {
		common.ErrorStrResp(c, "path is required", 400)
		return
	}
	path, err := viewHistoryPath(user, req.Path)
	if err != nil {
		common.ErrorResp(c, err, 403)
		return
	}
	if path == "/" {
		common.ErrorStrResp(c, "path is required", 400)
		return
	}
	if err := db.RecordViewHistory(user.ID, path, time.Now(), viewHistoryKeep); err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}

func ListViewHistory(c *gin.Context) {
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit <= 0 || limit > viewHistoryKeep {
		limit = viewHistoryKeep
	}
	rows, err := db.ListViewHistory(user.ID, limit)
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c, rows)
}

// DeleteViewHistory removes req.Path, or clears the history when it is empty.
func DeleteViewHistory(c *gin.Context) {
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	var req ViewHistoryReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorStrResp(c, "request invalid", 400)
		return
	}
	path := ""
	if req.Path != "" {
		var err error
		if path, err = viewHistoryPath(user, req.Path); err != nil {
			common.ErrorResp(c, err, 403)
			return
		}
	}
	if err := db.DeleteViewHistory(user.ID, path); err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}
