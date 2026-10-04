package handles

import (
	"path"
	"strconv"
	"strings"

	"github.com/OpenListTeam/OpenList/v4/internal/conf"
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
	"github.com/OpenListTeam/OpenList/v4/internal/pathindex"
	"github.com/OpenListTeam/OpenList/v4/server/common"
	"github.com/gin-gonic/gin"
)

type FindPathResp struct {
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
}

// localRoots lists the enabled Local storages for the path index.
func localRoots() []pathindex.Root {
	var roots []pathindex.Root
	for _, d := range op.GetAllStorages() {
		st := d.GetStorage()
		if st.Disabled || d.Config().Name != "Local" {
			continue
		}
		rp, ok := d.GetAddition().(driver.IRootPath)
		if !ok || rp.GetRootPath() == "" {
			continue
		}
		roots = append(roots, pathindex.Root{Mount: st.MountPath, Dir: rp.GetRootPath()})
	}
	return roots
}

// FindPaths matches q against every path of the Local storages, so a
// fragment such as "park" finds paper/siggraph2027/inventory/park2025magnet.pdf.
func FindPaths(c *gin.Context) {
	user := c.Request.Context().Value(conf.UserKey).(*model.User)
	q := strings.TrimSpace(c.Query("q"))
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil || limit <= 0 || limit > 200 {
		limit = 50
	}
	out := []FindPathResp{}
	if len([]rune(q)) < 2 {
		common.SuccessResp(c, out)
		return
	}
	snap := pathindex.Get(localRoots)
	hits := snap.Search(q, limit, func(e pathindex.Entry) bool {
		node := model.SearchNode{Parent: path.Dir(e.Path), Name: path.Base(e.Path), IsDir: e.IsDir}
		return isSearchNodeAccessible(user, node, "", op.GetNearestMeta)
	})
	base := strings.TrimSuffix(user.BasePath, "/")
	for _, h := range hits {
		p := strings.TrimPrefix(h.Path, base)
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		out = append(out, FindPathResp{Path: p, IsDir: h.IsDir})
	}
	common.SuccessResp(c, out)
}

// WarmPathIndex builds the path index once the storages have loaded, so the
// first query after a restart does not wait for the walk.
func WarmPathIndex() {
	go func() {
		<-conf.StoragesLoadSignal()
		pathindex.Get(localRoots)
	}()
}
