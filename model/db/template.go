package db

import (
	"context"
	"errors"
	"math"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	pb "scmc/rpc/pb/container"
)

type ContainerTemplates struct {
	Id         int64 `gorm:"primaryKey"`
	Name       string
	NodeId     int64
	ConfigJSON *pb.ContainerConfigs `gorm:"serializer:json"`
	CreatedAt  int64                `gorm:"autoCreateTime"`
	UpdatedAt  int64                `gorm:"autoUpdateTime"`
}

type PageInfo struct {
	PerPage    int64
	CurPage    int64
	TotalRows  int64
	TotalPages int64
}

func NewPageInfo(ctx context.Context, db *gorm.DB, perPage, nextPage int64) *PageInfo {
	var pageinfo PageInfo
	perPage = pageinfo.getPerPage(perPage)

	totalRows := pageinfo.getTotalRows(ctx, db)

	totalPages := pageinfo.getTotalPages(totalRows, perPage)

	pageinfo.getCurPage(nextPage, totalPages)

	return &pageinfo
}

func (p *PageInfo) getPerPage(perPage int64) int64 {
	if perPage == 0 {
		perPage = 10
	}

	p.PerPage = perPage
	return perPage
}

func (p *PageInfo) getCurPage(nextPage, totalPages int64) (curPage int64) {
	curPage = nextPage
	if curPage == 0 {
		curPage = 1
	}
	if curPage > totalPages {
		curPage = totalPages
	}
	p.CurPage = curPage
	return curPage
}

func (p *PageInfo) getTotalRows(ctx context.Context, db *gorm.DB) (totalRows int64) {
	var containerTemplates ContainerTemplates
	if result := db.WithContext(ctx).Model(&containerTemplates).Count(&totalRows); result.Error != nil {
		log.Warnln(result.Error)
		return 0
	}
	p.TotalRows = totalRows
	return totalRows
}

func (p *PageInfo) getTotalPages(totalRows, perPage int64) (totalPages int64) {
	totalPages = int64(math.Ceil(float64(totalRows) / float64(perPage)))
	p.TotalPages = totalPages
	return totalPages
}

func (p *PageInfo) Getoffset() int64 {
	return (p.CurPage - 1) * p.PerPage
}

func ListTemplate(ctx context.Context, perPage int64, nextPage int64) (*PageInfo, []ContainerTemplates, error) {
	db, err := getConn()
	if err != nil {
		return nil, nil, err
	}

	pageinfo := NewPageInfo(ctx, db, perPage, nextPage)

	offsetnum := pageinfo.Getoffset()
	var containerTemplates []ContainerTemplates
	if result := db.WithContext(ctx).Offset(int(offsetnum)).Limit(int(pageinfo.PerPage)).Find(&containerTemplates); result.Error != nil {
		log.Errorf("db list template %v", result.Error)
		return nil, nil, translateError(result.Error)
	}

	return pageinfo, containerTemplates, nil
}

func CreateTemplate(ctx context.Context, data *ContainerTemplates) (int64, error) {
	if data == nil || data.Name == "" {
		return -1, errors.New("invalid argument")
	}

	db, err := getConn()
	if err != nil {
		log.Debugln(err)
		return -1, err
	}

	if result := db.WithContext(ctx).Create(data); result.Error != nil {
		log.Errorf("db create template %v", result.Error)
		return -1, translateError(result.Error)
	}

	return data.Id, nil
}

func UpdateTemplate(ctx context.Context, data *ContainerTemplates) error {
	if data == nil || data.Name == "" {
		return errors.New("invalid argument")
	}

	db, err := getConn()
	if err != nil {
		return err
	}

	if result := db.Save(data); result.Error != nil {
		log.Errorf("db update template %v", result.Error)
		return translateError(result.Error)
	}

	return nil
}

func RemoveTemplate(ctx context.Context, ids []int64) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	var containerTemplates ContainerTemplates

	if result := db.WithContext(ctx).Delete(&containerTemplates, ids); result.Error != nil {
		log.Errorf("db delete template %v", result.Error)
		return translateError(result.Error)
	}

	return nil
}

func FindTemplate(id int64) (*ContainerTemplates, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var data ContainerTemplates

	if err := db.First(&data, "id = ?", id).Error; err != nil {
		log.Warnf("FindTemplate id=%v err=%v", id, err)
		return nil, translateError(err)
	}

	return &data, nil
}

func TemplateCount() (int64, error) {
	db, err := getConn()
	if err != nil {
		return 0, err
	}

	var c int64
	if err := db.Model(&ContainerTemplates{}).Count(&c).Error; err != nil {
		return 0, err
	}

	return c, nil
}
