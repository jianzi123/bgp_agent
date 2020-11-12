package procinfo

import (
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
)

const GITINFOFILE = "/code_version"

func GitInfo(ctx *gin.Context) {
	buff, err := ioutil.ReadFile(GITINFOFILE)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	ctx.JSON(http.StatusOK, string(buff))
}
