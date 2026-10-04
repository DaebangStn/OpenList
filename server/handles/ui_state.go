package handles

import (
	"regexp"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/db"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

var uiStateKey = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// uiStateMaxBytes bounds one stored value; a tab layout is a few KB.
const uiStateMaxBytes = 1 << 20

type UIStateReq struct {
	Key   string `json:"key" form:"key"`
	Value string `json:"value"`
}

func GetUIState(c *gin.Context) {
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	key := c.Query("key")
	if !uiStateKey.MatchString(key) {
		common.ErrorStrResp(c, "invalid key", 400)
		return
	}
	value, ok, err := db.GetUIState(user.ID, key)
	if err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	if !ok {
		common.SuccessResp(c, nil)
		return
	}
	common.SuccessResp(c, value)
}

func SetUIState(c *gin.Context) {
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	var req UIStateReq
	if err := c.ShouldBind(&req); err != nil || !uiStateKey.MatchString(req.Key) {
		common.ErrorStrResp(c, "invalid key", 400)
		return
	}
	if len(req.Value) > uiStateMaxBytes {
		common.ErrorStrResp(c, "value too large", 413)
		return
	}
	if err := db.SetUIState(user.ID, req.Key, req.Value); err != nil {
		common.ErrorResp(c, err, 500, true)
		return
	}
	common.SuccessResp(c)
}
