package base

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ResponseJson(c *gin.Context, code int, message string, data interface{}) {
	var result = make(map[string]interface{})
	result["code"] = code
	result["message"] = message
	result["data"] = data
	c.JSON(http.StatusOK, result)
}
