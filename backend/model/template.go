package model

import (
	"context"

	"math"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ContainerTemplates struct {
	Id          int64 `gorm:"primaryKey"`
	Name        string
	Config_json string
	CreatedAt   int64 `gorm:"autoCreateTime"`
	UpdatedAt   int64 `gorm:"autoUpdateTime"`
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
	log.Debugln("ListTemplate")
	db, err := getConn()
	if err != nil {
		return nil, nil, err
	}

	pageinfo := NewPageInfo(ctx, db, perPage, nextPage)
	// pageinfo := new(PageInfo)

	// perPage = pageinfo.GetPerPage(perPage)

	// var totalRows int64

	// var containerTemplates []ContainerTemplates
	// db.WithContext(ctx).Model(&containerTemplates).Count(&totalRows)
	// pageinfo.TotalRows = totalRows
	// totalRows := pageinfo.GetTotalRows(ctx, db)

	// totalPages := int64(math.Ceil(float64(totalRows) / float64(perPage)))
	// pageinfo.TotalPages = totalPages
	// totalPages := pageinfo.GetTotalPages(totalRows, perPage)

	// if nextPage == 0 {
	// 	nextPage = 1
	// }
	// if nextPage > totalPages {
	// 	nextPage = totalPages
	// }
	// pageinfo.CurPage = nextPage
	// nextPage = pageinfo.GetCurPage(nextPage, totalPages)

	// offsetnum := (nextPage - 1) * perPage
	offsetnum := pageinfo.Getoffset()
	var containerTemplates []ContainerTemplates
	if result := db.WithContext(ctx).Offset(int(offsetnum)).Limit(int(pageinfo.PerPage)).Find(&containerTemplates); result.Error != nil {
		log.Errorf("db list template %v", result.Error)
		return nil, nil, translateError(result.Error)
	}

	return pageinfo, containerTemplates, nil
}

func CreateTemplate(ctx context.Context, id int64, name string, configbyte []byte) (int64, error) {
	log.Debugln("CreateTemplate")
	db, err := getConn()
	if err != nil {
		log.Debugln(err)
		return -1, err
	}

	templateInfo := ContainerTemplates{
		Id:          id,
		Name:        name,
		Config_json: string(configbyte),
	}
	if result := db.WithContext(ctx).Create(&templateInfo); result.Error != nil {
		log.Errorf("db create template %v", result.Error)
		return -1, translateError(result.Error)
	}

	return templateInfo.Id, nil
}

func UpdateTemplate(ctx context.Context, id int64, name string, configbyte []byte) error {
	db, err := getConn()
	if err != nil {
		return err
	}

	// templateInfo := ContainerTemplates{}

	if result := db.WithContext(ctx).Model(&ContainerTemplates{}).Where("id=?", id).Updates(
		map[string]interface{}{
			"name":        name,
			"config_json": string(configbyte),
		},
	); result.Error != nil {
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
