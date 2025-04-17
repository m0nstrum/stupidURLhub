package model

import (
	"errors"
	"time"
)

const (
	MaxContentSize = 1024 * 1024 // 1MB
	MaxTagsCount   = 10
	MaxTagLength   = 50
)

var (
	ErrContentTooLarge = errors.New("содержимое пасты слишком большое")
	ErrTooManyTags     = errors.New("слишком много тегов")
	ErrTagTooLong      = errors.New("тег слишком длинный")
	ErrEmptyContent    = errors.New("содержимое пасты не может быть пустым")
)

type Paste struct {
	ID         string    `gorm:"primaryKey"` // uuid v7
	Slug       string    `gorm:"uniqueIndex;size:50;not null"`
	Content    string    `gorm:"type:text;not null"`
	EditToken  string    `gorm:"size:100;not null"` // page admin token
	Tags       []string  `gorm:"type:jsonb;default:'[]'"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime"`
	ViewCount  int       `gorm:"default:0"`
	LastViewed *time.Time
	Expires    *time.Time
}

func (p *Paste) Validate() error {
	if len(p.Content) == 0 {
		return ErrEmptyContent
	}

	if len(p.Content) > MaxContentSize {
		return ErrContentTooLarge
	}

	if len(p.Tags) > MaxTagsCount {
		return ErrTooManyTags
	}

	for _, tag := range p.Tags {
		if len(tag) > MaxTagLength {
			return ErrTagTooLong
		}
	}

	return nil
}

func (p *Paste) HasExpired() bool {
	return p.Expires != nil && time.Now().After(*p.Expires)
}
