package model

import (
	"math"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Paginator struct {
	PerPage    int64
	CurPage    int64
	TotalRows  int64
	TotalPages int64
}

func NewPaginator(db *gorm.DB, perPage, nextPage int64) *Paginator {
	var p Paginator
	perPage = p.getPerPage(perPage)
	totalRows := p.getTotalRows(db)
	totalPages := p.getTotalPages(totalRows, perPage)

	p.getCurPage(nextPage, totalPages)

	return &p
}

func (p *Paginator) getPerPage(perPage int64) int64 {
	if perPage == 0 {
		perPage = 10
	}

	p.PerPage = perPage
	return perPage
}

func (p *Paginator) getCurPage(nextPage, totalPages int64) (curPage int64) {
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

func (p *Paginator) getTotalRows(db *gorm.DB) (totalRows int64) {
	var containerTemplates ContainerTemplates
	if result := db.Model(&containerTemplates).Count(&totalRows); result.Error != nil {
		log.Warnln(result.Error)
		return 0
	}
	p.TotalRows = totalRows
	return totalRows
}

func (p *Paginator) getTotalPages(totalRows, perPage int64) (totalPages int64) {
	totalPages = int64(math.Ceil(float64(totalRows) / float64(perPage)))
	p.TotalPages = totalPages
	return totalPages
}

func (p *Paginator) getOffset() int64 {
	return (p.CurPage - 1) * p.PerPage
}

type Pager struct {
	PageSize   int64
	PageNo     int64
	TotalPages int64
}

// return fixed `pageSize`, `pageNo`, error
func PageQuery(pageSize, pageNo int64, model, condition, order, data interface{}) (*Pager, error) {
	db, err := getConn()
	if err != nil {
		return nil, err
	}

	var totalRows int64
	result := db.Model(model).Where(condition).Count(&totalRows)
	if result.Error != nil {
		return nil, result.Error
	}

	if pageSize <= 0 {
		pageSize = 10
	} else if pageSize > 50 {
		pageSize = 50
	}

	totalPages := int64(math.Ceil(float64(totalRows) / float64(pageSize)))
	if pageNo <= 0 {
		pageNo = 1
	}
	if pageNo > totalPages {
		pageNo = totalPages
	}

	offset := pageSize * (pageNo - 1)
	if result := db.Where(condition).Order(order).Offset(int(offset)).Limit(int(pageSize)).Find(data); result.Error != nil {
		log.Warnf("pageQuery err=%v", result.Error)
		return nil, translateError(result.Error)
	}
	return &Pager{
		PageSize:   pageSize,
		PageNo:     pageNo,
		TotalPages: totalPages,
	}, nil
}
