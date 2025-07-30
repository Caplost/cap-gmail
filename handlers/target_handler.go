package handlers

import (
	"gmail-forwarding-system/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TargetHandler struct {
	db *gorm.DB
}

// NewTargetHandler 创建转发对象处理器
func NewTargetHandler(db *gorm.DB) *TargetHandler {
	return &TargetHandler{db: db}
}

type CreateTargetRequest struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Keywords string `json:"keywords"`
}

type UpdateTargetRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email" binding:"omitempty,email"`
	Keywords string `json:"keywords"`
	IsActive *bool  `json:"is_active"`
}

// GetTargets 获取所有转发对象
func (th *TargetHandler) GetTargets(c *gin.Context) {
	targets, err := models.GetForwardingTargets(th.db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取转发对象失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  targets,
		"count": len(targets),
	})
}

// GetTarget 获取单个转发对象
func (th *TargetHandler) GetTarget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的ID格式",
		})
		return
	}

	target, err := models.GetForwardingTargetByID(th.db, uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "转发对象不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取转发对象失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": target,
	})
}

// CreateTarget 创建转发对象
func (th *TargetHandler) CreateTarget(c *gin.Context) {
	var req CreateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"details": err.Error(),
		})
		return
	}

	target := &models.ForwardingTarget{
		Name:     req.Name,
		Email:    req.Email,
		Keywords: req.Keywords,
		IsActive: true,
	}

	if err := models.CreateForwardingTarget(th.db, target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "创建转发对象失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "转发对象创建成功",
		"data":    target,
	})
}

// UpdateTarget 更新转发对象
func (th *TargetHandler) UpdateTarget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的ID格式",
		})
		return
	}

	target, err := models.GetForwardingTargetByID(th.db, uint(id))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "转发对象不存在",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取转发对象失败",
			"details": err.Error(),
		})
		return
	}

	var req UpdateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "请求参数错误",
			"details": err.Error(),
		})
		return
	}

	// 更新字段
	if req.Name != "" {
		target.Name = req.Name
	}
	if req.Email != "" {
		target.Email = req.Email
	}
	if req.Keywords != "" {
		target.Keywords = req.Keywords
	}
	if req.IsActive != nil {
		target.IsActive = *req.IsActive
	}

	if err := models.UpdateForwardingTarget(th.db, target); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "更新转发对象失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "转发对象更新成功",
		"data":    target,
	})
}

// DeleteTarget 删除转发对象（软删除）
func (th *TargetHandler) DeleteTarget(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "无效的ID格式",
		})
		return
	}

	if err := models.DeleteForwardingTarget(th.db, uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "删除转发对象失败",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "转发对象删除成功",
	})
}
