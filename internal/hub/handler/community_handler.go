package handler

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/gin-gonic/gin"
)

// deidentify 对文本进行脱敏处理，移除文件路径、URL、IP地址和API密钥
func deidentify(text string) string {
	reFilepath := regexp.MustCompile(`(?:[A-Za-z]:\\[\w\-. \\]+|\/[\w\-. \/]+)\.\w{1,6}`)
	text = reFilepath.ReplaceAllString(text, "[文件路径]")

	reURL := regexp.MustCompile(`https?:\/\/[^\s]+`)
	text = reURL.ReplaceAllString(text, "[URL]")

	reIP := regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	text = reIP.ReplaceAllString(text, "[IP地址]")

	reAPIKey := regexp.MustCompile(`(?:sk-[a-zA-Z0-9]+|Bearer\s+[a-zA-Z0-9\-_\.]+)`)
	text = reAPIKey.ReplaceAllString(text, "[密钥]")

	return text
}

// deidentifyMap 递归对 map 中的字符串字段进行脱敏
func deidentifyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = deidentify(val)
		case map[string]interface{}:
			result[k] = deidentifyMap(val)
		case []interface{}:
			deidentifySlice := make([]interface{}, len(val))
			for i, item := range val {
				switch itemVal := item.(type) {
				case string:
					deidentifySlice[i] = deidentify(itemVal)
				case map[string]interface{}:
					deidentifySlice[i] = deidentifyMap(itemVal)
				default:
					deidentifySlice[i] = item
				}
			}
			result[k] = deidentifySlice
		default:
			result[k] = v
		}
	}
	return result
}

type publishCommunityWorkerReq struct {
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Domain      string                 `json:"domain"`
	Scope       string                 `json:"scope"`
	Handbook    map[string]interface{} `json:"handbook"`
	Playbooks   map[string]interface{} `json:"playbooks"`
	Tags        []string               `json:"tags"`
	Deidentify  bool                   `json:"deidentify"`
}

// PublishWorker 发布社区 Worker
func (h *Handler) PublishWorker(c *gin.Context) {
	var req publishCommunityWorkerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	userID := c.GetInt64("user_id")

	handbook := req.Handbook
	playbooks := req.Playbooks
	if req.Deidentify {
		if handbook != nil {
			handbook = deidentifyMap(handbook)
		}
		if playbooks != nil {
			playbooks = deidentifyMap(playbooks)
		}
	}

	w, err := h.Svc.PublishWorker(c.Request.Context(), userID, req.Title, req.Description, req.Domain, req.Scope, handbook, playbooks, req.Tags)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": w})
}

// ListCommunityWorkers 获取社区 Worker 列表
func (h *Handler) ListCommunityWorkers(c *gin.Context) {
	search := c.Query("search")
	domain := c.Query("domain")
	sort := c.DefaultQuery("sort", "latest")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	list, total, err := h.Svc.ListCommunityWorkers(c.Request.Context(), search, domain, sort, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list, "total": total})
}

// GetCommunityWorker 获取单个社区 Worker 详情
func (h *Handler) GetCommunityWorker(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}
	w, err := h.Svc.GetCommunityWorker(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": w})
}

type installWorkerReq struct {
	BusinessCode string `json:"business_code"`
}

// InstallWorker 安装社区 Worker 到指定业务
func (h *Handler) InstallWorker(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid id"})
		return
	}
	var req installWorkerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	if err := h.Svc.InstallWorker(c.Request.Context(), id, req.BusinessCode); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// ListCommunityWorkerReviews 获取社区 Worker 的评论列表
func (h *Handler) ListCommunityWorkerReviews(c *gin.Context) {
	workerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid worker id"})
		return
	}
	list, err := h.Svc.GetCommunityWorkerReviews(c.Request.Context(), workerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": list})
}

type addCommunityWorkerReviewReq struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

// AddCommunityWorkerReview 添加社区 Worker 评论
func (h *Handler) AddCommunityWorkerReview(c *gin.Context) {
	workerID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "invalid worker id"})
		return
	}
	userID := c.GetInt64("user_id")
	var req addCommunityWorkerReviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": err.Error()})
		return
	}
	r, err := h.Svc.AddCommunityWorkerReview(c.Request.Context(), workerID, userID, req.Rating, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": r})
}
